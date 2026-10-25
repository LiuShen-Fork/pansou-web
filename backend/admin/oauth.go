package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func oauthRedirectURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := c.Request.Host
	if xf := c.GetHeader("X-Forwarded-Host"); xf != "" {
		host = xf
	}
	return scheme + "://" + host + "/api/auth/oauth/callback"
}

func buildOAuthConfig(c *gin.Context, p *OAuthProvider) (*oauth2.Config, error) {
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
		RedirectURL:  oauthRedirectURL(c),
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
	cfg, err := buildOAuthConfig(c, p)
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
	state, err := CreateOAuthState(p.ID, mode, userID)
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
	cfg, err := buildOAuthConfig(c, p)
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
			_, _ = EnsureAPIToken(id)
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
