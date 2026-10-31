package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadStandaloneDefaults 读取二进制发行包自带的 upstream-defaults.env。
// Docker 版由 start.sh 负责，这里只在环境变量为空时补默认值，不覆盖显式配置。
func loadStandaloneDefaults() {
	candidates := []string{"upstream-defaults.env"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "upstream-defaults.env"))
	}
	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		values := map[string]string{}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			values[strings.TrimSpace(key)] = value
		}
		f.Close()
		if v := values["UPSTREAM_CHANNELS"]; v != "" && os.Getenv("CHANNELS") == "" {
			_ = os.Setenv("CHANNELS", v)
		}
		if v := values["UPSTREAM_ENABLED_PLUGINS"]; v != "" && os.Getenv("ENABLED_PLUGINS") == "" {
			_ = os.Setenv("ENABLED_PLUGINS", v)
		}
		return
	}
}
