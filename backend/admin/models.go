package admin

import "time"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"

	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123456"

	CookieName = "pansou_session"

	SessionTTL = 7 * 24 * time.Hour
)

// User 是后台账户体系的用户。PasswordHash 永不序列化到 API。
type User struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	MustChangePassword bool   `json:"must_change_password"`
	DailySearchLimit   int    `json:"daily_search_limit"`
	// APIDailySearchLimit 仅统计 /api/search 的调用；0 表示跟随每日搜索额度。
	APIDailySearchLimit int    `json:"api_daily_search_limit"`
	APISearchCountToday int    `json:"api_search_count_today,omitempty"`
	SearchCountToday    int    `json:"search_count_today,omitempty"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
	LastLoginAt         string `json:"last_login_at,omitempty"`

	PasswordHash string `json:"-"`
}

func (u *User) IsAdmin() bool { return u != nil && u.Role == RoleAdmin }
func (u *User) Active() bool  { return u != nil && u.Status == StatusActive }

type Session struct {
	ID        string `json:"id"`
	UserID    int64  `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type APIToken struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

// Settings 是账户/搜索开关。所有开关都可以在管理后台修改。
type Settings struct {
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description"`
	// SiteSlogan 显示在首页 Logo 下方的副标题。
	SiteSlogan string `json:"site_slogan"`
	// SiteIcon 支持 http(s) 地址、站内路径或 data:image/*。
	SiteIcon     string `json:"site_icon"`
	Announcement string `json:"announcement"`

	// 邮箱登录：允许用「用户名或邮箱 + 密码」登录。管理员账号始终允许密码登录，避免锁死。
	EmailLoginEnabled bool `json:"email_login_enabled"`
	// 邮箱注册：开启时必须同时配置并启用 SMTP，否则保存会被拒绝。
	EmailRegisterEnabled bool `json:"email_register_enabled"`
	// OAuth2 登录 / 注册：可独立开关，至少需要一个启用的 provider。
	OAuthLoginEnabled    bool `json:"oauth_login_enabled"`
	OAuthRegisterEnabled bool `json:"oauth_register_enabled"`

	// APIRequireLogin 为 true 时，/api/search、/api/check/links 与插件 Web 接口都需要登录。
	APIRequireLogin bool `json:"api_require_login"`
	// WebSearchEnabled 控制前端「直接搜索」是否对匿名用户开放。
	WebSearchEnabled bool `json:"web_search_enabled"`
	// AllowUserAPITokens 允许用户在用户中心生成个人 API 令牌。
	AllowUserAPITokens bool `json:"allow_user_api_tokens"`
	// DefaultDailySearchLimit 为 0 表示不限制；用户级设置优先。
	DefaultDailySearchLimit int `json:"default_daily_search_limit"`
	// AnonymousWebSearchLimit 匿名网页搜索的每分钟请求数上限。
	AnonymousWebSearchLimit int `json:"anonymous_web_search_limit"`
	// AnonymousDailyLimit 同一 IP 每天可用的匿名搜索次数，0 表示不限制。
	AnonymousDailyLimit int `json:"anonymous_daily_limit"`
	// AnonymousGlobalDailyLimit 全站每天允许的匿名搜索总次数，0 表示不限制。
	AnonymousGlobalDailyLimit int `json:"anonymous_global_daily_limit"`
	SessionTTLHours           int `json:"session_ttl_hours"`
}

func DefaultSettings() Settings {
	return Settings{
		SiteName:                  "清羽盘搜",
		SiteDescription:           "网盘资源搜索",
		SiteSlogan:                "清羽飞扬自建网盘搜索，众多资源，一键直达",
		SiteIcon:                  "/favicon.ico",
		EmailLoginEnabled:         true,
		EmailRegisterEnabled:      false,
		OAuthLoginEnabled:         false,
		OAuthRegisterEnabled:      false,
		APIRequireLogin:           true,
		WebSearchEnabled:          true,
		AllowUserAPITokens:        true,
		DefaultDailySearchLimit:   0,
		AnonymousWebSearchLimit:   20,
		AnonymousDailyLimit:       0,
		AnonymousGlobalDailyLimit: 0,
		SessionTTLHours:           168,
	}
}

type EmailSettings struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	UseTLS   bool   `json:"use_tls"`
}

func DefaultEmailSettings() EmailSettings {
	return EmailSettings{Port: 587, UseTLS: true, FromName: "清羽盘搜"}
}

type OAuthProvider struct {
	ID           int64  `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"userinfo_url"`
	Scopes       string `json:"scopes"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type OAuthAccount struct {
	ProviderID   int64  `json:"provider_id"`
	ProviderSlug string `json:"provider_slug"`
	ProviderName string `json:"provider_name"`
	Email        string `json:"email,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type SearchLog struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Source    string `json:"source"`
	Keyword   string `json:"keyword"`
	IP        string `json:"ip"`
	CreatedAt string `json:"created_at"`
}

type Overview struct {
	UsersTotal        int         `json:"users_total"`
	UsersActive       int         `json:"users_active"`
	AdminsTotal       int         `json:"admins_total"`
	SearchesToday     int         `json:"searches_today"`
	SearchesTotal     int         `json:"searches_total"`
	AnonymousToday    int         `json:"anonymous_today"`
	SessionsActive    int         `json:"sessions_active"`
	RecentSearches    []SearchLog `json:"recent_searches"`
	DefaultAdminAlive bool        `json:"default_admin_alive"`
}
