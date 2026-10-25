package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Account  string `json:"account"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type profileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func PublicConfig(c *gin.Context) {
	s := LoadSettings()
	e := LoadEmailSettings()
	providers, _ := ListOAuthProviders(true)
	list := make([]gin.H, 0, len(providers))
	for _, p := range providers {
		list = append(list, gin.H{"slug": p.Slug, "name": p.Name})
	}
	c.JSON(http.StatusOK, gin.H{
		"site": gin.H{
			"name":         s.SiteName,
			"description":  s.SiteDescription,
			"announcement": s.Announcement,
		},
		"auth": gin.H{
			"email_login_enabled":        s.EmailLoginEnabled,
			"email_register_enabled":     s.EmailRegisterEnabled,
			"oauth_login_enabled":        s.OAuthLoginEnabled,
			"oauth_register_enabled":     s.OAuthRegisterEnabled,
			"api_require_login":          s.APIRequireLogin,
			"web_search_enabled":         s.WebSearchEnabled,
			"allow_user_api_tokens":      s.AllowUserAPITokens,
			"default_daily_search_limit": s.DefaultDailySearchLimit,
			"smtp_configured":            e.Enabled && e.Host != "",
		},
		"oauth_providers": list,
	})
}

func Me(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": "UNAUTHORIZED"})
		return
	}
	s := LoadSettings()
	token, _ := GetAPIToken(u.ID)
	accounts, _ := ListUserOAuthAccounts(u.ID)
	count, _ := UserSearchCountToday(u.ID)
	u.SearchCountToday = count
	c.JSON(http.StatusOK, gin.H{
		"user": u,
		"auth": gin.H{
			"email_login_enabled":        s.EmailLoginEnabled,
			"email_register_enabled":     s.EmailRegisterEnabled,
			"oauth_login_enabled":        s.OAuthLoginEnabled,
			"oauth_register_enabled":     s.OAuthRegisterEnabled,
			"api_require_login":          s.APIRequireLogin,
			"web_search_enabled":         s.WebSearchEnabled,
			"allow_user_api_tokens":      s.AllowUserAPITokens,
			"default_daily_search_limit": s.DefaultDailySearchLimit,
		},
		"token":          token,
		"oauth_accounts": accounts,
		"is_admin":       u.IsAdmin(),
	})
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	account := strings.TrimSpace(req.Account)
	if account == "" {
		account = strings.TrimSpace(req.Username)
	}
	if account == "" || len(account) > 254 || req.Password == "" || len(req.Password) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码", "code": "BAD_REQUEST"})
		return
	}
	ip := c.ClientIP()
	byIP, byAccount := CountRecentFailures(ip, account, 15*time.Minute)
	if byIP >= 5 || byAccount >= 5 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "登录失败次数过多，请 15 分钟后再试", "code": "LOGIN_THROTTLED"})
		return
	}

	settings := LoadSettings()
	user, err := FindUserByAccount(account)
	if err != nil || !CheckPassword(user.PasswordHash, req.Password) {
		RecordLoginAttempt(ip, account, false)
		time.Sleep(250 * time.Millisecond)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误", "code": "INVALID_CREDENTIALS"})
		return
	}
	if !user.Active() {
		RecordLoginAttempt(ip, account, false)
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用", "code": "USER_DISABLED"})
		return
	}
	// 关闭邮箱登录后，仅管理员仍可用密码登录，避免把自己锁在门外。
	if !settings.EmailLoginEnabled && !user.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "密码登录已被管理员关闭，请使用 OAuth 登录", "code": "PASSWORD_LOGIN_DISABLED"})
		return
	}

	ttl := SessionTTLForSettings(settings)
	token, _, err := CreateSession(user.ID, ip, c.Request.UserAgent(), ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败", "code": "SESSION_FAILED"})
		return
	}
	TouchLastLogin(user.ID)
	RecordLoginAttempt(ip, account, true)
	setSessionCookie(c, token, ttl)
	count, _ := UserSearchCountToday(user.ID)
	user.SearchCountToday = count

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": time.Now().Add(ttl).Unix(),
		"user":       user,
	})
}

func SendRegisterCode(c *gin.Context) {
	settings := LoadSettings()
	if !settings.EmailRegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "邮箱注册未开启", "code": "REGISTER_DISABLED"})
		return
	}
	emailCfg := LoadEmailSettings()
	if !emailCfg.Enabled || emailCfg.Host == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "管理员尚未配置 SMTP，暂时无法注册", "code": "EMAIL_NOT_CONFIGURED"})
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	email := NormalizeEmail(body.Email)
	if err := ValidateEmail(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_EMAIL"})
		return
	}
	if u, err := GetUserByEmail(email); err == nil && u != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册", "code": "EMAIL_EXISTS"})
		return
	}
	if ok, msg := EmailCodeSendAllowed(email); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": msg, "code": "RATE_LIMITED"})
		return
	}
	code, err := CreateEmailCode(email, "register", 10*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成验证码失败", "code": "CODE_FAILED"})
		return
	}
	subject, bodyText := BuildCodeMail(settings.SiteName, code, "register")
	if err := SendMail(emailCfg, email, subject, bodyText); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "验证码邮件发送失败：" + err.Error(), "code": "EMAIL_SEND_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "expires_in": 600})
}

func Register(c *gin.Context) {
	settings := LoadSettings()
	if !settings.EmailRegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "注册未开启", "code": "REGISTER_DISABLED"})
		return
	}
	emailCfg := LoadEmailSettings()
	if !emailCfg.Enabled || emailCfg.Host == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "SMTP 未配置，无法完成邮箱验证", "code": "EMAIL_NOT_CONFIGURED"})
		return
	}
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if err := ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_USERNAME"})
		return
	}
	if err := ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_PASSWORD"})
		return
	}
	email := NormalizeEmail(req.Email)
	if err := ValidateEmail(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_EMAIL"})
		return
	}
	if strings.TrimSpace(req.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写邮箱验证码", "code": "CODE_REQUIRED"})
		return
	}
	if err := VerifyEmailCode(email, req.Code, "register"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "CODE_INVALID"})
		return
	}
	user, err := CreateUser(req.Username, email, req.Password, RoleUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "CREATE_FAILED"})
		return
	}
	ttl := SessionTTLForSettings(settings)
	token, _, err := CreateSession(user.ID, c.ClientIP(), c.Request.UserAgent(), ttl)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败", "code": "SESSION_FAILED"})
		return
	}
	TouchLastLogin(user.ID)
	setSessionCookie(c, token, ttl)
	c.JSON(http.StatusCreated, gin.H{
		"token":      token,
		"expires_at": time.Now().Add(ttl).Unix(),
		"user":       user,
	})
}

func Logout(c *gin.Context) {
	if sid := CurrentSessionID(c); sid != "" {
		DeleteSession(sid)
	}
	clearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ChangePassword(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": "UNAUTHORIZED"})
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if u.PasswordHash != "" && !CheckPassword(u.PasswordHash, req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前密码不正确", "code": "INVALID_PASSWORD"})
		return
	}
	if err := UpdateUserPassword(u.ID, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "BAD_PASSWORD"})
		return
	}
	if sid := CurrentSessionID(c); sid != "" {
		DeleteOtherSessions(u.ID, sid)
	} else {
		DeleteUserSessions(u.ID)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func UpdateProfile(c *gin.Context) {
	u := CurrentUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": "UNAUTHORIZED"})
		return
	}
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	upd := UserUpdate{}
	if strings.TrimSpace(req.Username) != "" && strings.TrimSpace(req.Username) != u.Username {
		upd.Username = &req.Username
	}
	email := NormalizeEmail(req.Email)
	if email != "" && email != u.Email {
		upd.Email = &email
	}
	updated, err := UpdateUser(u.ID, upd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "UPDATE_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": updated})
}

func APITokenGet(c *gin.Context) {
	settings := LoadSettings()
	if !settings.AllowUserAPITokens {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员已关闭个人 API 令牌", "code": "API_TOKEN_DISABLED"})
		return
	}
	u := CurrentUser(c)
	t, err := EnsureAPIToken(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取令牌失败", "code": "TOKEN_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": t})
}

func APITokenReset(c *gin.Context) {
	settings := LoadSettings()
	if !settings.AllowUserAPITokens {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员已关闭个人 API 令牌", "code": "API_TOKEN_DISABLED"})
		return
	}
	u := CurrentUser(c)
	t, token, err := ResetAPIToken(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置令牌失败", "code": "TOKEN_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "prefix": t.Prefix, "created_at": t.CreatedAt})
}

func OAuthBindings(c *gin.Context) {
	u := CurrentUser(c)
	list, err := ListUserOAuthAccounts(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取绑定失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func UnbindOAuth(c *gin.Context) {
	u := CurrentUser(c)
	providerID, err := strconv.ParseInt(c.Query("provider_id"), 10, 64)
	if err != nil || providerID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 provider_id"})
		return
	}
	if err := UnbindOAuthAccount(u.ID, providerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
