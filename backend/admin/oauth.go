package admin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"pansou/util/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// oauthRedirectBase 根据浏览器实际访问地址推断 OAuth 对外地址。
//
// 反向代理可能在 TLS 终止后以 HTTP 访问后端，也可能不传 X-Forwarded-Proto。
// 因此先看浏览器发来的 Origin/Referer，再看标准 Forwarded / X-Forwarded-*，
// 最后才回退到请求自身的 TLS 与 Host。部署时不需要额外配置域名。
func oauthRedirectBase(c *gin.Context) string {
	scheme, host := requestPublicSchemeHost(c)
	return scheme + "://" + host
}

func requestPublicSchemeHost(c *gin.Context) (string, string) {
	// 浏览器访问页面时会在 Origin/Referer 中带上真实的 scheme 与 host。
	for _, raw := range []string{c.GetHeader("Origin"), c.GetHeader("Referer")} {
		if u, err := url.Parse(strings.TrimSpace(raw)); err == nil &&
			u.Host != "" && (u.Scheme == "http" || u.Scheme == "https") {
			return strings.ToLower(u.Scheme), u.Host
		}
	}

	scheme, host := "", ""
	if forwarded := firstHeaderValue(c.GetHeader("Forwarded")); forwarded != "" {
		for _, part := range strings.Split(forwarded, ";") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			value := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			switch strings.ToLower(strings.TrimSpace(kv[0])) {
			case "proto":
				scheme = strings.ToLower(value)
			case "host":
				host = value
			}
		}
	}
	if scheme == "" {
		for _, header := range []string{"X-Forwarded-Proto", "X-Forwarded-Scheme", "X-Url-Scheme"} {
			if value := strings.ToLower(firstHeaderValue(c.GetHeader(header))); value == "http" || value == "https" {
				scheme = value
				break
			}
		}
	}
	if scheme == "" && strings.EqualFold(firstHeaderValue(c.GetHeader("X-Forwarded-Ssl")), "on") {
		scheme = "https"
	}
	if host == "" {
		host = firstHeaderValue(c.GetHeader("X-Forwarded-Host"))
	}
	if host == "" {
		host = c.Request.Host
	}
	if scheme == "" {
		// 公网域名默认按 HTTPS 处理；OAuth 服务商通常也只接受 HTTPS 回调。
		if c.Request.TLS != nil || !isLocalHost(host) {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme, host
}

func isLocalHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func firstHeaderValue(raw string) string {
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}

func oauthRedirectURL(c *gin.Context) string {
	return oauthRedirectBase(c) + "/api/auth/oauth/callback"
}

func buildOAuthConfig(p *OAuthProvider, redirectURL string) (*oauth2.Config, error) {
	if p == nil || !p.Enabled || p.ClientID == "" || p.ClientSecret == "" {
		return nil, fmt.Errorf("该 OAuth 服务商未启用或未配置")
	}
	if p.AuthURL == "" || p.TokenURL == "" || p.UserInfoURL == "" {
		return nil, fmt.Errorf("OAuth 服务商缺少 Auth/Token/UserInfo 地址")
	}
	scopes := strings.Fields(p.Scopes)
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthURL,
			TokenURL: p.TokenURL,
		},
	}, nil
}

type oauthUserInfo struct {
	Subject  string
	Email    string
	Name     string
	Username string
}

func fetchOAuthUser(ctx context.Context, c *gin.Context, p *OAuthProvider, token *oauth2.Token) (*oauthUserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pansou-web")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("获取用户信息失败: %s", truncate(string(body), 200))
	}
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}
	info := &oauthUserInfo{}
	for _, k := range []string{"sub", "id", "user_id", "uuid"} {
		if v, ok := raw[k]; ok && fmt.Sprint(v) != "" {
			info.Subject = fmt.Sprint(v)
			break
		}
	}
	if v, ok := raw["email"].(string); ok {
		info.Email = NormalizeEmail(v)
	}
	for _, k := range []string{"preferred_username", "login", "username", "nickname"} {
		if v, ok := raw[k].(string); ok && v != "" {
			info.Username = v
			break
		}
	}
	if v, ok := raw["name"].(string); ok {
		info.Name = v
	}
	if info.Username == "" {
		info.Username = info.Name
	}
	if info.Username == "" && info.Email != "" {
		info.Username = strings.Split(info.Email, "@")[0]
	}

	// GitHub 的 /user 在用户隐藏邮箱时返回空，需要再查 /user/emails。
	if info.Email == "" && strings.HasPrefix(strings.ToLower(p.UserInfoURL), "https://api.github.com") {
		if emails, err := fetchGitHubEmails(ctx, client, token); err == nil {
			for _, e := range emails {
				if primary, _ := e["primary"].(bool); primary {
					if em, ok := e["email"].(string); ok {
						info.Email = NormalizeEmail(em)
						break
					}
				}
			}
		}
	}
	if info.Subject == "" {
		return nil, fmt.Errorf("服务商未返回用户唯一标识")
	}
	return info, nil
}

func fetchGitHubEmails(ctx context.Context, client *http.Client, token *oauth2.Token) ([]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pansou-web")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var emails []map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&emails); err != nil {
		return nil, err
	}
	return emails, nil
}

func OAuthStart(c *gin.Context) {
	settings := LoadSettings()
	if !settings.OAuthLoginEnabled && !settings.OAuthRegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "OAuth 登录/注册未开启", "code": "OAUTH_DISABLED"})
		return
	}
	p, err := GetOAuthProviderBySlug(c.Param("slug"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "code": "OAUTH_PROVIDER_NOT_FOUND"})
		return
	}
	primary, err := GetPrimaryOAuthProvider()
	if err != nil || primary.ID != p.ID {
		c.JSON(http.StatusNotFound, gin.H{"error": "该 OAuth 服务商已停用", "code": "OAUTH_PROVIDER_NOT_FOUND"})
		return
	}
	redirectURL := oauthRedirectURL(c)
	cfg, err := buildOAuthConfig(p, redirectURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "OAUTH_NOT_CONFIGURED"})
		return
	}

	mode := c.DefaultQuery("mode", "login")
	userID := int64(0)
	if mode == "bind" {
		u := CurrentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录后再绑定", "code": "UNAUTHORIZED"})
			return
		}
		userID = u.ID
	} else {
		mode = "login"
	}
	state, err := CreateOAuthState(p.ID, mode, userID, redirectURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 OAuth 状态失败"})
		return
	}
	c.Redirect(http.StatusFound, cfg.AuthCodeURL(state, oauth2.AccessTypeOnline))
}

func OAuthCallback(c *gin.Context) {
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	if state == "" || code == "" {
		oauthRedirectError(c, "OAuth 回调缺少参数")
		return
	}
	st, err := TakeOAuthState(state)
	if err != nil {
		oauthRedirectError(c, err.Error())
		return
	}
	p, err := GetOAuthProviderByID(st.ProviderID)
	if err != nil {
		oauthRedirectError(c, err.Error())
		return
	}
	redirectURL := st.RedirectURI
	if redirectURL == "" {
		// 兼容升级前创建的旧 state。
		redirectURL = oauthRedirectURL(c)
	}
	cfg, err := buildOAuthConfig(p, redirectURL)
	if err != nil {
		oauthRedirectError(c, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		oauthRedirectError(c, "换取访问令牌失败: "+err.Error())
		return
	}
	info, err := fetchOAuthUser(ctx, c, p, token)
	if err != nil {
		oauthRedirectError(c, err.Error())
		return
	}

	settings := LoadSettings()

	if st.Mode == "bind" {
		user, err := GetUserByID(st.UserID)
		if err != nil {
			oauthRedirectError(c, "绑定账号不存在")
			return
		}
		if existingID, err := GetOAuthAccount(p.ID, info.Subject); err == nil && existingID != user.ID {
			oauthRedirectError(c, "该第三方账号已绑定到其他用户")
			return
		}
		if err := BindOAuthAccount(p.ID, info.Subject, user.ID, info.Email); err != nil {
			oauthRedirectError(c, "绑定失败: "+err.Error())
			return
		}
		RecordAudit(user, "oauth_bind", p.Name, "绑定第三方账号", c.ClientIP(), c.GetHeader("User-Agent"))
		oauthRedirectSuccess(c, "bound")
		return
	}

	userID, err := GetOAuthAccount(p.ID, info.Subject)
	if err == nil {
		if !settings.OAuthLoginEnabled {
			oauthRedirectError(c, "OAuth 登录未开启")
			return
		}
		user, err := GetUserByID(userID)
		if err != nil || !user.Active() {
			oauthRedirectError(c, "账号不存在或已被禁用")
			return
		}
		finishOAuthLogin(c, user)
		return
	}

	if !settings.OAuthRegisterEnabled {
		oauthRedirectError(c, "该第三方账号尚未注册，且 OAuth 注册未开启")
		return
	}
	username := uniqueUsername(info.Username)
	user, err := CreateOAuthUser(username, info.Email)
	if err != nil {
		oauthRedirectError(c, "创建账号失败: "+err.Error())
		return
	}
	if err := BindOAuthAccount(p.ID, info.Subject, user.ID, info.Email); err != nil {
		oauthRedirectError(c, "绑定第三方账号失败: "+err.Error())
		return
	}
	RecordAudit(user, "register", p.Name, "通过 OAuth 注册", c.ClientIP(), c.GetHeader("User-Agent"))
	finishOAuthLogin(c, user)
}

func finishOAuthLogin(c *gin.Context, user *User) {
	settings := LoadSettings()
	token, _, err := CreateSession(user.ID, c.ClientIP(), c.Request.UserAgent(), SessionTTLForSettings(settings))
	if err != nil {
		oauthRedirectError(c, "创建会话失败")
		return
	}
	TouchLastLogin(user.ID)
	setSessionCookie(c, token, SessionTTLForSettings(settings))
	RecordAudit(user, "login_success", "oauth", "OAuth 登录成功", c.ClientIP(), c.GetHeader("User-Agent"))
	oauthRedirectSuccess(c, "login")
}

func oauthRedirectSuccess(c *gin.Context, action string) {
	c.Redirect(http.StatusFound, "/#/user?oauth="+url.QueryEscape(action))
}

func oauthRedirectError(c *gin.Context, msg string) {
	c.Redirect(http.StatusFound, "/#/login?oauth_error="+url.QueryEscape(msg))
}

func CreateOAuthUser(username, email string) (*User, error) {
	username = sanitizeUsername(username)
	now := Now()
	var res interface{ LastInsertId() (int64, error) }
	var err error
	for i := 0; i < 5; i++ {
		name := username
		if i > 0 {
			name = fmt.Sprintf("%s_%s", username, randomHex(2))
		}
		res, err = DB.Exec(
			`INSERT INTO users (username, email, password_hash, role, status, must_change_password, daily_search_limit, created_at, updated_at)
			 VALUES (?, ?, '', ?, ?, 0, 0, ?, ?)`,
			name, NormalizeEmail(email), RoleUser, StatusActive, now, now,
		)
		if err == nil {
			id, _ := res.LastInsertId()
			return GetUserByID(id)
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, err
		}
	}
	return nil, ErrUserExists
}

func sanitizeUsername(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
		if b.Len() >= 24 {
			break
		}
	}
	name := strings.Trim(b.String(), "_-")
	if len(name) < 2 {
		name = "user_" + randomHex(3)
	}
	switch strings.ToLower(name) {
	case "admin", "root", "system", "administrator":
		name += "_" + randomHex(2)
	}
	return name
}

func uniqueUsername(raw string) string {
	name := sanitizeUsername(raw)
	if _, err := GetUserByUsername(name); err != nil {
		return name
	}
	return name + "_" + randomHex(3)
}
