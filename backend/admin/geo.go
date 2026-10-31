package admin

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// GeoInfo 是写入搜索/审计日志的 IP 归属地。未启用或查不到时字段为空。
type GeoInfo struct {
	Country string
	Region  string
	City    string
}

type geoState struct {
	sync.RWMutex
	// xdb.Searcher 的 Search 会更新内部 ioCount，不能并发调用；日志场景用互斥锁串行即可。
	searchMu sync.Mutex
	enabled  bool
	v4       *xdb.Searcher
	v6       *xdb.Searcher
}

var ipGeo geoState

// openXDB 校验版本并把整个 xdb 文件载入内存。
// ip2region 的 NewWithBuffer 搜索器可以安全并发使用；xdb 文件通常只有几十 MB。
func openXDB(path string, wantVersion int, label string) (*xdb.Searcher, error) {
	header, err := xdb.LoadHeaderFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取 %s xdb 文件：%w", label, err)
	}
	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		return nil, fmt.Errorf("%s xdb 文件版本无效：%w", label, err)
	}
	if version.Id != wantVersion {
		return nil, fmt.Errorf("%s 数据库版本不匹配：需要 IPv%d xdb", label, wantVersion)
	}
	if err := xdb.VerifyFromFile(path); err != nil {
		return nil, fmt.Errorf("%s xdb 文件校验失败：%w", label, err)
	}
	content, err := xdb.LoadContentFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载 %s xdb 文件失败：%w", label, err)
	}
	searcher, err := xdb.NewWithBuffer(version, content)
	if err != nil {
		return nil, fmt.Errorf("初始化 %s xdb 查询失败：%w", label, err)
	}
	return searcher, nil
}

// ValidateGeoIPSettings 在保存设置前检查 xdb 文件是否存在、版本是否匹配。
func ValidateGeoIPSettings(s Settings) error {
	if !s.IPGeoEnabled {
		return nil
	}
	v4 := strings.TrimSpace(s.IPGeoIPv4Path)
	v6 := strings.TrimSpace(s.IPGeoIPv6Path)
	if v4 == "" {
		return fmt.Errorf("开启 IP 归属地后必须配置 IPv4 xdb 文件")
	}
	searcher, err := openXDB(v4, xdb.IPv4VersionNo, "IPv4")
	if err != nil {
		return err
	}
	searcher.Close()
	if v6 != "" {
		searcher, err = openXDB(v6, xdb.IPv6VersionNo, "IPv6")
		if err != nil {
			return err
		}
		searcher.Close()
	}
	return nil
}

// ReloadGeoIP 按当前设置加载/卸载 ip2region xdb。失败时保留旧实例。
func ReloadGeoIP() error {
	s := LoadSettings()
	if !s.IPGeoEnabled {
		ipGeo.Lock()
		old4, old6 := ipGeo.v4, ipGeo.v6
		ipGeo.enabled = false
		ipGeo.v4 = nil
		ipGeo.v6 = nil
		if old4 != nil {
			old4.Close()
		}
		if old6 != nil {
			old6.Close()
		}
		ipGeo.Unlock()
		return nil
	}
	if err := ValidateGeoIPSettings(s); err != nil {
		return err
	}
	v4, err := openXDB(strings.TrimSpace(s.IPGeoIPv4Path), xdb.IPv4VersionNo, "IPv4")
	if err != nil {
		return err
	}
	var v6 *xdb.Searcher
	if path := strings.TrimSpace(s.IPGeoIPv6Path); path != "" {
		v6, err = openXDB(path, xdb.IPv6VersionNo, "IPv6")
		if err != nil {
			v4.Close()
			return err
		}
	}
	ipGeo.Lock()
	old4, old6 := ipGeo.v4, ipGeo.v6
	ipGeo.enabled = true
	ipGeo.v4 = v4
	ipGeo.v6 = v6
	// 持有写锁关闭旧实例：此时不会再有 LookupIP 持有旧指针。
	if old4 != nil {
		old4.Close()
	}
	if old6 != nil {
		old6.Close()
	}
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
	searcher := ipGeo.v4
	if ip.To4() == nil {
		searcher = ipGeo.v6
	}
	if !enabled || searcher == nil {
		ipGeo.RUnlock()
		return GeoInfo{}
	}
	// 整个查询期间持有读锁，ReloadGeoIP 替换并关闭旧 searcher 时不会被并发读取。
	ipGeo.searchMu.Lock()
	region, err := searcher.Search(ip.String())
	ipGeo.searchMu.Unlock()
	ipGeo.RUnlock()
	if err != nil {
		return GeoInfo{}
	}
	return parseIP2Region(region)
}

// parseIP2Region 解析 ip2region 的 "国家|区域|省份|城市|ISP" 格式。
// 未知字段会写成 0，空值不应进入日志。
func parseIP2Region(raw string) GeoInfo {
	parts := strings.Split(strings.TrimSpace(raw), "|")
	field := func(i int) string {
		if i >= len(parts) {
			return ""
		}
		v := strings.TrimSpace(parts[i])
		if v == "" || v == "0" || v == "-" {
			return ""
		}
		return v
	}
	region := field(2)
	if region == "" {
		region = field(1)
	}
	return GeoInfo{
		Country: field(0),
		Region:  region,
		City:    field(3),
	}
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
