package admin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"pansou/config"
	"pansou/util"
)

const settingKeyRuntime = "runtime_config"

// RuntimeSettings 是可以在管理后台维护的运行参数。
//
// 与站点/认证设置不同，这些值影响出站网络、缓存与并发。后台记录优先于
// 环境变量；没有记录时继续沿用镜像默认值或环境变量，保证旧部署平滑升级。
type RuntimeSettings struct {
	ProxyURL               string `json:"proxy_url"`
	CacheEnabled           bool   `json:"cache_enabled"`
	CacheTTLMinutes        int    `json:"cache_ttl_minutes"`
	DefaultConcurrency     int    `json:"default_concurrency"`
	OutboundMaxConcurrency int    `json:"outbound_max_concurrency"`
	// WebTokenSecret 仅用于持久化，接口响应永远不会回传。
	WebTokenSecret string `json:"web_token_secret,omitempty"`
	// WebTokenSecretCleared 表示管理员明确清除过密钥，重启后也不应回退到环境变量。
	WebTokenSecretCleared bool `json:"web_token_secret_cleared,omitempty"`
}

func defaultRuntimeSettings() RuntimeSettings {
	cacheEnabled := true
	outbound := 0
	proxyURL := ""
	if config.AppConfig != nil {
		cacheEnabled = config.AppConfig.CacheEnabled
		outbound = config.AppConfig.OutboundMaxConcurrency
		proxyURL = config.GetProxyURL()
	}
	return RuntimeSettings{
		ProxyURL:               proxyURL,
		CacheEnabled:           cacheEnabled,
		CacheTTLMinutes:        config.GetCacheTTLMinutes(),
		DefaultConcurrency:     config.ExplicitDefaultConcurrency(),
		OutboundMaxConcurrency: outbound,
	}
}

// LoadRuntimeSettings 返回后台保存的运行参数。第二个返回值表示数据库中
// 是否已经存在记录；不存在时返回当前生效值，便于前端表单直接展示。
func LoadRuntimeSettings() (RuntimeSettings, bool) {
	s := RuntimeSettings{}
	if err := GetSetting(settingKeyRuntime, &s); err != nil {
		return defaultRuntimeSettings(), false
	}
	return s, true
}

func SaveRuntimeSettings(s RuntimeSettings) error {
	return SetSetting(settingKeyRuntime, s)
}

// ApplyRuntimeSettings 在配置初始化后、启动搜索服务前调用，把后台值覆盖
// 到运行配置上。缓存开关与出口并发只能在启动阶段应用，因此这里统一处理。
func ApplyRuntimeSettings() {
	s, ok := LoadRuntimeSettings()
	if !ok {
		return
	}
	if config.AppConfig == nil {
		return
	}
	config.SetProxyURL(s.ProxyURL)
	config.AppConfig.CacheEnabled = s.CacheEnabled
	config.SetCacheTTLMinutes(s.CacheTTLMinutes)
	// 0 表示后台接管为自动计算，同时忽略环境变量 CONCURRENCY。
	config.SetDefaultConcurrency(s.DefaultConcurrency)
	config.AppConfig.OutboundMaxConcurrency = s.OutboundMaxConcurrency
	// 没有显式密钥也没有明确清除时，保留环境变量或进程随机密钥。
	if s.WebTokenSecret != "" || s.WebTokenSecretCleared {
		SetWebTokenSecret(s.WebTokenSecret)
	}
}

func effectiveRuntimeSettings() gin.H {
	if config.AppConfig == nil {
		return gin.H{}
	}
	concurrency := config.GetDefaultConcurrency()
	return gin.H{
		"proxy_url":                config.GetProxyURL(),
		"cache_enabled":            config.AppConfig.CacheEnabled,
		"cache_ttl_minutes":        config.GetCacheTTLMinutes(),
		"default_concurrency":      concurrency,
		"default_concurrency_auto": config.ExplicitDefaultConcurrency() == 0,
		"outbound_max_concurrency": config.GetOutboundMaxConcurrency(),
	}
}

func AdminGetRuntime(c *gin.Context) {
	s, _ := LoadRuntimeSettings()
	s.WebTokenSecret = ""
	c.JSON(http.StatusOK, gin.H{
		"settings":             s,
		"web_token_secret_set": WebTokenSecretConfigured(),
		"effective":            effectiveRuntimeSettings(),
	})
}

type runtimePayload struct {
	RuntimeSettings
	ClearWebTokenSecret bool `json:"clear_web_token_secret"`
}

func AdminPutRuntime(c *gin.Context) {
	var body runtimePayload
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	s := body.RuntimeSettings
	s.ProxyURL = strings.TrimSpace(s.ProxyURL)
	if err := util.ValidateProxyURL(s.ProxyURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "代理地址无效: " + err.Error(), "code": "BAD_PROXY"})
		return
	}
	if s.CacheTTLMinutes < 1 || s.CacheTTLMinutes > 10080 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缓存 TTL 需在 1-10080 分钟之间", "code": "BAD_CACHE_TTL"})
		return
	}
	if s.DefaultConcurrency < 0 || s.DefaultConcurrency > 100000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "默认并发数不合法", "code": "BAD_CONCURRENCY"})
		return
	}
	if s.OutboundMaxConcurrency < 0 || s.OutboundMaxConcurrency > 100000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "出口并发数不合法", "code": "BAD_OUTBOUND_CONCURRENCY"})
		return
	}

	old, oldExists := LoadRuntimeSettings()
	secret := ""
	secretCleared := false
	if oldExists {
		secret = old.WebTokenSecret
		secretCleared = old.WebTokenSecretCleared
	}
	if body.ClearWebTokenSecret {
		secret = ""
		secretCleared = true
	} else if v := strings.TrimSpace(body.WebTokenSecret); v != "" {
		secret = v
		secretCleared = false
	}
	if secret != "" && (len(secret) < 16 || len(secret) > 512) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "网页搜索签名密钥长度需在 16-512 个字符之间",
			"code":  "BAD_WEB_TOKEN_SECRET",
		})
		return
	}
	s.WebTokenSecret = secret
	s.WebTokenSecretCleared = secretCleared

	restart := make([]string, 0, 4)
	if s.ProxyURL != config.GetProxyURL() {
		restart = append(restart, "proxy_url")
	}
	if config.AppConfig != nil && config.AppConfig.CacheEnabled != s.CacheEnabled {
		restart = append(restart, "cache_enabled")
	}
	newOutbound := s.OutboundMaxConcurrency
	if newOutbound <= 0 {
		newOutbound = config.DefaultOutboundMaxConcurrency
	}
	if config.GetOutboundMaxConcurrency() != newOutbound {
		restart = append(restart, "outbound_max_concurrency")
	}
	concurrencyNeedsRestart := s.DefaultConcurrency == 0 && config.ExplicitDefaultConcurrency() != 0
	if concurrencyNeedsRestart {
		// 从固定值切回自动时，需要重启后按实际插件数重算。
		restart = append(restart, "default_concurrency")
	}

	if err := SaveRuntimeSettings(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存运行参数失败", "code": "SAVE_FAILED"})
		return
	}

	// 可热更新的项立即应用；缓存开关与出口并发等待容器重启。
	config.SetCacheTTLMinutes(s.CacheTTLMinutes)
	if s.DefaultConcurrency > 0 || !concurrencyNeedsRestart {
		config.SetDefaultConcurrency(s.DefaultConcurrency)
	}
	// 只有在密钥确实变化时才重建随机密钥，避免每次保存都让现有网页凭证失效。
	secretChanged := body.ClearWebTokenSecret ||
		(!oldExists && s.WebTokenSecret != "") ||
		(oldExists && (old.WebTokenSecret != s.WebTokenSecret || old.WebTokenSecretCleared != s.WebTokenSecretCleared))
	if secretChanged {
		SetWebTokenSecret(s.WebTokenSecret)
	}

	response := s
	response.WebTokenSecret = ""
	c.JSON(http.StatusOK, gin.H{
		"settings":             response,
		"web_token_secret_set": WebTokenSecretConfigured(),
		"effective":            effectiveRuntimeSettings(),
		"restart_required":     restart,
	})
}
