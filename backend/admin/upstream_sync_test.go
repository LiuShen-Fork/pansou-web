package admin

import (
	"reflect"
	"testing"
)

func TestParseUpstreamDefaults(t *testing.T) {
	dockerfile := `
ENV CACHE_PATH=/app/cache \
    CHANNELS=tgsearchers7,Aliyun_4K_Movies,yunpanx, \
    ENABLED_PLUGINS=pansearch,qqpd,weibo
`
	channels, plugins, err := ParseUpstreamDefaults(dockerfile)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if want := []string{"tgsearchers7", "Aliyun_4K_Movies", "yunpanx"}; !reflect.DeepEqual(channels, want) {
		t.Errorf("频道 = %v, 期望 %v", channels, want)
	}
	if want := []string{"pansearch", "qqpd", "weibo"}; !reflect.DeepEqual(plugins, want) {
		t.Errorf("插件 = %v, 期望 %v", plugins, want)
	}
}

func TestParseUpstreamDefaultsRejectsUnsafeTokens(t *testing.T) {
	dockerfile := "ENV CHANNELS='ok,bad$token,ok,../etc' ENABLED_PLUGINS='good,$(evil)'"
	channels, plugins, err := ParseUpstreamDefaults(dockerfile)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if want := []string{"ok"}; !reflect.DeepEqual(channels, want) {
		t.Errorf("频道 = %v, 期望 %v", channels, want)
	}
	if want := []string{"good"}; !reflect.DeepEqual(plugins, want) {
		t.Errorf("插件 = %v, 期望 %v", plugins, want)
	}
}

func TestSplitSafeDeduplicates(t *testing.T) {
	got := splitSafe("a,b,a, ,c")
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("splitSafe = %v, 期望 %v", got, want)
	}
}

func TestSanitizeUsername(t *testing.T) {
	if got := sanitizeUsername("admin"); got == "admin" {
		t.Errorf("保留用户名应被改写，实际 %q", got)
	}
	if got := sanitizeUsername("a b/c"); got != "abc" {
		t.Errorf("非法字符应被移除，实际 %q", got)
	}
	if got := sanitizeUsername(""); len(got) < 2 {
		t.Errorf("空用户名应生成兜底值，实际 %q", got)
	}
}
