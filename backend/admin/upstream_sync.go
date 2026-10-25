package admin

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"pansou/util/json"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"pansou/config"
)

const (
	settingKeyUpstreamSync       = "upstream_sync"
	settingKeyUpstreamSyncStatus = "upstream_sync_status"
)

var (
	upstreamSyncMu      sync.Mutex
	upstreamSyncRunning bool
)

type UpstreamSyncSettings struct {
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"interval_hours"`
	Repo          string `json:"repo"`
	Ref           string `json:"ref"`
}

func DefaultUpstreamSyncSettings() UpstreamSyncSettings {
	return UpstreamSyncSettings{
		Enabled:       true,
		IntervalHours: 24,
		Repo:          "fish2018/pansou",
		Ref:           "main",
	}
}

func LoadUpstreamSyncSettings() UpstreamSyncSettings {
	s := DefaultUpstreamSyncSettings()
	if err := GetSetting(settingKeyUpstreamSync, &s); err != nil {
		return DefaultUpstreamSyncSettings()
	}
	if s.IntervalHours <= 0 {
		s.IntervalHours = 24
	}
	if s.Repo == "" {
		s.Repo = "fish2018/pansou"
	}
	if s.Ref == "" {
		s.Ref = "main"
	}
	return s
}

func SaveUpstreamSyncSettings(s UpstreamSyncSettings) error {
	if err := SetSetting(settingKeyUpstreamSync, s); err != nil {
		return err
	}
	return nil
}

type UpstreamSyncStatus struct {
	LastCheckAt     string `json:"last_check_at"`
	LastSuccessAt   string `json:"last_success_at"`
	LastRevision    string `json:"last_revision"`
	LastError       string `json:"last_error"`
	ChannelCount    int    `json:"channel_count"`
	PluginCount     int    `json:"plugin_count"`
	ChannelsChanged bool   `json:"channels_changed"`
	PluginsChanged  bool   `json:"plugins_changed"`
	RestartRequired bool   `json:"restart_required"`
}

func LoadUpstreamSyncStatus() UpstreamSyncStatus {
	var st UpstreamSyncStatus
	_ = GetSetting(settingKeyUpstreamSyncStatus, &st)
	return st
}

func saveUpstreamSyncStatus(st UpstreamSyncStatus) {
	_ = SetSetting(settingKeyUpstreamSyncStatus, st)
}

var upstreamEnvPattern = regexp.MustCompile(`(?:^|\s)(CHANNELS|ENABLED_PLUGINS)=([^\s]+)`)
var safeTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var safeRepoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)
var safeRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// ParseUpstreamDefaults 从上游 Dockerfile 中解析 CHANNELS 与 ENABLED_PLUGINS。
func ParseUpstreamDefaults(dockerfile string) (channels, plugins []string, err error) {
	flat := strings.ReplaceAll(dockerfile, "\\\r\n", " ")
	flat = strings.ReplaceAll(flat, "\\\n", " ")
	values := map[string]string{}
	for _, m := range upstreamEnvPattern.FindAllStringSubmatch(flat, -1) {
		values[m[1]] = strings.Trim(m[2], `"'`)
	}
	if values["CHANNELS"] == "" {
		return nil, nil, fmt.Errorf("上游 Dockerfile 未找到 CHANNELS")
	}
	if values["ENABLED_PLUGINS"] == "" {
		return nil, nil, fmt.Errorf("上游 Dockerfile 未找到 ENABLED_PLUGINS")
	}
	channels = splitSafe(values["CHANNELS"])
	plugins = splitSafe(values["ENABLED_PLUGINS"])
	if len(channels) == 0 || len(plugins) == 0 {
		return nil, nil, fmt.Errorf("上游默认值为空")
	}
	return channels, plugins, nil
}

func splitSafe(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || !safeTokenPattern.MatchString(p) || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func upstreamDockerfileURLs(repo, ref string) []string {
	if override := strings.TrimSpace(os.Getenv("UPSTREAM_DOCKERFILE_URL")); override != "" {
		return []string{override}
	}
	// raw 优先；国内网络经常连不上 raw.githubusercontent.com，回退到 jsDelivr。
	return []string{
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/Dockerfile", repo, ref),
		fmt.Sprintf("https://cdn.jsdelivr.net/gh/%s@%s/Dockerfile", repo, ref),
	}
}

func upstreamHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// 标准 HTTPS_PROXY 已由 DefaultTransport 处理；这里额外兼容项目常用的 PROXY
	// 变量（仅 http/https 形式，socks5 交给部署层的全局代理）。
	if proxy := strings.TrimSpace(os.Getenv("PROXY")); strings.HasPrefix(proxy, "http://") || strings.HasPrefix(proxy, "https://") {
		if u, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}

func fetchUpstreamDockerfile(ctx context.Context, repo, ref string) (string, error) {
	client := upstreamHTTPClient(25 * time.Second)
	var errs []string
	for _, endpoint := range upstreamDockerfileURLs(repo, ref) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		req.Header.Set("User-Agent", "pansou-web-upstream-sync")
		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", endpoint, err))
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			errs = append(errs, fmt.Sprintf("%s: HTTP %d", endpoint, resp.StatusCode))
			continue
		}
		if readErr != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", endpoint, readErr))
			continue
		}
		return string(body), nil
	}
	return "", fmt.Errorf("所有上游地址均不可用：%s", strings.Join(errs, "; "))
}

func fetchUpstreamRevision(ctx context.Context, repo, ref string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pansou-web-upstream-sync")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var data struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return ""
	}
	return data.SHA
}

func upstreamDefaultsOutputPath() string {
	if v := strings.TrimSpace(os.Getenv("UPSTREAM_DEFAULTS_OUTPUT")); v != "" {
		return v
	}
	if _, err := os.Stat("/app/data"); err == nil {
		return "/app/data/upstream-defaults.env"
	}
	return filepath.Join("data", "upstream-defaults.env")
}

func writeUpstreamDefaultsFile(settings UpstreamSyncSettings, revision string, channels, plugins []string) error {
	path := upstreamDefaultsOutputPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join([]string{
		"# 由 PanSou 管理后台上游自动同步生成，请勿手工编辑。",
		fmt.Sprintf("# 来源: https://github.com/%s/blob/%s/Dockerfile", settings.Repo, firstNonEmpty(revision, settings.Ref)),
		fmt.Sprintf("# 生成时间: %s", Now()),
		fmt.Sprintf("UPSTREAM_REVISION='%s'", firstNonEmpty(revision, settings.Ref)),
		fmt.Sprintf("UPSTREAM_CHANNELS='%s'", strings.Join(channels, ",")),
		fmt.Sprintf("UPSTREAM_ENABLED_PLUGINS='%s'", strings.Join(plugins, ",")),
		"",
	}, "\n")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// RunUpstreamSync 立即抓取上游 Dockerfile 并应用默认频道/插件。
// 频道会热更新；插件列表写入持久化文件，重启后生效（已编译进二进制的插件才会真正启用）。
func RunUpstreamSync(force bool) (UpstreamSyncStatus, error) {
	upstreamSyncMu.Lock()
	if upstreamSyncRunning {
		upstreamSyncMu.Unlock()
		return LoadUpstreamSyncStatus(), fmt.Errorf("同步正在进行中")
	}
	upstreamSyncRunning = true
	upstreamSyncMu.Unlock()
	defer func() {
		upstreamSyncMu.Lock()
		upstreamSyncRunning = false
		upstreamSyncMu.Unlock()
	}()

	settings := LoadUpstreamSyncSettings()
	if !settings.Enabled && !force {
		return LoadUpstreamSyncStatus(), fmt.Errorf("上游自动同步未启用")
	}
	st := LoadUpstreamSyncStatus()
	st.LastCheckAt = Now()
	st.LastError = ""

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	dockerfile, err := fetchUpstreamDockerfile(ctx, settings.Repo, settings.Ref)
	if err != nil {
		st.LastError = err.Error()
		saveUpstreamSyncStatus(st)
		return st, err
	}
	channels, plugins, err := ParseUpstreamDefaults(dockerfile)
	if err != nil {
		st.LastError = err.Error()
		saveUpstreamSyncStatus(st)
		return st, err
	}

	revision := fetchUpstreamRevision(ctx, settings.Repo, settings.Ref)
	currentChannels := config.GetDefaultChannels()
	currentPlugins := []string{}
	if config.AppConfig != nil {
		currentPlugins = config.AppConfig.EnabledPlugins
	}
	st.ChannelCount = len(channels)
	st.PluginCount = len(plugins)
	st.ChannelsChanged = !sameStrings(currentChannels, channels)
	st.PluginsChanged = !sameStrings(currentPlugins, plugins)
	st.RestartRequired = st.PluginsChanged
	st.LastRevision = firstNonEmpty(revision, settings.Ref)

	if st.ChannelsChanged {
		config.SetDefaultChannels(channels)
	}
	if st.ChannelsChanged || st.PluginsChanged {
		if err := writeUpstreamDefaultsFile(settings, revision, channels, plugins); err != nil {
			st.LastError = "写入默认值文件失败: " + err.Error()
		}
	}
	if st.LastError == "" {
		st.LastSuccessAt = Now()
	}
	saveUpstreamSyncStatus(st)
	log.Printf("[admin] 上游同步完成：频道 %d 个（变化=%v），插件 %d 个（变化=%v），重启后生效",
		len(channels), st.ChannelsChanged, len(plugins), st.PluginsChanged)
	return st, nil
}

// StartUpstreamSyncScheduler 启动后台定时同步。首次启动 30 秒后检查一次。
func StartUpstreamSyncScheduler() {
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		<-timer.C
		for {
			settings := LoadUpstreamSyncSettings()
			if settings.Enabled {
				st := LoadUpstreamSyncStatus()
				interval := time.Duration(settings.IntervalHours) * time.Hour
				if st.LastError != "" && interval > time.Hour {
					// 失败后最多每小时重试一次，避免配置错误被拖到一整天。
					interval = time.Hour
				}
				due := true
				if t, err := time.Parse(time.RFC3339, st.LastCheckAt); err == nil {
					due = time.Since(t) >= interval
				}
				if due {
					if _, err := RunUpstreamSync(false); err != nil {
						log.Printf("[admin] 上游自动同步失败: %v", err)
					}
				}
			}
			time.Sleep(10 * time.Minute)
		}
	}()
}

// ---------------- Admin API ----------------

func AdminGetUpstreamSync(c *gin.Context) {
	settings := LoadUpstreamSyncSettings()
	st := LoadUpstreamSyncStatus()
	next := ""
	if t, err := time.Parse(time.RFC3339, st.LastCheckAt); err == nil {
		next = t.Add(time.Duration(settings.IntervalHours) * time.Hour).Format(time.RFC3339)
	}
	upstreamSyncMu.Lock()
	running := upstreamSyncRunning
	upstreamSyncMu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"settings":          settings,
		"status":            st,
		"running":           running,
		"next_check_at":     next,
		"channels":          config.GetDefaultChannels(),
		"enabled_plugins":   enabledPluginsSnapshot(),
		"defaults_env_path": upstreamDefaultsOutputPath(),
	})
}

func enabledPluginsSnapshot() []string {
	if config.AppConfig == nil {
		return nil
	}
	out := make([]string, len(config.AppConfig.EnabledPlugins))
	copy(out, config.AppConfig.EnabledPlugins)
	return out
}

func AdminPutUpstreamSync(c *gin.Context) {
	var body UpstreamSyncSettings
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if body.IntervalHours < 1 || body.IntervalHours > 720 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "同步间隔需在 1-720 小时之间", "code": "BAD_INTERVAL"})
		return
	}
	if !safeRepoPattern.MatchString(body.Repo) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仓库格式应为 owner/name", "code": "BAD_REPO"})
		return
	}
	if body.Ref == "" {
		body.Ref = "main"
	}
	if !safeRefPattern.MatchString(body.Ref) || strings.Contains(body.Ref, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分支名包含非法字符", "code": "BAD_REF"})
		return
	}
	if err := SaveUpstreamSyncSettings(body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AdminRunUpstreamSync(c *gin.Context) {
	st, err := RunUpstreamSync(true)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "status": st})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "status": st})
}
