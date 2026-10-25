package api

import (
	"github.com/gin-gonic/gin"
	"pansou/admin"
	"pansou/config"
	"pansou/plugin"
	"pansou/service"
	"pansou/util"
)

// SetupRouter 设置路由
func SetupRouter(searchService *service.SearchService) *gin.Engine {
	// 设置搜索服务
	SetSearchService(searchService)

	// 设置为生产模式
	gin.SetMode(gin.ReleaseMode)

	// 创建默认路由
	r := gin.Default()

	// 添加中间件
	r.Use(CORSMiddleware())
	r.Use(LoggerMiddleware())
	r.Use(util.GzipMiddleware()) // 添加压缩中间件

	// 账户体系与访问控制。AttachUser 只识别身份，AccessGate 负责按后台开关
	// 拦截搜索/链接检测/插件 Web 接口；旧版 AUTH_USERS 认证已由后台取代。
	r.Use(admin.AttachUser())
	r.Use(admin.CSRFGuard())
	r.Use(admin.AccessGate())

	// 定义API路由组
	api := r.Group("/api")
	{
		// 搜索接口 - 支持POST和GET两种方式
		api.POST("/search", SearchHandler)
		api.GET("/search", SearchHandler) // 添加GET方式支持
		// 网页端匿名搜索入口：受后台「前端直接搜索」开关控制，
		// 与需要登录的 /api/search 分开，避免把公开网页搜索变成 API 后门。
		api.POST("/web/search", SearchHandler)
		api.GET("/web/search", SearchHandler)
		api.POST("/check/links", CheckHandler)

		// 健康检查接口
		api.GET("/health", func(c *gin.Context) {
			// 根据配置决定是否返回插件信息
			pluginCount := 0
			pluginNames := []string{}
			pluginsEnabled := config.AppConfig.AsyncPluginEnabled

			if pluginsEnabled && searchService != nil && searchService.GetPluginManager() != nil {
				plugins := searchService.GetPluginManager().GetPlugins()
				pluginCount = len(plugins)
				for _, p := range plugins {
					pluginNames = append(pluginNames, p.Name())
				}
			}

			// 获取频道信息
			channels := config.AppConfig.DefaultChannels
			channelsCount := len(channels)

			adminSettings := admin.LoadSettings()
			response := gin.H{
				"status":                 "ok",
				"auth_enabled":           true, // 新账户体系始终启用
				"admin_enabled":          admin.Enabled(),
				"api_require_login":      adminSettings.APIRequireLogin,
				"web_search_enabled":     adminSettings.WebSearchEnabled,
				"email_register_enabled": adminSettings.EmailRegisterEnabled,
				"plugins_enabled":        pluginsEnabled,
				"channels":               channels,
				"channels_count":         channelsCount,
				// 存活观测：累积每轮产出/报错，一眼看出哪些插件与频道是失效的。
				// 只报事实不做淘汰——窗口内零产出不代表无数据（内容仍会经后台补齐进缓存），
				// 是否停用由部署方按这里的名单决定。
				"liveness": service.LivenessSnapshot(),
				// TG 可达性：被墙时 TG 阶段会被直接跳过，这里给出结论、原因与探测时间。
				"tg": service.TGReachabilitySnapshot(),
			}

			// 只有当插件启用时才返回插件相关信息
			if pluginsEnabled {
				response["plugin_count"] = pluginCount
				response["plugins"] = pluginNames
			}

			c.JSON(200, response)
		})
	}

	// 账户 / 用户中心 / 管理后台接口
	admin.RegisterRoutes(r)

	// 注册插件的Web路由（如果插件实现了PluginWithWebHandler接口）
	// 只有当插件功能启用且插件在启用列表中时才注册路由
	if config.AppConfig.AsyncPluginEnabled && searchService != nil && searchService.GetPluginManager() != nil {
		enabledPlugins := searchService.GetPluginManager().GetPlugins()
		for _, p := range enabledPlugins {
			if webPlugin, ok := p.(plugin.PluginWithWebHandler); ok {
				webPlugin.RegisterWebRoutes(r.Group(""))
			}
		}
	}

	return r
}
