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
	ctxUserKey        = "pansou_admin_user"
	ctxSessionIDKey   = "pansou_admin_session"
	ctxAuthMethod     = "pansou_admin_auth_method"
	ctxSearchLogIDKey = "pansou_search_log_id"
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

		source := "api"
		if isWebSearch {
			source = "web"
		}
		ip := c.ClientIP()

		if (isSearch || isWebSearch) && user == nil {
			perMinute := settings.AnonymousRateLimit
			if !allowAnonymousSearch(ip, perMinute) {
				denyQuota(c, "ANON_RATE_LIMITED", "anonymous_minute",
					fmt.Sprintf("匿名搜索过于频繁（每分钟上限 %d 次），请稍后再试或登录使用", perMinute),
					perMinute, perMinute, 60, true)
				return
			}
			if used, err := AnonymousIPSearchCountToday(ip); err == nil && used >= settings.AnonymousDailyLimit {
				denyQuota(c, "ANON_IP_DAILY_LIMIT", "anonymous_ip_daily",
					fmt.Sprintf("当前 IP 今日匿名搜索次数已达上限（%d 次），登录后可享受更高额度", settings.AnonymousDailyLimit),
					settings.AnonymousDailyLimit, used, 0, true)
				return
			}
			if used, err := AnonymousGlobalSearchCountToday(); err == nil && used >= settings.AnonymousGlobalDailyLimit {
				denyQuota(c, "ANON_GLOBAL_DAILY_LIMIT", "anonymous_global_daily",
					fmt.Sprintf("今日全站匿名搜索额度已用完（%d 次），登录后仍可继续搜索", settings.AnonymousGlobalDailyLimit),
					settings.AnonymousGlobalDailyLimit, used, 0, true)
				return
			}
		}
		if (isSearch || isWebSearch) && user != nil {
			dailyLimit, rateLimit, countToday := EffectiveWebDailyLimit(user, settings), EffectiveWebRateLimit(user, settings), UserWebSearchCountToday
			dailyCode, rateCode, scope := "WEB_DAILY_LIMIT_REACHED", "WEB_RATE_LIMITED", "user_web"
			if isSearch {
				dailyLimit, rateLimit, countToday = EffectiveAPIDailyLimit(user, settings), EffectiveAPIRateLimit(user, settings), UserAPISearchCountToday
				dailyCode, rateCode, scope = "API_DAILY_LIMIT_REACHED", "API_RATE_LIMITED", "user_api"
			}
			if used, err := UserRecentSearchCount(user.ID, source, time.Minute); err == nil && used >= rateLimit {
				denyQuota(c, rateCode, scope+"_minute",
					fmt.Sprintf("当前账号每分钟最多 %d 次 %s 请求，请稍后再试", rateLimit, quotaSourceName(isSearch)),
					rateLimit, used, 60, false)
				return
			}
			if used, err := countToday(user.ID); err == nil && used >= dailyLimit {
				denyQuota(c, dailyCode, scope+"_daily",
					fmt.Sprintf("当前账号今日%s额度已用完（%d 次），请联系管理员提高额度", quotaSourceName(isSearch), dailyLimit),
					dailyLimit, used, 0, false)
				return
			}
		}

		// 只有真正的搜索计入搜索日志与配额；插件 Web 接口/链接检测
		// 属于账户操作，不应消耗用户的每日搜索次数。
		if isSearch || isWebSearch {
			id := StartSearchLog(user, source, c.Query("kw"), ip, c.GetHeader("User-Agent"))
			c.Set(ctxSearchLogIDKey, id)
		}
		c.Next()
	}
}

func quotaSourceName(isAPI bool) string {
	if isAPI {
		return "API"
	}
	return "网页搜索"
}

// denyQuota 返回带配额详情的 429，并记录一条审计日志。
func denyQuota(c *gin.Context, code, scope, message string, limit, used, retryAfter int, loginRecommended bool) {
	if user := CurrentUser(c); user != nil {
		RecordAudit(user, "quota_exceeded", scope, message, c.ClientIP(), c.GetHeader("User-Agent"))
	} else {
		RecordAudit(nil, "quota_exceeded", scope, message, c.ClientIP(), c.GetHeader("User-Agent"))
	}
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error":             message,
		"code":              code,
		"scope":             scope,
		"limit":             limit,
		"used":              used,
		"retry_after":       retryAfter,
		"login_recommended": loginRecommended,
		"contact_admin":     !loginRecommended,
	})
}

// SearchLogID 返回当前请求对应的搜索日志 ID。
func SearchLogID(c *gin.Context) int64 {
	if v, ok := c.Get(ctxSearchLogIDKey); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 0
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
		perMinute = DefaultSettings().AnonymousRateLimit
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
