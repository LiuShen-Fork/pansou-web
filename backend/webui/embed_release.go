//go:build release_frontend

package webui

import (
	"embed"
	"io/fs"
)

// 构建工作流会先把 frontend/dist 复制到 backend/webui/dist，再使用
// -tags release_frontend 编译，得到一个自带前端的单文件二进制。
//go:embed all:dist
var dist embed.FS

func init() {
	sub, err := fs.Sub(dist, "dist")
	if err == nil {
		embedded = sub
	}
}
