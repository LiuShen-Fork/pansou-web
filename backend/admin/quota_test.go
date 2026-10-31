package admin

import (
	"testing"
	"time"
)

func TestDefaultQuotaValues(t *testing.T) {
	s := DefaultSettings()
	if s.DefaultWebDailyLimit != 100 || s.DefaultWebRateLimit != 10 {
		t.Fatalf("网页默认额度错误: %+v", s)
	}
	if s.DefaultAPIDailyLimit != 100 || s.DefaultAPIRateLimit != 10 {
		t.Fatalf("API 默认额度错误: %+v", s)
	}
	if s.AnonymousDailyLimit != 20 || s.AnonymousRateLimit != 4 || s.AnonymousGlobalDailyLimit != 200 {
		t.Fatalf("匿名默认额度错误: %+v", s)
	}
	if s.QuotaDefaultsVersion != 1 {
		t.Fatalf("配额版本 = %d，期望 1", s.QuotaDefaultsVersion)
	}
}

func TestNormalizeQuotaMigratesOldSettings(t *testing.T) {
	s := Settings{QuotaDefaultsVersion: 0, AnonymousDailyLimit: 0, AnonymousGlobalDailyLimit: 0}
	normalizeQuotaSettings(&s)
	if s.QuotaDefaultsVersion != 1 {
		t.Fatal("迁移后版本未更新")
	}
	if s.AnonymousDailyLimit != 20 || s.AnonymousRateLimit != 4 || s.AnonymousGlobalDailyLimit != 200 {
		t.Fatalf("匿名额度未迁移: %+v", s)
	}
	if s.DefaultWebDailyLimit != 100 || s.DefaultAPIDailyLimit != 100 ||
		s.DefaultWebRateLimit != 10 || s.DefaultAPIRateLimit != 10 {
		t.Fatalf("登录用户默认额度未迁移: %+v", s)
	}
}

func TestEffectiveQuotaUsesPerUserOverride(t *testing.T) {
	s := DefaultSettings()
	u := &User{
		WebDailySearchLimit: 7,
		APIDailySearchLimit: 9,
		WebRateLimitPerMin:  3,
		APIRateLimitPerMin:  4,
	}
	if got := EffectiveWebDailyLimit(u, s); got != 7 {
		t.Fatalf("网页每日额度 = %d，期望 7", got)
	}
	if got := EffectiveAPIDailyLimit(u, s); got != 9 {
		t.Fatalf("API 每日额度 = %d，期望 9", got)
	}
	if got := EffectiveWebRateLimit(u, s); got != 3 {
		t.Fatalf("网页限频 = %d，期望 3", got)
	}
	if got := EffectiveAPIRateLimit(u, s); got != 4 {
		t.Fatalf("API 限频 = %d，期望 4", got)
	}
	if got := EffectiveWebDailyLimit(&User{}, s); got != 100 {
		t.Fatalf("未配置时应跟随默认: %d", got)
	}
}

func TestQuotaDayBoundaryUsesLocalMidnight(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 30, 0, 0, time.Local)

	start := localDayStart(now)
	if want := time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local).UTC(); !start.Equal(want) {
		t.Fatalf("今日起点 = %s，期望 %s", start.Format(time.RFC3339), want.Format(time.RFC3339))
	}
	next := nextLocalDayStart(now)
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.Local).UTC(); !next.Equal(want) {
		t.Fatalf("明日起点 = %s，期望 %s", next.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestValidateGeoIPSettingsRequiresIPv4(t *testing.T) {
	if err := ValidateGeoIPSettings(Settings{IPGeoEnabled: true}); err == nil {
		t.Fatal("开启 GeoIP 且未配置 IPv4 文件时应报错")
	}
	if err := ValidateGeoIPSettings(Settings{IPGeoEnabled: true, IPGeoIPv4Path: "/not/exist.mmdb"}); err == nil {
		t.Fatal("不存在的 IPv4 文件应报错")
	}
	if err := ValidateGeoIPSettings(Settings{IPGeoEnabled: false}); err != nil {
		t.Fatalf("未开启时不应校验文件: %v", err)
	}
}

func TestAuditLogLifecycle(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	RecordAudit(nil, "quota_exceeded", "anonymous_ip_daily", "测试配额触发", "198.51.100.9", "test-agent")
	list, total, err := ListAuditLogs(1, 10, "", "quota_exceeded")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("审计日志数量 = %d/%d，期望 1", len(list), total)
	}
	if list[0].Event != "quota_exceeded" || list[0].Detail != "测试配额触发" {
		t.Fatalf("审计内容错误: %+v", list[0])
	}
}
