package admin

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"pansou/util/json"
)

var safeProviderSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,31}$`)

const (
	settingKeyAuth  = "auth"
	settingKeyEmail = "email"
)

func GetSetting(key string, dest any) error {
	var raw string
	if err := DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&raw); err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func SetSetting(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = DB.Exec(
		`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, string(raw), Now(),
	)
	return err
}

var settingsCache struct {
	sync.RWMutex
	value   Settings
	expires time.Time
}

func InvalidateSettingsCache() {
	settingsCache.Lock()
	settingsCache.expires = time.Time{}
	settingsCache.Unlock()
}

func LoadSettings() Settings {
	now := time.Now()
	settingsCache.RLock()
	if now.Before(settingsCache.expires) {
		v := settingsCache.value
		settingsCache.RUnlock()
		return v
	}
	settingsCache.RUnlock()

	s := DefaultSettings()
	if err := GetSetting(settingKeyAuth, &s); err != nil {
		s = DefaultSettings()
	}
	// 旧数据缺失的字段补默认值，避免因为新增字段变成零值。
	if s.SessionTTLHours <= 0 {
		s.SessionTTLHours = 168
	}
	if s.AnonymousWebSearchLimit <= 0 {
		s.AnonymousWebSearchLimit = 20
	}
	// 兼容旧默认值：仍为英文默认名时自动迁移到中文默认名。
	if s.SiteName == "" || s.SiteName == "Qingyu PanSou" {
		s.SiteName = DefaultSettings().SiteName
	}
	if strings.TrimSpace(s.SiteSlogan) == "" {
		s.SiteSlogan = DefaultSettings().SiteSlogan
	}
	if strings.TrimSpace(s.SiteIcon) == "" {
		s.SiteIcon = DefaultSettings().SiteIcon
	}

	settingsCache.Lock()
	settingsCache.value = s
	settingsCache.expires = now.Add(3 * time.Second)
	settingsCache.Unlock()
	return s
}

func SaveSettings(s Settings) error {
	if s.SessionTTLHours <= 0 {
		s.SessionTTLHours = 168
	}
	if s.AnonymousWebSearchLimit <= 0 {
		s.AnonymousWebSearchLimit = 20
	}
	if err := SetSetting(settingKeyAuth, s); err != nil {
		return err
	}
	InvalidateSettingsCache()
	return nil
}

func LoadEmailSettings() EmailSettings {
	e := DefaultEmailSettings()
	if err := GetSetting(settingKeyEmail, &e); err != nil {
		return DefaultEmailSettings()
	}
	if e.Port <= 0 {
		e.Port = 587
	}
	return e
}

func SaveEmailSettings(e EmailSettings) error {
	return SetSetting(settingKeyEmail, e)
}

// ---------------- OAuth provider ----------------

func ListOAuthProviders(onlyEnabled bool) ([]OAuthProvider, error) {
	query := `SELECT id, slug, name, enabled, client_id, client_secret, auth_url, token_url, userinfo_url, scopes, created_at, updated_at
	          FROM oauth_providers`
	if onlyEnabled {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY id ASC`
	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []OAuthProvider
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func GetOAuthProviderBySlug(slug string) (*OAuthProvider, error) {
	row := DB.QueryRow(`SELECT id, slug, name, enabled, client_id, client_secret, auth_url, token_url, userinfo_url, scopes, created_at, updated_at
	                    FROM oauth_providers WHERE slug = ? COLLATE NOCASE`, strings.TrimSpace(slug))
	p, err := scanProvider(row)
	if isNoRows(err) {
		return nil, fmt.Errorf("未找到 OAuth 服务商")
	}
	return p, err
}

// GetPrimaryOAuthProvider 返回唯一被管理的 OAuth2 服务商（ID 最小的一条）。
// 历史版本可能留下多条记录，界面只展示和编辑这一条。
func GetPrimaryOAuthProvider() (*OAuthProvider, error) {
	row := DB.QueryRow(`SELECT id, slug, name, enabled, client_id, client_secret, auth_url, token_url, userinfo_url, scopes, created_at, updated_at
	                    FROM oauth_providers ORDER BY id ASC LIMIT 1`)
	p, err := scanProvider(row)
	if isNoRows(err) {
		return nil, sql.ErrNoRows
	}
	return p, err
}

// UpsertPrimaryOAuthProvider 新建或更新唯一的 OAuth2 服务商。
// 忽略调用方传入的 ID 与 Slug，从根源上保证管理端只有一个入口。
func UpsertPrimaryOAuthProvider(p *OAuthProvider) (*OAuthProvider, error) {
	primary, err := GetPrimaryOAuthProvider()
	if err == nil {
		p.ID = primary.ID
		p.Slug = primary.Slug
		if strings.TrimSpace(p.ClientSecret) == "" {
			p.ClientSecret = primary.ClientSecret
		}
		if strings.TrimSpace(p.Name) == "" {
			p.Name = primary.Name
		}
	} else if !isNoRows(err) {
		return nil, err
	} else {
		p.ID = 0
		p.Slug = "oauth"
	}

	for attempt := 0; attempt < 3; attempt++ {
		if err := SaveOAuthProvider(p); err == nil {
			return GetOAuthProviderByID(p.ID)
		} else if !strings.Contains(err.Error(), "已存在") {
			return nil, err
		}
		// 极端情况下表中已有同名 slug 的历史记录，生成一个可读且唯一的标识。
		p.Slug = "oauth-" + randomHex(3)
	}
	return nil, fmt.Errorf("保存 OAuth 服务商失败：标识冲突")
}

func GetOAuthProviderByID(id int64) (*OAuthProvider, error) {
	row := DB.QueryRow(`SELECT id, slug, name, enabled, client_id, client_secret, auth_url, token_url, userinfo_url, scopes, created_at, updated_at
	                    FROM oauth_providers WHERE id = ?`, id)
	p, err := scanProvider(row)
	if isNoRows(err) {
		return nil, fmt.Errorf("未找到 OAuth 服务商")
	}
	return p, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvider(row rowScanner) (*OAuthProvider, error) {
	p := &OAuthProvider{}
	var enabled int
	if err := row.Scan(&p.ID, &p.Slug, &p.Name, &enabled, &p.ClientID, &p.ClientSecret,
		&p.AuthURL, &p.TokenURL, &p.UserInfoURL, &p.Scopes, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Enabled = enabled == 1
	return p, nil
}

func SaveOAuthProvider(p *OAuthProvider) error {
	p.Slug = strings.ToLower(strings.TrimSpace(p.Slug))
	p.Name = strings.TrimSpace(p.Name)
	if p.Slug == "" || p.Name == "" {
		return fmt.Errorf("服务商标识和名称不能为空")
	}
	if !safeProviderSlug.MatchString(p.Slug) {
		return fmt.Errorf("服务商标识仅允许小写字母、数字、下划线和连字符")
	}
	now := Now()
	if p.ID == 0 {
		res, err := DB.Exec(
			`INSERT INTO oauth_providers (slug, name, enabled, client_id, client_secret, auth_url, token_url, userinfo_url, scopes, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Slug, p.Name, boolInt(p.Enabled), p.ClientID, p.ClientSecret, p.AuthURL, p.TokenURL, p.UserInfoURL, p.Scopes, now, now,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return fmt.Errorf("服务商标识已存在")
			}
			return err
		}
		p.ID, _ = res.LastInsertId()
		p.CreatedAt = now
		p.UpdatedAt = now
		return nil
	}
	_, err := DB.Exec(
		`UPDATE oauth_providers SET slug = ?, name = ?, enabled = ?, client_id = ?, client_secret = ?,
		 auth_url = ?, token_url = ?, userinfo_url = ?, scopes = ?, updated_at = ? WHERE id = ?`,
		p.Slug, p.Name, boolInt(p.Enabled), p.ClientID, p.ClientSecret, p.AuthURL, p.TokenURL, p.UserInfoURL, p.Scopes, now, p.ID,
	)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("服务商标识已存在")
	}
	p.UpdatedAt = now
	return err
}

func DeleteOAuthProvider(id int64) error {
	_, err := DB.Exec(`DELETE FROM oauth_providers WHERE id = ?`, id)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ---------------- OAuth account binding ----------------

func GetOAuthAccount(providerID int64, subject string) (int64, error) {
	var userID int64
	err := DB.QueryRow(`SELECT user_id FROM oauth_accounts WHERE provider_id = ? AND subject = ?`, providerID, subject).Scan(&userID)
	if isNoRows(err) {
		return 0, sql.ErrNoRows
	}
	return userID, err
}

func BindOAuthAccount(providerID int64, subject string, userID int64, email string) error {
	_, err := DB.Exec(
		`INSERT INTO oauth_accounts (provider_id, subject, user_id, email, created_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(provider_id, subject) DO UPDATE SET user_id = excluded.user_id, email = excluded.email`,
		providerID, subject, userID, email, Now(),
	)
	return err
}

func ListUserOAuthAccounts(userID int64) ([]OAuthAccount, error) {
	rows, err := DB.Query(
		`SELECT a.provider_id, p.slug, p.name, a.email, a.created_at
		 FROM oauth_accounts a JOIN oauth_providers p ON p.id = a.provider_id
		 WHERE a.user_id = ? ORDER BY a.id ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []OAuthAccount
	for rows.Next() {
		var a OAuthAccount
		if err := rows.Scan(&a.ProviderID, &a.ProviderSlug, &a.ProviderName, &a.Email, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func UnbindOAuthAccount(userID, providerID int64) error {
	// 至少保留一种可登录方式：没有密码的用户不能解绑最后一个 OAuth 账号。
	var hash string
	if err := DB.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash); err != nil {
		return err
	}
	if hash == "" {
		var count int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM oauth_accounts WHERE user_id = ?`, userID).Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("这是账号最后一种登录方式，请先设置密码")
		}
	}
	_, err := DB.Exec(`DELETE FROM oauth_accounts WHERE user_id = ? AND provider_id = ?`, userID, providerID)
	return err
}

// ---------------- OAuth state ----------------

type oauthState struct {
	State      string
	ProviderID int64
	Mode       string
	UserID     int64
	ExpiresAt  string
	CreatedAt  string
}

func CreateOAuthState(providerID int64, mode string, userID int64) (string, error) {
	state := randomHex(24)
	now := Now()
	_, err := DB.Exec(
		`INSERT INTO oauth_states (state, provider_id, mode, user_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		state, providerID, mode, userID, timePlus(15*60), now,
	)
	if err != nil {
		return "", err
	}
	return state, nil
}

func TakeOAuthState(state string) (*oauthState, error) {
	st := &oauthState{}
	err := DB.QueryRow(
		`SELECT state, provider_id, mode, user_id, expires_at, created_at FROM oauth_states WHERE state = ?`, state,
	).Scan(&st.State, &st.ProviderID, &st.Mode, &st.UserID, &st.ExpiresAt, &st.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("OAuth 状态无效或已过期")
		}
		return nil, err
	}
	_, _ = DB.Exec(`DELETE FROM oauth_states WHERE state = ?`, state)
	if expired(st.ExpiresAt) {
		return nil, fmt.Errorf("OAuth 状态已过期")
	}
	return st, nil
}

func CleanupExpiredAuthData() {
	now := Now()
	_, _ = DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	_, _ = DB.Exec(`DELETE FROM oauth_states WHERE expires_at < ?`, now)
	_, _ = DB.Exec(`DELETE FROM email_codes WHERE expires_at < ?`, now)
	_, _ = DB.Exec(`DELETE FROM login_attempts WHERE created_at < ?`, timeMinus(7*24*60*60))
	// 搜索日志保留 180 天，避免长期运行把 SQLite 撑大；更早的统计不再保留。
	_, _ = DB.Exec(`DELETE FROM search_logs WHERE created_at < ?`, timeMinus(180*24*60*60))
}
