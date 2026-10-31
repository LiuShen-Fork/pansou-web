package webui

import "io/fs"

// 发布版二进制会把前端 dist 嵌入进来；Docker 版仍由 nginx 提供静态文件，
// 这里保持为空，不影响现有部署。
var embedded fs.FS

// FS 返回内嵌前端文件系统；未启用 release_frontend 构建标签时为 nil。
func FS() fs.FS { return embedded }
