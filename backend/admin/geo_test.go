package admin

import "testing"

func TestParseIP2Region(t *testing.T) {
	tests := []struct {
		raw  string
		want GeoInfo
	}{
		{"中国|0|广东省|深圳市|电信", GeoInfo{Country: "中国", Region: "广东省", City: "深圳市"}},
		{"美国|0|加利福尼亚|洛杉矶|0", GeoInfo{Country: "美国", Region: "加利福尼亚", City: "洛杉矶"}},
		{"中国|0|0|0|0", GeoInfo{Country: "中国"}},
		{"0|0|0|0|0", GeoInfo{}},
		{"", GeoInfo{}},
	}
	for _, tt := range tests {
		if got := parseIP2Region(tt.raw); got != tt.want {
			t.Fatalf("parseIP2Region(%q) = %+v，期望 %+v", tt.raw, got, tt.want)
		}
	}
}
