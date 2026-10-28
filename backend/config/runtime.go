package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// DefaultOutboundMaxConcurrency 是出口并发总闸的默认值。
const DefaultOutboundMaxConcurrency = 128

// 后台运行参数的运行时覆盖层。
//
// 这些值由管理后台写入数据库，并在启动时覆盖环境变量；其中代理、缓存 TTL
// 与默认并发支持保存后热更新。环境变量仍作为首次启动/无后台记录时的默认值。
// 用原子值/读写锁保护，避免后台保存时与正在进行的搜索发生数据竞争。
var (
	cacheTTLOverride           atomic.Int64
	defaultConcurrencyOverride atomic.Int64
	concurrencyAdminManaged    atomic.Bool

	proxyOverrideMu  sync.RWMutex
	proxyOverride    string
	proxyOverrideSet bool
)

// GetCacheTTLMinutes 返回当前生效的缓存 TTL（分钟）。
func GetCacheTTLMinutes() int {
	if v := cacheTTLOverride.Load(); v > 0 {
		return int(v)
	}
	if AppConfig == nil {
		return 60
	}
	return AppConfig.CacheTTLMinutes
}

// SetCacheTTLMinutes 热更新缓存 TTL（分钟）。<=0 时回到配置文件/环境变量值。
func SetCacheTTLMinutes(minutes int) {
	if minutes <= 0 {
		cacheTTLOverride.Store(0)
		return
	}
	cacheTTLOverride.Store(int64(minutes))
}

// GetDefaultConcurrency 返回当前生效的默认搜索并发数。
func GetDefaultConcurrency() int {
	if v := defaultConcurrencyOverride.Load(); v > 0 {
		return int(v)
	}
	if AppConfig == nil {
		return 1
	}
	return AppConfig.DefaultConcurrency
}

// SetDefaultConcurrency 热更新默认搜索并发数。<=0 表示由后台接管并恢复自动计算，
// 此时环境变量 CONCURRENCY 不再生效。
func SetDefaultConcurrency(concurrency int) {
	concurrencyAdminManaged.Store(true)
	if concurrency <= 0 {
		defaultConcurrencyOverride.Store(0)
		return
	}
	defaultConcurrencyOverride.Store(int64(concurrency))
}

// DefaultConcurrencyOverridden 返回是否由后台显式指定了并发数。
func DefaultConcurrencyOverridden() bool {
	return defaultConcurrencyOverride.Load() > 0
}

// DefaultConcurrencyAdminManaged 返回并发数是否已由后台接管。
func DefaultConcurrencyAdminManaged() bool {
	return concurrencyAdminManaged.Load()
}

// ExplicitDefaultConcurrency 返回后台或环境变量显式指定的并发数；0 表示自动。
func ExplicitDefaultConcurrency() int {
	if concurrencyAdminManaged.Load() {
		return int(defaultConcurrencyOverride.Load())
	}
	if raw := strings.TrimSpace(os.Getenv("CONCURRENCY")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// GetOutboundMaxConcurrency 返回当前生效的出口并发上限（0 配置时使用默认值）。
func GetOutboundMaxConcurrency() int {
	if AppConfig != nil && AppConfig.OutboundMaxConcurrency > 0 {
		return AppConfig.OutboundMaxConcurrency
	}
	return DefaultOutboundMaxConcurrency
}

// GetProxyURL 返回当前生效的出站代理地址。
func GetProxyURL() string {
	proxyOverrideMu.RLock()
	if proxyOverrideSet {
		v := proxyOverride
		proxyOverrideMu.RUnlock()
		return v
	}
	proxyOverrideMu.RUnlock()
	if AppConfig == nil {
		return ""
	}
	return AppConfig.ProxyURL
}

// SetProxyURL 热更新出站代理地址；空字符串表示明确不使用代理。
func SetProxyURL(proxyURL string) {
	proxyOverrideMu.Lock()
	proxyOverride = proxyURL
	proxyOverrideSet = true
	proxyOverrideMu.Unlock()
}
