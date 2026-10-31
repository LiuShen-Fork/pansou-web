package admin

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// GeoInfo 是写入搜索/审计日志的 IP 归属地。未启用或查不到时字段为空。
type GeoInfo struct {
	Country string
	Region  string
	City    string
}

type geoState struct {
	sync.RWMutex
	enabled bool
	v4      *maxminddb.Reader
	v6      *maxminddb.Reader
}

var ipGeo geoState

type maxmindRecord struct {
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

// ValidateGeoIPSettings 在保存设置前检查文件是否存在且可被 MMDB 读取。
func ValidateGeoIPSettings(s Settings) error {
	if !s.IPGeoEnabled {
		return nil
	}
	v4 := strings.TrimSpace(s.IPGeoIPv4Path)
	v6 := strings.TrimSpace(s.IPGeoIPv6Path)
	if v4 == "" {
		return fmt.Errorf("开启 IP 归属地后必须配置 IPv4 数据库文件")
	}
	reader, err := maxminddb.Open(v4)
	if err != nil {
		return fmt.Errorf("无法读取 IPv4 数据库：%w", err)
	}
	reader.Close()
	if v6 != "" {
		reader, err := maxminddb.Open(v6)
		if err != nil {
			return fmt.Errorf("无法读取 IPv6 数据库：%w", err)
		}
		reader.Close()
	}
	return nil
}

// ReloadGeoIP 按当前设置加载/卸载 MMDB。失败时保留旧实例。
func ReloadGeoIP() error {
	s := LoadSettings()
	if !s.IPGeoEnabled {
		ipGeo.Lock()
		ipGeo.enabled = false
		ipGeo.v4 = nil
		ipGeo.v6 = nil
		ipGeo.Unlock()
		return nil
	}
	if err := ValidateGeoIPSettings(s); err != nil {
		return err
	}
	v4, err := maxminddb.Open(strings.TrimSpace(s.IPGeoIPv4Path))
	if err != nil {
		return err
	}
	var v6 *maxminddb.Reader
	if path := strings.TrimSpace(s.IPGeoIPv6Path); path != "" {
		v6, err = maxminddb.Open(path)
		if err != nil {
			v4.Close()
			return err
		}
	}
	ipGeo.Lock()
	ipGeo.enabled = true
	ipGeo.v4 = v4
	ipGeo.v6 = v6
	ipGeo.Unlock()
	return nil
}

// LookupIP 返回 IP 的归属地；私有地址、非法地址或未启用时返回空。
func LookupIP(raw string) GeoInfo {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return GeoInfo{}
	}
	ipGeo.RLock()
	enabled := ipGeo.enabled
	reader := ipGeo.v4
	if ip.To4() == nil && ipGeo.v6 != nil {
		reader = ipGeo.v6
	}
	ipGeo.RUnlock()
	if !enabled || reader == nil {
		return GeoInfo{}
	}
	var rec maxmindRecord
	if err := reader.Lookup(ip, &rec); err != nil {
		return GeoInfo{}
	}
	info := GeoInfo{
		Country: localized(rec.Country.Names, rec.Country.ISOCode),
		City:    localized(rec.City.Names, ""),
	}
	if len(rec.Subdivisions) > 0 {
		info.Region = localized(rec.Subdivisions[0].Names, "")
	}
	return info
}

func localized(names map[string]string, fallback string) string {
	for _, key := range []string{"zh-CN", "zh", "en"} {
		if v := strings.TrimSpace(names[key]); v != "" {
			return v
		}
	}
	return strings.TrimSpace(fallback)
}

// GeoIPStatus 供管理后台展示当前加载状态。
func GeoIPStatus() map[string]any {
	ipGeo.RLock()
	defer ipGeo.RUnlock()
	return map[string]any{
		"enabled":     ipGeo.enabled,
		"ipv4_loaded": ipGeo.v4 != nil,
		"ipv6_loaded": ipGeo.v6 != nil,
	}
}
