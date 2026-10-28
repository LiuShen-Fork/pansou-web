package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"pansou/config"
	jsonutil "pansou/util/json"
)

func withTestConfig(t *testing.T) func() {
	t.Helper()
	old := config.AppConfig
	config.AppConfig = &config.Config{
		CacheEnabled:           true,
		CacheTTLMinutes:        60,
		OutboundMaxConcurrency: 0,
		DefaultConcurrency:     30,
	}
	return func() {
		config.SetCacheTTLMinutes(0)
		config.SetDefaultConcurrency(0)
		config.SetProxyURL("")
		config.AppConfig = old
	}
}

func TestApplyRuntimeSettingsOverridesConfig(t *testing.T) {
	cleanupDB := setupTestDB(t)
	defer cleanupDB()
	cleanupConfig := withTestConfig(t)
	defer cleanupConfig()

	want := RuntimeSettings{
		ProxyURL:               "socks5://127.0.0.1:7890",
		CacheEnabled:           false,
		CacheTTLMinutes:        120,
		DefaultConcurrency:     42,
		OutboundMaxConcurrency: 256,
		WebTokenSecret:         "0123456789abcdef",
	}
	if err := SaveRuntimeSettings(want); err != nil {
		t.Fatal(err)
	}

	ApplyRuntimeSettings()

	if config.AppConfig.CacheEnabled {
		t.Fatal("后台关闭缓存后仍为启用状态")
	}
	if got := config.GetCacheTTLMinutes(); got != 120 {
		t.Fatalf("缓存 TTL = %d，期望 120", got)
	}
	if got := config.GetDefaultConcurrency(); got != 42 {
		t.Fatalf("默认并发 = %d，期望 42", got)
	}
	if got := config.GetOutboundMaxConcurrency(); got != 256 {
		t.Fatalf("出口并发 = %d，期望 256", got)
	}
	if got := config.GetProxyURL(); got != want.ProxyURL {
		t.Fatalf("代理 = %q，期望 %q", got, want.ProxyURL)
	}
	if !WebTokenSecretConfigured() {
		t.Fatal("后台签名密钥未生效")
	}
}

func TestAdminRuntimeSaveValidatesAndReportsRestart(t *testing.T) {
	cleanupDB := setupTestDB(t)
	defer cleanupDB()
	cleanupConfig := withTestConfig(t)
	defer cleanupConfig()

	gin.SetMode(gin.TestMode)
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPut, "/api/admin/runtime", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		AdminPutRuntime(c)
		return rec
	}

	bad := call(`{"proxy_url":"ftp://127.0.0.1:21","cache_enabled":true,"cache_ttl_minutes":60,"default_concurrency":0,"outbound_max_concurrency":0}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("非法代理返回 %d，期望 400", bad.Code)
	}

	rec := call(`{"proxy_url":"http://127.0.0.1:7890","cache_enabled":false,"cache_ttl_minutes":90,"default_concurrency":64,"outbound_max_concurrency":256}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("保存失败：%d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		RestartRequired []string        `json:"restart_required"`
		Settings        RuntimeSettings `json:"settings"`
	}
	if err := jsonutil.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(resp.RestartRequired, ",")
	for _, key := range []string{"proxy_url", "cache_enabled", "outbound_max_concurrency"} {
		if !strings.Contains(got, key) {
			t.Fatalf("restart_required = %v，期望包含 %s", resp.RestartRequired, key)
		}
	}
	if resp.Settings.WebTokenSecret != "" {
		t.Fatal("响应泄露了签名密钥")
	}
	if got := config.GetCacheTTLMinutes(); got != 90 {
		t.Fatalf("TTL 未立即生效：%d", got)
	}
	if got := config.GetDefaultConcurrency(); got != 64 {
		t.Fatalf("默认并发未立即生效：%d", got)
	}
	// 缓存开关与出口并发需要重启，保存后当前运行值不应被改写。
	if !config.AppConfig.CacheEnabled {
		t.Fatal("缓存开关在未重启时被提前改写")
	}
	if got := config.GetOutboundMaxConcurrency(); got != config.DefaultOutboundMaxConcurrency {
		t.Fatalf("出口并发在未重启时被提前改写：%d", got)
	}
}
