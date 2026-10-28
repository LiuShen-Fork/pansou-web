package admin

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"pansou/util/json"
)

// 网页搜索凭证：匿名用户必须先从同源页面获取短期签名 token，
// 且 token 与 IP / User-Agent 绑定。它不能替代登录，但能显著提高
// 直接把 /api/web/search 当成免费公开接口的逆向成本。
const webTokenTTL = 2 * time.Hour

type webTokenPayload struct {
	ExpiresAt int64  `json:"exp"`
	IPHash    string `json:"ip"`
	UAHash    string `json:"ua"`
}

var (
	webTokenSecretMu       sync.RWMutex
	webTokenSecret         []byte
	webTokenSecretExplicit bool
)

func initWebTokenSecret() {
	value := strings.TrimSpace(os.Getenv("WEB_TOKEN_SECRET"))
	if len(value) >= 16 {
		webTokenSecret = []byte(value)
		webTokenSecretExplicit = true
		return
	}
	webTokenSecret = randomWebTokenSecret()
}

func randomWebTokenSecret() []byte {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败极罕见；退化为进程级种子，重启即失效。
		return []byte(Now() + "-pansou-web-token")
	}
	return buf
}

func init() {
	initWebTokenSecret()
}

// SetWebTokenSecret 由管理后台设置网页搜索凭证签名密钥。
// 空字符串或长度不足 16 时恢复为进程级随机密钥（现有凭证立即失效）。
func SetWebTokenSecret(value string) {
	value = strings.TrimSpace(value)
	webTokenSecretMu.Lock()
	defer webTokenSecretMu.Unlock()
	if len(value) >= 16 {
		webTokenSecret = []byte(value)
		webTokenSecretExplicit = true
		return
	}
	webTokenSecret = randomWebTokenSecret()
	webTokenSecretExplicit = false
}

// WebTokenSecretConfigured 返回当前是否使用显式配置的密钥（环境变量或后台）。
func WebTokenSecretConfigured() bool {
	webTokenSecretMu.RLock()
	defer webTokenSecretMu.RUnlock()
	return webTokenSecretExplicit
}

func currentWebTokenSecret() []byte {
	webTokenSecretMu.RLock()
	defer webTokenSecretMu.RUnlock()
	out := make([]byte, len(webTokenSecret))
	copy(out, webTokenSecret)
	return out
}

func hashBinding(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func signWebPayload(payload []byte) []byte {
	mac := hmac.New(sha256.New, currentWebTokenSecret())
	mac.Write(payload)
	return mac.Sum(nil)
}

func issueWebToken(ip, ua string) (string, int64) {
	expires := time.Now().Add(webTokenTTL).Unix()
	payload, _ := json.Marshal(webTokenPayload{
		ExpiresAt: expires,
		IPHash:    hashBinding(strings.TrimSpace(ip)),
		UAHash:    hashBinding(strings.TrimSpace(ua)),
	})
	sig := signWebPayload(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig), expires
}

func ValidateWebToken(token, ip, ua string) bool {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 || len(token) > 1024 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expected := signWebPayload(payload)
	if subtle.ConstantTimeCompare(sig, expected) != 1 {
		return false
	}
	var data webTokenPayload
	if err := json.Unmarshal(payload, &data); err != nil {
		return false
	}
	if data.ExpiresAt <= time.Now().Unix() {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(data.IPHash), []byte(hashBinding(strings.TrimSpace(ip)))) != 1 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(data.UAHash), []byte(hashBinding(strings.TrimSpace(ua)))) == 1
}

// sameOriginRequest 只接受浏览器同源页面发起的请求。
// 现代浏览器会发送 Sec-Fetch-Site；缺失时再回退校验 Origin / Referer。
func sameOriginRequest(c *gin.Context) bool {
	if site := strings.ToLower(strings.TrimSpace(c.GetHeader("Sec-Fetch-Site"))); site != "" {
		return site == "same-origin" || site == "none"
	}
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	if origin == "" {
		origin = strings.TrimSpace(c.GetHeader("Referer"))
	}
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	expected := c.Request.Host
	if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
		expected = forwarded
	}
	if strings.EqualFold(u.Host, expected) {
		return true
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		// 本地开发时 Vite 代理会把 Host 改成后端地址，放行本机来源。
		return true
	}
	return false
}

// WebTokenIssue 为同源页面签发短期网页搜索凭证。
func WebTokenIssue(c *gin.Context) {
	settings := LoadSettings()
	if !settings.WebSearchEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "网页直接搜索已关闭", "code": "WEB_SEARCH_DISABLED"})
		return
	}
	if !sameOriginRequest(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "请从站点页面发起搜索", "code": "WEB_ORIGIN_REJECTED"})
		return
	}
	token, expires := issueWebToken(c.ClientIP(), c.GetHeader("User-Agent"))
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"token": token, "expires_at": expires})
}
