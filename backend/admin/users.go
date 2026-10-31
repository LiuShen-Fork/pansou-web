package admin

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrUserNotFound   = errors.New("用户不存在")
	ErrUserExists     = errors.New("用户名或邮箱已存在")
	ErrInvalidLogin   = errors.New("用户名或密码错误")
	ErrUserDisabled   = errors.New("账号已被禁用")
	ErrSessionInvalid = errors.New("登录状态已失效")
)

const userColumns = `id, username, email, password_hash, role, status, must_change_password,
	COALESCE(daily_search_limit, 0), COALESCE(api_daily_search_limit, 0),
	COALESCE(web_rate_limit_per_min, 0), COALESCE(api_rate_limit_per_min, 0),
	created_at, updated_at, COALESCE(last_login_at, '')`

func scanUser(row rowScanner) (*User, error) {
	u := &User{}
	var mustChange int
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Status,
		&mustChange, &u.WebDailySearchLimit, &u.APIDailySearchLimit,
		&u.WebRateLimitPerMin, &u.APIRateLimitPerMin,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
		return nil, err
	}
	u.MustChangePassword = mustChange == 1
	return u, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}
	if len(email) > 254 {
		return fmt.Errorf("邮箱过长")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("邮箱格式不正确")
	}
	return nil
}

func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 2 || len(username) > 32 {
		return fmt.Errorf("用户名长度需为 2-32 个字符")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("用户名仅允许字母、数字、下划线和连字符")
	}
	switch strings.ToLower(username) {
	case "admin", "root", "system", "administrator":
		return fmt.Errorf("该用户名为系统保留")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 || len(password) > 128 {
		return fmt.Errorf("密码长度需为 8-128 位")
	}
	return nil
}

func CreateUser(username, email, password, role string) (*User, error) {
	username = strings.TrimSpace(username)
	if role == RoleAdmin {
		// 管理员允许沿用 admin 作为用户名（默认账号），仅校验字符集。
		if err := validateUsernameChars(username); err != nil {
			return nil, err
		}
	} else if err := ValidateUsername(username); err != nil {
		return nil, err
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	email = NormalizeEmail(email)
	if email != "" {
		if err := ValidateEmail(email); err != nil {
			return nil, err
		}
	}
	if role != RoleAdmin && role != RoleUser {
		role = RoleUser
	}
	if role == RoleAdmin {
		// 管理员全局唯一，只由初始化流程创建；接口层永远不能新增管理员。
		var admins int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&admins); err != nil {
			return nil, err
		}
		if admins > 0 {
			return nil, fmt.Errorf("管理员账号全局唯一，不能创建新的管理员")
		}
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	now := Now()
	res, err := DB.Exec(
		`INSERT INTO users (username, email, password_hash, role, status, must_change_password, daily_search_limit, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, 0, ?, ?)`,
		username, email, hash, role, StatusActive, now, now,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrUserExists
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return GetUserByID(id)
}

func validateUsernameChars(username string) error {
	username = strings.TrimSpace(username)
	if len(username) < 2 || len(username) > 32 {
		return fmt.Errorf("用户名长度需为 2-32 个字符")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("用户名仅允许字母、数字、下划线和连字符")
	}
	return nil
}

func GetUserByID(id int64) (*User, error) {
	row := DB.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if isNoRows(err) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func GetUserByUsername(username string) (*User, error) {
	row := DB.QueryRow(`SELECT `+userColumns+` FROM users WHERE username = ? COLLATE NOCASE`, strings.TrimSpace(username))
	u, err := scanUser(row)
	if isNoRows(err) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func GetUserByEmail(email string) (*User, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return nil, ErrUserNotFound
	}
	row := DB.QueryRow(`SELECT `+userColumns+` FROM users WHERE email = ? COLLATE NOCASE`, email)
	u, err := scanUser(row)
	if isNoRows(err) {
		return nil, ErrUserNotFound
	}
	return u, err
}

// FindUserByAccount 依次按用户名、邮箱查找。
func FindUserByAccount(account string) (*User, error) {
	account = strings.TrimSpace(account)
	if u, err := GetUserByUsername(account); err == nil {
		return u, nil
	}
	if strings.Contains(account, "@") {
		return GetUserByEmail(account)
	}
	return nil, ErrUserNotFound
}

func ListUsers(query, role, status string, page, pageSize int) ([]User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	conds := []string{}
	args := []any{}
	if q := strings.TrimSpace(query); q != "" {
		conds = append(conds, `(username LIKE ? OR email LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if role == RoleAdmin || role == RoleUser {
		conds = append(conds, `role = ?`)
		args = append(args, role)
	}
	if status == StatusActive || status == StatusDisabled {
		conds = append(conds, `status = ?`)
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = ` WHERE ` + strings.Join(conds, ` AND `)
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := DB.Query(`SELECT `+userColumns+` FROM users`+where+` ORDER BY id ASC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// 必须先关闭结果集再逐条查搜索次数，否则会占着连接等待第二条连接。
	rows.Close()
	for i := range list {
		list[i].WebSearchCountToday, _ = UserWebSearchCountToday(list[i].ID)
		list[i].APISearchCountToday, _ = UserAPISearchCountToday(list[i].ID)
	}
	return list, total, nil
}

type UserUpdate struct {
	Username            *string `json:"username"`
	Email               *string `json:"email"`
	Role                *string `json:"role"`
	Status              *string `json:"status"`
	WebDailySearchLimit *int    `json:"web_daily_search_limit"`
	APIDailySearchLimit *int    `json:"api_daily_search_limit"`
	WebRateLimitPerMin  *int    `json:"web_rate_limit_per_min"`
	APIRateLimitPerMin  *int    `json:"api_rate_limit_per_min"`
	Password            *string `json:"password"`
	MustChangePass      *bool   `json:"must_change_password"`
}

func UpdateUser(id int64, upd UserUpdate) (*User, error) {
	u, err := GetUserByID(id)
	if err != nil {
		return nil, err
	}
	if upd.Username != nil {
		name := strings.TrimSpace(*upd.Username)
		// 普通用户不能改成 admin/root 等保留名；管理员允许沿用 admin。
		var nameErr error
		if u.Role == RoleAdmin {
			nameErr = validateUsernameChars(name)
		} else {
			nameErr = ValidateUsername(name)
		}
		if nameErr != nil {
			return nil, nameErr
		}
		if _, err := DB.Exec(`UPDATE users SET username = ?, updated_at = ? WHERE id = ?`, name, Now(), id); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil, ErrUserExists
			}
			return nil, err
		}
	}
	if upd.Email != nil {
		email := NormalizeEmail(*upd.Email)
		if email != "" {
			if err := ValidateEmail(email); err != nil {
				return nil, err
			}
		}
		if _, err := DB.Exec(`UPDATE users SET email = ?, updated_at = ? WHERE id = ?`, email, Now(), id); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil, ErrUserExists
			}
			return nil, err
		}
	}
	if upd.Role != nil {
		role := *upd.Role
		if role != RoleAdmin && role != RoleUser {
			return nil, fmt.Errorf("角色只能是 admin 或 user")
		}
		// 管理员全局唯一且不可通过接口变更：既不能把普通用户提升为管理员，
		// 也不能降级唯一的管理员，避免权限扩散或后台被锁死。
		if role != u.Role {
			if role == RoleAdmin {
				return nil, fmt.Errorf("管理员账号全局唯一，不能提升其他用户为管理员")
			}
			return nil, fmt.Errorf("不能降级唯一的管理员")
		}
	}
	if upd.Status != nil {
		status := *upd.Status
		if status != StatusActive && status != StatusDisabled {
			return nil, fmt.Errorf("状态只能是 active 或 disabled")
		}
		if _, err := DB.Exec(`UPDATE users SET status = ?, updated_at = ? WHERE id = ?`, status, Now(), id); err != nil {
			return nil, err
		}
		if status == StatusDisabled {
			DeleteUserSessions(id)
		}
	}
	if upd.WebDailySearchLimit != nil {
		limit := *upd.WebDailySearchLimit
		if limit < 0 || limit > 100000 {
			return nil, fmt.Errorf("网页搜索每日上限不合法")
		}
		if _, err := DB.Exec(`UPDATE users SET daily_search_limit = ?, updated_at = ? WHERE id = ?`, limit, Now(), id); err != nil {
			return nil, err
		}
	}
	if upd.APIDailySearchLimit != nil {
		limit := *upd.APIDailySearchLimit
		if limit < 0 || limit > 100000 {
			return nil, fmt.Errorf("API 每日调用上限不合法")
		}
		if _, err := DB.Exec(`UPDATE users SET api_daily_search_limit = ?, updated_at = ? WHERE id = ?`, limit, Now(), id); err != nil {
			return nil, err
		}
	}
	if upd.WebRateLimitPerMin != nil {
		limit := *upd.WebRateLimitPerMin
		if limit < 0 || limit > 100000 {
			return nil, fmt.Errorf("网页搜索每分钟上限不合法")
		}
		if _, err := DB.Exec(`UPDATE users SET web_rate_limit_per_min = ?, updated_at = ? WHERE id = ?`, limit, Now(), id); err != nil {
			return nil, err
		}
	}
	if upd.APIRateLimitPerMin != nil {
		limit := *upd.APIRateLimitPerMin
		if limit < 0 || limit > 100000 {
			return nil, fmt.Errorf("API 每分钟上限不合法")
		}
		if _, err := DB.Exec(`UPDATE users SET api_rate_limit_per_min = ?, updated_at = ? WHERE id = ?`, limit, Now(), id); err != nil {
			return nil, err
		}
	}
	if upd.Password != nil {
		if err := ValidatePassword(*upd.Password); err != nil {
			return nil, err
		}
		hash, err := HashPassword(*upd.Password)
		if err != nil {
			return nil, err
		}
		mustChange := 0
		if upd.MustChangePass != nil && *upd.MustChangePass {
			mustChange = 1
		}
		if _, err := DB.Exec(`UPDATE users SET password_hash = ?, must_change_password = ?, updated_at = ? WHERE id = ?`, hash, mustChange, Now(), id); err != nil {
			return nil, err
		}
		DeleteUserSessions(id)
	} else if upd.MustChangePass != nil {
		if _, err := DB.Exec(`UPDATE users SET must_change_password = ?, updated_at = ? WHERE id = ?`, boolInt(*upd.MustChangePass), Now(), id); err != nil {
			return nil, err
		}
	}
	return GetUserByID(id)
}

func DeleteUser(id, currentUserID int64) error {
	if id == currentUserID {
		return fmt.Errorf("不能删除当前登录的账号")
	}
	u, err := GetUserByID(id)
	if err != nil {
		return err
	}
	if u.Role == RoleAdmin {
		return fmt.Errorf("管理员账号全局唯一，不能删除")
	}
	_, err = DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func UpdateUserPassword(id int64, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`UPDATE users SET password_hash = ?, must_change_password = 0, updated_at = ? WHERE id = ?`, hash, Now(), id)
	return err
}

func TouchLastLogin(id int64) {
	_, _ = DB.Exec(`UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`, Now(), Now(), id)
}

// ---------------- 会话 ----------------

func SessionTTLForSettings(s Settings) time.Duration {
	hours := s.SessionTTLHours
	if hours <= 0 {
		hours = 168
	}
	return time.Duration(hours) * time.Hour
}

func CreateSession(userID int64, ip, userAgent string, ttl time.Duration) (string, *Session, error) {
	token := randomHex(32)
	hash := sha256Hex(token)
	now := time.Now().UTC()
	expires := now.Add(ttl)
	id := randomHex(12)
	s := &Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expires.Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
		IP:        ip,
		UserAgent: truncate(userAgent, 200),
	}
	_, err := DB.Exec(
		`INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, ip, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, hash, s.ExpiresAt, s.CreatedAt, s.IP, s.UserAgent,
	)
	if err != nil {
		return "", nil, err
	}
	return token, s, nil
}

func GetSessionUser(token string) (*User, *Session, error) {
	if token == "" || len(token) > 256 {
		return nil, nil, ErrSessionInvalid
	}
	hash := sha256Hex(token)
	s := &Session{}
	err := DB.QueryRow(
		`SELECT id, user_id, expires_at, created_at, COALESCE(ip,''), COALESCE(user_agent,'') FROM sessions WHERE token_hash = ?`, hash,
	).Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt, &s.IP, &s.UserAgent)
	if isNoRows(err) {
		return nil, nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, nil, err
	}
	if expired(s.ExpiresAt) {
		_, _ = DB.Exec(`DELETE FROM sessions WHERE id = ?`, s.ID)
		return nil, nil, ErrSessionInvalid
	}
	u, err := GetUserByID(s.UserID)
	if err != nil {
		return nil, nil, ErrSessionInvalid
	}
	if !u.Active() {
		return nil, nil, ErrUserDisabled
	}
	return u, s, nil
}

func DeleteSession(id string) { _, _ = DB.Exec(`DELETE FROM sessions WHERE id = ?`, id) }
func DeleteUserSessions(userID int64) {
	_, _ = DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
}
func DeleteOtherSessions(userID int64, keepSessionID string) {
	_, _ = DB.Exec(`DELETE FROM sessions WHERE user_id = ? AND id <> ?`, userID, keepSessionID)
}

// ---------------- API 令牌 ----------------

// MaxAPITokensPerUser 单个用户最多可持有的令牌数量。
// 所有令牌共享同一账号的每日搜索配额。
const MaxAPITokensPerUser = 10

const defaultTokenName = "默认令牌"

func normalizeTokenName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultTokenName
	}
	runes := []rune(name)
	if len(runes) > 32 {
		name = string(runes[:32])
	}
	return name
}

func ListAPITokens(userID int64) ([]APIToken, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, name, prefix, created_at, COALESCE(last_used_at,'')
		 FROM api_tokens WHERE user_id = ? ORDER BY id ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]APIToken, 0, 4)
	for rows.Next() {
		var t APIToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Prefix, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// GetAPIToken 返回用户最早创建的令牌，保留给旧接口与概览使用。
func GetAPIToken(userID int64) (*APIToken, error) {
	items, err := ListAPITokens(userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return &items[0], nil
}

// CreateAPIToken 新增一个令牌并返回明文；明文只在此刻出现一次。
func CreateAPIToken(userID int64, name string) (*APIToken, string, error) {
	items, err := ListAPITokens(userID)
	if err != nil {
		return nil, "", err
	}
	if len(items) >= MaxAPITokensPerUser {
		return nil, "", fmt.Errorf("最多只能创建 %d 个 API 令牌", MaxAPITokensPerUser)
	}
	token := "ps_" + randomHex(24)
	hash := sha256Hex(token)
	prefix := token[:11]
	now := Now()
	res, err := DB.Exec(
		`INSERT INTO api_tokens (user_id, name, token_hash, prefix, created_at, last_used_at) VALUES (?, ?, ?, ?, ?, '')`,
		userID, normalizeTokenName(name), hash, prefix, now,
	)
	if err != nil {
		return nil, "", err
	}
	id, _ := res.LastInsertId()
	return &APIToken{ID: id, UserID: userID, Name: normalizeTokenName(name), Prefix: prefix, CreatedAt: now}, token, nil
}

// DeleteAPIToken 删除属于该用户的指定令牌。
func DeleteAPIToken(userID, id int64) error {
	res, err := DB.Exec(`DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ResetAPIToken 撤销全部旧令牌并新建一个，返回明文。
func ResetAPIToken(userID int64) (*APIToken, string, error) {
	if _, err := DB.Exec(`DELETE FROM api_tokens WHERE user_id = ?`, userID); err != nil {
		return nil, "", err
	}
	return CreateAPIToken(userID, defaultTokenName)
}

func GetUserByAPIToken(token string) (*User, error) {
	if !strings.HasPrefix(token, "ps_") || len(token) > 128 {
		return nil, ErrSessionInvalid
	}
	var userID int64
	err := DB.QueryRow(`SELECT user_id FROM api_tokens WHERE token_hash = ?`, sha256Hex(token)).Scan(&userID)
	if isNoRows(err) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}
	_, _ = DB.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE token_hash = ?`, Now(), sha256Hex(token))
	u, err := GetUserByID(userID)
	if err != nil || !u.Active() {
		return nil, ErrSessionInvalid
	}
	return u, nil
}

// ---------------- 登录失败限流 ----------------

func RecordLoginAttempt(ip, account string, success bool) {
	_, _ = DB.Exec(`INSERT INTO login_attempts (ip, account, success, created_at) VALUES (?, ?, ?, ?)`,
		ip, account, boolInt(success), Now())
}

func CountRecentFailures(ip, account string, window time.Duration) (int, int) {
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)
	var byIP, byAccount int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE ip = ? AND success = 0 AND created_at >= ?`, ip, since).Scan(&byIP)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE account = ? COLLATE NOCASE AND success = 0 AND created_at >= ?`, account, since).Scan(&byAccount)
	return byIP, byAccount
}

// ---------------- 搜索日志与配额 ----------------

// EffectiveWebDailyLimit 返回用户网页搜索的每日额度。
func EffectiveWebDailyLimit(u *User, s Settings) int {
	if u != nil && u.WebDailySearchLimit > 0 {
		return u.WebDailySearchLimit
	}
	return s.DefaultWebDailyLimit
}

// EffectiveAPIDailyLimit 返回用户 API 调用的每日额度。
func EffectiveAPIDailyLimit(u *User, s Settings) int {
	if u != nil && u.APIDailySearchLimit > 0 {
		return u.APIDailySearchLimit
	}
	return s.DefaultAPIDailyLimit
}

// EffectiveWebRateLimit 返回用户网页搜索的每分钟上限。
func EffectiveWebRateLimit(u *User, s Settings) int {
	if u != nil && u.WebRateLimitPerMin > 0 {
		return u.WebRateLimitPerMin
	}
	return s.DefaultWebRateLimit
}

// EffectiveAPIRateLimit 返回用户 API 调用的每分钟上限。
func EffectiveAPIRateLimit(u *User, s Settings) int {
	if u != nil && u.APIRateLimitPerMin > 0 {
		return u.APIRateLimitPerMin
	}
	return s.DefaultAPIRateLimit
}

// UserWebSearchCountToday 统计登录用户通过网页入口完成的搜索次数。
func UserWebSearchCountToday(userID int64) (int, error) {
	return userSearchCountToday(userID, "web")
}

// UserAPISearchCountToday 只统计通过 /api/search 的调用，用于独立 API 额度。
func UserAPISearchCountToday(userID int64) (int, error) {
	return userSearchCountToday(userID, "api")
}

func userSearchCountToday(userID int64, source string) (int, error) {
	if userID <= 0 {
		return 0, nil
	}
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM search_logs WHERE user_id = ? AND source = ? AND created_at >= ?`,
		userID, source, todayStart(),
	).Scan(&count)
	return count, err
}

// UserRecentSearchCount 统计用户在最近 window 内某一来源的搜索次数，用于分钟级限频。
func UserRecentSearchCount(userID int64, source string, window time.Duration) (int, error) {
	if userID <= 0 {
		return 0, nil
	}
	since := time.Now().UTC().Add(-window).Format(time.RFC3339)
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM search_logs WHERE user_id = ? AND source = ? AND created_at >= ?`,
		userID, source, since,
	).Scan(&count)
	return count, err
}

// AnonymousIPSearchCountToday 统计同一 IP 当天的匿名搜索次数。
func AnonymousIPSearchCountToday(ip string) (int, error) {
	if strings.TrimSpace(ip) == "" {
		return 0, nil
	}
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM search_logs WHERE user_id = 0 AND ip = ? AND created_at >= ?`,
		ip, todayStart(),
	).Scan(&count)
	return count, err
}

// AnonymousGlobalSearchCountToday 统计全站当天的匿名搜索总次数。
func AnonymousGlobalSearchCountToday() (int, error) {
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM search_logs WHERE user_id = 0 AND created_at >= ?`,
		todayStart(),
	).Scan(&count)
	return count, err
}

// StartSearchLog 在配额校验通过后写入一条待完成任务，返回日志 ID。
// 搜索处理完成后由 FinishSearchLog 补上关键词、耗时、结果数与状态。
func StartSearchLog(u *User, source, keyword, ip, userAgent string) int64 {
	if DB == nil {
		return 0
	}
	userID := int64(0)
	username := ""
	if u != nil {
		userID = u.ID
		username = u.Username
	}
	geo := LookupIP(ip)
	res, err := DB.Exec(
		`INSERT INTO search_logs (user_id, username, source, keyword, ip, user_agent, country, region, city, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
		userID, username, source, truncate(keyword, 200), ip, truncate(userAgent, 300),
		geo.Country, geo.Region, geo.City, Now(),
	)
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	return id
}

// FinishSearchLog 补充一次搜索任务的最终状态。日志 ID 为 0 时安全忽略。
func FinishSearchLog(id int64, keyword string, resultCount int, durationMS int64, status, errMsg string) {
	if id <= 0 || DB == nil {
		return
	}
	update := `UPDATE search_logs SET result_count = ?, duration_ms = ?, status = ?, error = ?`
	args := []any{resultCount, durationMS, status, truncate(errMsg, 500)}
	if kw := strings.TrimSpace(keyword); kw != "" {
		update += `, keyword = ?`
		args = append(args, truncate(kw, 200))
	}
	update += ` WHERE id = ?`
	args = append(args, id)
	_, _ = DB.Exec(update, args...)
}

func RecordSearch(u *User, source, keyword, ip string) int64 {
	return StartSearchLog(u, source, keyword, ip, "")
}

func ListSearchLogs(page, pageSize int, query, source, status string) ([]SearchLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	conds := []string{}
	args := []any{}
	if q := strings.TrimSpace(query); q != "" {
		conds = append(conds, `(username LIKE ? OR keyword LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	if source = strings.TrimSpace(source); source != "" {
		conds = append(conds, `source = ?`)
		args = append(args, source)
	}
	if status = strings.TrimSpace(status); status != "" {
		conds = append(conds, `status = ?`)
		args = append(args, status)
	}
	where := ""
	if len(conds) > 0 {
		where = ` WHERE ` + strings.Join(conds, ` AND `)
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM search_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := DB.Query(
		`SELECT id, user_id, username, source, keyword, ip, user_agent, country, region, city,
		        duration_ms, result_count, status, error, created_at
		 FROM search_logs`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []SearchLog
	for rows.Next() {
		var l SearchLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Source, &l.Keyword, &l.IP,
			&l.UserAgent, &l.Country, &l.Region, &l.City,
			&l.DurationMS, &l.ResultCount, &l.Status, &l.Error, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, l)
	}
	return list, total, rows.Err()
}

// RecordAudit 写入全局审计日志。调用方保证 event 为稳定的 snake_case 标识。
// 审计失败不影响主流程。
func RecordAudit(u *User, event, target, detail, ip, userAgent string) {
	if DB == nil {
		return
	}
	userID := int64(0)
	username := ""
	if u != nil {
		userID = u.ID
		username = u.Username
	}
	geo := LookupIP(ip)
	_, _ = DB.Exec(
		`INSERT INTO audit_logs (user_id, username, event, target, detail, ip, country, region, city, user_agent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, username, truncate(event, 80), truncate(target, 200), truncate(detail, 1000),
		ip, geo.Country, geo.Region, geo.City, truncate(userAgent, 300), Now(),
	)
}

func ListAuditLogs(page, pageSize int, query, event string) ([]AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	conds := []string{}
	args := []any{}
	if q := strings.TrimSpace(query); q != "" {
		conds = append(conds, `(username LIKE ? OR target LIKE ? OR detail LIKE ? OR ip LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	if event = strings.TrimSpace(event); event != "" {
		conds = append(conds, `event = ?`)
		args = append(args, event)
	}
	where := ""
	if len(conds) > 0 {
		where = ` WHERE ` + strings.Join(conds, ` AND `)
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM audit_logs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := DB.Query(
		`SELECT id, user_id, username, event, target, detail, ip, country, region, city, user_agent, created_at
		 FROM audit_logs`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Event, &l.Target, &l.Detail,
			&l.IP, &l.Country, &l.Region, &l.City, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, l)
	}
	return list, total, rows.Err()
}

func topNameCounts(query string, args ...any) []NameCount {
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := make([]NameCount, 0, 5)
	for rows.Next() {
		var item NameCount
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return out
		}
		out = append(out, item)
	}
	return out
}

// ListUserSearchLogs 返回某个用户最近的搜索任务，供个人中心展示。
func ListUserSearchLogs(userID int64, limit int) ([]SearchLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := DB.Query(
		`SELECT id, user_id, username, source, keyword, ip, user_agent, country, region, city,
		        duration_ms, result_count, status, error, created_at
		 FROM search_logs WHERE user_id = ? ORDER BY id DESC LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]SearchLog, 0, limit)
	for rows.Next() {
		var l SearchLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Source, &l.Keyword, &l.IP,
			&l.UserAgent, &l.Country, &l.Region, &l.City,
			&l.DurationMS, &l.ResultCount, &l.Status, &l.Error, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func GetOverview() (*Overview, error) {
	o := &Overview{}
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&o.UsersTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE status = ?`, StatusActive).Scan(&o.UsersActive)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&o.AdminsTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE created_at >= ?`, todayStart()).Scan(&o.SearchesToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs`).Scan(&o.SearchesTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE user_id = 0 AND created_at >= ?`, todayStart()).Scan(&o.AnonymousToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE user_id > 0 AND source = 'web' AND created_at >= ?`, todayStart()).Scan(&o.WebUserToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE user_id > 0 AND source = 'api' AND created_at >= ?`, todayStart()).Scan(&o.APIUserToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE event = 'quota_exceeded' AND created_at >= ?`, todayStart()).Scan(&o.QuotaExceededToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE created_at >= ?`, todayStart()).Scan(&o.AuditToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE expires_at > ?`, Now()).Scan(&o.SessionsActive)
	o.TopCountries = topNameCounts(`SELECT country AS name, COUNT(*) AS count FROM search_logs
		WHERE country <> '' AND created_at >= ? GROUP BY country ORDER BY count DESC LIMIT 5`, todayStart())
	o.TopIPs = topNameCounts(`SELECT ip AS name, COUNT(*) AS count FROM search_logs
		WHERE user_id = 0 AND ip <> '' AND created_at >= ? GROUP BY ip ORDER BY count DESC LIMIT 5`, todayStart())
	logs, _, err := ListSearchLogs(1, 10, "", "", "")
	if err != nil {
		return nil, err
	}
	o.RecentSearches = logs
	audit, _, _ := ListAuditLogs(1, 8, "", "")
	o.RecentAudit = audit
	var adminHash string
	if err := DB.QueryRow(`SELECT password_hash FROM users WHERE username = ? COLLATE NOCASE`, DefaultAdminUsername).Scan(&adminHash); err == nil {
		o.DefaultAdminAlive = CheckPassword(adminHash, DefaultAdminPassword)
	}
	return o, nil
}

// 每日额度按服务器本地时区的自然日计算（Docker 镜像默认 Asia/Shanghai），
// 但数据库里的 created_at 是 UTC，因此边界要换算成 UTC 再比较。
func localDayStart(now time.Time) time.Time {
	local := now.Local()
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).UTC()
}

func nextLocalDayStart(now time.Time) time.Time {
	local := now.Local()
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).
		AddDate(0, 0, 1).UTC()
}

func todayStart() string {
	return localDayStart(time.Now()).Format(time.RFC3339)
}

func nextDayStart() string {
	return nextLocalDayStart(time.Now()).Format(time.RFC3339)
}

// ---------------- 工具 ----------------

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败属于不可恢复的环境问题，退回时间戳保证进程可用。
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func timePlus(seconds int) string {
	return time.Now().UTC().Add(time.Duration(seconds) * time.Second).Format(time.RFC3339)
}

func timeMinus(seconds int) string {
	return time.Now().UTC().Add(-time.Duration(seconds) * time.Second).Format(time.RFC3339)
}

func expired(rfc3339 string) bool {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(t)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// 按 rune 截断，避免把 UTF-8 字符切成半个字节。
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
