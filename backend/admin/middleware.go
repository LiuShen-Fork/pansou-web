package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserKey      = "pansou_admin_user"
	ctxSessionIDKey = "pansou_admin_session"
	ctxAuthMethod   = "pansou_admin_auth_method"
)

// CurrentUser 返回当前请求已识别的用户（可能是 nil）。
func CurrentUser(c *gin.Context) *User {
	if v, ok := c.Get(ctxUserKey); ok {
		if u, ok := v.(*User); ok {
			return u
		}
	}
	return nil
}

func CurrentSessionID(c *gin.Context) string {
	if v, ok := c.Get(ctxSessionIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func bearerToken(c *gin.Context) (string, bool) {
	h := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:]), true
	}
	if t := strings.TrimSpace(c.GetHeader("X-API-Token")); t != "" {
		return t, true
	}
	return "", false
}

func sessionToken(c *gin.Context) string {
	if t, err := c.Cookie(CookieName); err == nil {
		return strings.TrimSpace(t)
	}
	return ""
}

// AttachUser 尝试识别会话或 API 令牌，但不会拒绝匿名请求。
func AttachUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Enabled() {
			c.Next()
			return
		}
		if token, ok := bearerToken(c); ok && token != "" {
			if strings.HasPrefix(token, "ps_") {
				if u, err := GetUserByAPIToken(token); err == nil {
					c.Set(ctxUserKey, u)
					c.Set(ctxAuthMethod, "api_token")
					c.Next()
					return
				}
				// API 令牌无效时不要回退到 Cookie：显式凭证优先。
				c.Next()
				return
			}
			if u, sess, err := GetSessionUser(token); err == nil {
				c.Set(ctxUserKey, u)
				c.Set(ctxSessionIDKey, sess.ID)
				c.Set(ctxAuthMethod, "bearer")
				c.Next()
				return
			}
			c.Next()
			return
		}

		if token := sessionToken(c); token != "" {
			if u, sess, err := GetSessionUser(token); err == nil {
				c.Set(ctxUserKey, u)
				c.Set(ctxSessionIDKey, sess.ID)
				c.Set(ctxAuthMethod, "cookie")
			}
		}
		c.Next()
	}
}

// RequireAuth 要求已登录；默认管理员未改密码时只放行必要的接口。
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "请先登录",
				"code":  "UNAUTHORIZED",
			})
			return
		}
		if u.MustChangePassword && !passwordChangeAllowed(c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "请先修改默认密码后再使用系统",
				"code":  "MUST_CHANGE_PASSWORD",
			})
			return
		}
		c.Next()
	}
}

func passwordChangeAllowed(path string) bool {
	switch path {
	case "/api/auth/me", "/api/auth/logout", "/api/auth/change-password":
		return true
	}
	return false
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录", "code": "UNAUTHORIZED"})
			return
		}
		if !u.IsAdmin() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限", "code": "FORBIDDEN"})
			return
		}
		c.Next()
	}
}

// AccessGate 控制搜索接口、链接检测接口和插件 Web 接口的访问。
func AccessGate() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		isSearch := path == "/api/search"
		isWebSearch := path == "/api/web/search"
		isPluginWeb := strings.HasPrefix(path, "/qqpd/") ||
			strings.HasPrefix(path, "/gying/") ||
			strings.HasPrefix(path, "/panlian/") ||
			strings.HasPrefix(path, "/weibo/") ||
			strings.HasPrefix(path, "/woniu/")
		isCheck := path == "/api/check/links" || path == "/api/check"
		if !isSearch && !isWebSearch && !isPluginWeb && !isCheck {
			c.Next()
			return
		}
		if !Enabled() {
			c.Next()
			return
		}

		settings := LoadSettings()
		user := CurrentUser(c)

		if user == nil && (isPluginWeb || isCheck) && settings.APIRequireLogin {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "该接口需要登录后使用", "code": "UNAUTHORIZED"})
			return
		}
		if user == nil && isSearch && settings.APIRequireLogin {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API 搜索需要登录，请在请求头携带 Authorization: Bearer <API令牌>",
				"code":  "UNAUTHORIZED",
			})
			return
		}
		if user == nil && isWebSearch && !settings.WebSearchEnabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "网页直接搜索已被管理员关闭，请登录后使用",
				"code":  "WEB_SEARCH_DISABLED",
			})
			return
		}
		// 匿名网页搜索必须携带同源页面签发的短期凭证，避免直接把
		// /api/web/search 当成公开 API。已登录/API 令牌请求不受此限制。
		if user == nil && isWebSearch {
			token := c.GetHeader("X-Web-Token")
			if !sameOriginRequest(c) || !ValidateWebToken(token, c.ClientIP(), c.GetHeader("User-Agent")) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "网页搜索凭证无效，请刷新页面后重试",
					"code":  "WEB_TOKEN_REQUIRED",
				})
				return
			}
		}
		if user != nil && user.MustChangePassword {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "请先修改默认密码后再使用搜索功能",
				"code":  "MUST_CHANGE_PASSWORD",
			})
			return
		}

		if (isSearch || isWebSearch) && user == nil {
			if !allowAnonymousSearch(c.ClientIP(), settings.AnonymousWebSearchLimit) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "请求过于频繁，请稍后再试或登录使用",
					"code":  "RATE_LIMITED",
				})
				return
			}
			if settings.AnonymousDailyLimit > 0 {
				count, err := AnonymousIPSearchCountToday(c.ClientIP())
				if err == nil && count >= settings.AnonymousDailyLimit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"error": fmt.Sprintf("当前 IP 今日匿名搜索次数已达上限（%d 次），请登录后继续使用", settings.AnonymousDailyLimit),
						"code":  "ANON_IP_DAILY_LIMIT",
					})
					return
				}
			}
			if settings.AnonymousGlobalDailyLimit > 0 {
				count, err := AnonymousGlobalSearchCountToday()
				if err == nil && count >= settings.AnonymousGlobalDailyLimit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"error": "今日全站匿名搜索额度已用完，请登录后继续使用",
						"code":  "ANON_GLOBAL_DAILY_LIMIT",
					})
					return
				}
			}
		}
		if (isSearch || isWebSearch) && user != nil {
			if limit := EffectiveDailyLimit(user, settings); limit > 0 {
				count, _ := UserSearchCountToday(user.ID)
				if count >= limit {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"error": fmt.Sprintf("今日搜索次数已达上限（%d 次）", limit),
						"code":  "DAILY_LIMIT_REACHED",
					})
					return
				}
			}
			// API 独立额度：仅统计 /api/search，所有令牌共享。
			if isSearch {
				if limit := EffectiveAPIDailyLimit(user, settings); limit > 0 {
					count, _ := UserAPISearchCountToday(user.ID)
					if count >= limit {
						c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
							"error": fmt.Sprintf("今日 API 调用次数已达上限（%d 次）", limit),
							"code":  "API_DAILY_LIMIT_REACHED",
						})
						return
					}
				}
			}
		}

		// 只有真正的搜索计入搜索日志与配额；插件 Web 接口/链接检测
		// 属于账户操作，不应消耗用户的每日搜索次数。
		if isSearch || isWebSearch {
			source := "api"
			if isWebSearch {
				source = "web"
			}
			RecordSearch(user, source, c.Query("kw"), c.ClientIP())
		}
		c.Next()
	}
}

// CSRFGuard 对使用 Cookie 的非 GET 请求校验 Origin，避免跨站表单提交。
func CSRFGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if _, ok := bearerToken(c); ok {
			c.Next()
			return
		}
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" {
			c.Next()
			return
		}
		u, err := url.Parse(origin)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "非法请求来源", "code": "CSRF_REJECTED"})
			return
		}
		expected := c.Request.Host
		if xf := c.GetHeader("X-Forwarded-Host"); xf != "" {
			expected = xf
		}
		if strings.EqualFold(u.Host, expected) {
			c.Next()
			return
		}
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "非法请求来源", "code": "CSRF_REJECTED"})
	}
}

// ---------------- 匿名搜索限流 ----------------

type anonBucket struct {
	count int
	reset time.Time
}

var anonLimiter = struct {
	sync.Mutex
	buckets map[string]*anonBucket
}{buckets: map[string]*anonBucket{}}

func allowAnonymousSearch(ip string, perMinute int) bool {
	if perMinute <= 0 {
		perMinute = 20
	}
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	anonLimiter.Lock()
	defer anonLimiter.Unlock()
	b, ok := anonLimiter.buckets[ip]
	if !ok || now.After(b.reset) {
		anonLimiter.buckets[ip] = &anonBucket{count: 1, reset: now.Add(time.Minute)}
		if len(anonLimiter.buckets) > 10000 {
			for k, v := range anonLimiter.buckets {
				if now.After(v.reset) {
					delete(anonLimiter.buckets, k)
				}
			}
		}
		return true
	}
	if b.count >= perMinute {
		return false
	}
	b.count++
	return true
}

// ---------------- Cookie ----------------

func setSessionCookie(c *gin.Context, token string, ttl time.Duration) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, token, int(ttl.Seconds()), "/", "", secure, true)
}

func clearSessionCookie(c *gin.Context) {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, "", -1, "/", "", secure, true)
}
