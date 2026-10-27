package admin

import "github.com/gin-gonic/gin"

// RegisterRoutes 挂载认证、用户中心与管理员接口。
// 必须在初始化 gin 路由时、注册业务路由前后均可调用；
// AttachUser / AccessGate 需要在 api.SetupRouter 中作为全局中间件提前挂载。
func RegisterRoutes(r *gin.Engine) {
	// 公开接口
	r.GET("/api/auth/public-config", PublicConfig)
	r.GET("/api/web/token", WebTokenIssue)
	r.POST("/api/auth/login", Login)
	r.POST("/api/auth/register", Register)
	r.POST("/api/auth/register/send-code", SendRegisterCode)
	r.GET("/api/auth/oauth/callback", OAuthCallback)
	r.GET("/api/auth/oauth/:slug/start", OAuthStart)

	// 登录用户
	auth := r.Group("/api/auth")
	auth.Use(RequireAuth())
	{
		auth.GET("/me", Me)
		auth.POST("/logout", Logout)
		auth.POST("/change-password", ChangePassword)
		auth.POST("/profile", UpdateProfile)
		auth.GET("/token", APITokenGet)
		auth.POST("/token/reset", APITokenReset)
		auth.GET("/tokens", APITokenGet)
		auth.POST("/tokens", APITokenCreate)
		auth.DELETE("/tokens/:id", APITokenDelete)
		auth.GET("/oauth/bindings", OAuthBindings)
		auth.DELETE("/oauth/bindings", UnbindOAuth)
	}

	// 管理员
	admin := r.Group("/api/admin")
	admin.Use(RequireAuth(), RequireAdmin())
	{
		admin.GET("/overview", AdminOverview)
		admin.GET("/stats", AdminStats)
		admin.GET("/settings", AdminGetSettings)
		admin.PUT("/settings", AdminPutSettings)
		admin.POST("/settings/email/test", AdminTestEmail)

		admin.GET("/users", AdminListUsers)
		admin.POST("/users", AdminCreateUser)
		admin.PATCH("/users/:id", AdminUpdateUser)
		admin.DELETE("/users/:id", AdminDeleteUser)

		admin.GET("/search-logs", AdminSearchLogs)

		admin.GET("/upstream-sync", AdminGetUpstreamSync)
		admin.PUT("/upstream-sync", AdminPutUpstreamSync)
		admin.POST("/upstream-sync/run", AdminRunUpstreamSync)

		admin.GET("/oauth-provider", AdminGetOAuthProvider)
		admin.PUT("/oauth-provider", AdminPutOAuthProvider)
		admin.GET("/oauth-providers", AdminListOAuthProviders)
		admin.POST("/oauth-providers", AdminSaveOAuthProvider)
		admin.PUT("/oauth-providers", AdminSaveOAuthProvider)
		admin.DELETE("/oauth-providers/:id", AdminDeleteOAuthProvider)
	}
}
