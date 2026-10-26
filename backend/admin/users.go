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
	COALESCE(daily_search_limit, 0), created_at, updated_at, COALESCE(last_login_at, '')`

func scanUser(row rowScanner) (*User, error) {
	u := &User{}
	var mustChange int
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.Status,
		&mustChange, &u.DailySearchLimit, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
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
		list[i].SearchCountToday, _ = UserSearchCountToday(list[i].ID)
	}
	return list, total, nil
}

type UserUpdate struct {
	Username         *string `json:"username"`
	Email            *string `json:"email"`
	Role             *string `json:"role"`
	Status           *string `json:"status"`
	DailySearchLimit *int    `json:"daily_search_limit"`
	Password         *string `json:"password"`
	MustChangePass   *bool   `json:"must_change_password"`
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
		// 不允许把最后一个管理员降级。
		if u.Role == RoleAdmin && role != RoleAdmin {
			var admins int
			if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&admins); err != nil {
				return nil, err
			}
			if admins <= 1 {
				return nil, fmt.Errorf("不能降级最后一个管理员")
			}
		}
		if _, err := DB.Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, role, Now(), id); err != nil {
			return nil, err
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
	if upd.DailySearchLimit != nil {
		limit := *upd.DailySearchLimit
		if limit < 0 || limit > 100000 {
			return nil, fmt.Errorf("每日搜索上限不合法")
		}
		if _, err := DB.Exec(`UPDATE users SET daily_search_limit = ?, updated_at = ? WHERE id = ?`, limit, Now(), id); err != nil {
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
		var admins int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&admins); err != nil {
			return err
		}
		if admins <= 1 {
			return fmt.Errorf("不能删除最后一个管理员")
		}
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

func GetAPIToken(userID int64) (*APIToken, error) {
	t := &APIToken{}
	err := DB.QueryRow(
		`SELECT id, user_id, prefix, created_at, COALESCE(last_used_at,'') FROM api_tokens WHERE user_id = ?`, userID,
	).Scan(&t.ID, &t.UserID, &t.Prefix, &t.CreatedAt, &t.LastUsedAt)
	if isNoRows(err) {
		return nil, sql.ErrNoRows
	}
	return t, err
}

// ResetAPIToken 生成新令牌并返回明文；旧令牌立即失效。
func ResetAPIToken(userID int64) (*APIToken, string, error) {
	token := "ps_" + randomHex(24)
	hash := sha256Hex(token)
	prefix := token[:11]
	now := Now()
	_, err := DB.Exec(
		`INSERT INTO api_tokens (user_id, token_hash, prefix, created_at, last_used_at) VALUES (?, ?, ?, ?, '')
		 ON CONFLICT(user_id) DO UPDATE SET token_hash = excluded.token_hash, prefix = excluded.prefix,
		 created_at = excluded.created_at, last_used_at = ''`,
		userID, hash, prefix, now,
	)
	if err != nil {
		return nil, "", err
	}
	t, err := GetAPIToken(userID)
	return t, token, err
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
	_, _ = DB.Exec(`UPDATE api_tokens SET last_used_at = ? WHERE user_id = ?`, Now(), userID)
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

func EffectiveDailyLimit(u *User, s Settings) int {
	if u != nil && u.DailySearchLimit > 0 {
		return u.DailySearchLimit
	}
	return s.DefaultDailySearchLimit
}

func UserSearchCountToday(userID int64) (int, error) {
	if userID <= 0 {
		return 0, nil
	}
	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE user_id = ? AND created_at >= ?`, userID, todayStart()).Scan(&count)
	return count, err
}

func RecordSearch(u *User, source, keyword, ip string) {
	userID := int64(0)
	username := ""
	if u != nil {
		userID = u.ID
		username = u.Username
	}
	_, _ = DB.Exec(
		`INSERT INTO search_logs (user_id, username, source, keyword, ip, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, username, source, truncate(keyword, 200), ip, Now(),
	)
}

func ListSearchLogs(page, pageSize int, query, source string) ([]SearchLog, int, error) {
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
		`SELECT id, user_id, username, source, keyword, ip, created_at FROM search_logs`+where+
			` ORDER BY id DESC LIMIT ? OFFSET ?`, args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []SearchLog
	for rows.Next() {
		var l SearchLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Username, &l.Source, &l.Keyword, &l.IP, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, l)
	}
	return list, total, rows.Err()
}

func GetOverview() (*Overview, error) {
	o := &Overview{}
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&o.UsersTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE status = ?`, StatusActive).Scan(&o.UsersActive)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&o.AdminsTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE created_at >= ?`, todayStart()).Scan(&o.SearchesToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs`).Scan(&o.SearchesTotal)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM search_logs WHERE user_id = 0 AND created_at >= ?`, todayStart()).Scan(&o.AnonymousToday)
	_ = DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE expires_at > ?`, Now()).Scan(&o.SessionsActive)
	logs, _, err := ListSearchLogs(1, 10, "", "")
	if err != nil {
		return nil, err
	}
	o.RecentSearches = logs
	var adminHash string
	if err := DB.QueryRow(`SELECT password_hash FROM users WHERE username = ? COLLATE NOCASE`, DefaultAdminUsername).Scan(&adminHash); err == nil {
		o.DefaultAdminAlive = CheckPassword(adminHash, DefaultAdminPassword)
	}
	return o, nil
}

func todayStart() string {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
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
