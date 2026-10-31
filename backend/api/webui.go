package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"pansou/webui"
)

// setupWebUI 为「自带前端的单文件二进制」提供静态文件与 SPA 回退。
// Docker 镜像由 nginx 提供前端，webui.FS() 为 nil，不会注册任何处理器。
func setupWebUI(r *gin.Engine) {
	files := webui.FS()
	if files == nil {
		return
	}
	server := http.FileServer(http.FS(files))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在", "code": "NOT_FOUND"})
			return
		}
		requestPath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if requestPath == "" || requestPath == "." {
			requestPath = "index.html"
		}
		if info, err := fs.Stat(files, requestPath); err == nil && !info.IsDir() {
			server.ServeHTTP(c.Writer, c.Request)
			return
		}
		// 前端使用 hash 路由，未知路径回退到 index.html。
		c.Request.URL.Path = "/"
		server.ServeHTTP(c.Writer, c.Request)
	})
}
