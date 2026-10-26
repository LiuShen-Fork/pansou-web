package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type adminSiteSettings struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Slogan       string `json:"slogan"`
	Icon         string `json:"icon"`
	Announcement string `json:"announcement"`
}

type adminSettingsPayload struct {
	Site  adminSiteSettings `json:"site"`
	Auth  Settings          `json:"auth"`
	Email EmailSettings     `json:"email"`
}

// isSafeIconURL 限制图标来源，避免把 javascript: 等危险协议写进页面。
func isSafeIconURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "/") ||
		strings.HasPrefix(lower, "data:image/")
}

func AdminGetSettings(c *gin.Context) {
	s := LoadSettings()
	e := LoadEmailSettings()
	e.Password = ""
	providers, _ := ListOAuthProviders(false)
	for i := range providers {
		providers[i].ClientSecret = ""
	}
	// site 已单独返回名称、Slogan 与图标，auth 中不再重复携带，
	// 避免上传图标后设置页响应体翻倍。
	auth := s
	auth.SiteSlogan = ""
	auth.SiteIcon = ""
	c.JSON(http.StatusOK, gin.H{
		"site": adminSiteSettings{
			Name:         s.SiteName,
			Description:  s.SiteDescription,
			Slogan:       s.SiteSlogan,
			Icon:         s.SiteIcon,
			Announcement: s.Announcement,
		},
		"auth":               auth,
		"email":              e,
		"email_password_set": LoadEmailSettings().Password != "",
		"oauth_providers":    providers,
	})
}

func AdminPutSettings(c *gin.Context) {
	var payload adminSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if strings.TrimSpace(payload.Site.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "站点名称不能为空", "code": "BAD_SITE_NAME"})
		return
	}
	s := payload.Auth
	s.SiteName = strings.TrimSpace(payload.Site.Name)
	s.SiteDescription = strings.TrimSpace(payload.Site.Description)
	s.SiteSlogan = strings.TrimSpace(payload.Site.Slogan)
	if s.SiteSlogan == "" {
		s.SiteSlogan = DefaultSettings().SiteSlogan
	}
	s.SiteIcon = strings.TrimSpace(payload.Site.Icon)
	if s.SiteIcon == "" {
		s.SiteIcon = DefaultSettings().SiteIcon
	}
	// 前端限制原始图片 512KB，Base64 后约 683KB，这里留出余量（外链不受影响）。
	if len(s.SiteIcon) > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "站点图标过大，请使用不超过 512KB 的图片或外链", "code": "BAD_SITE_ICON"})
		return
	}
	if !isSafeIconURL(s.SiteIcon) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "站点图标仅支持 http(s)、站内路径或图片 Data URL", "code": "BAD_SITE_ICON"})
		return
	}
	if n := len([]rune(s.SiteSlogan)); n > 120 {
		s.SiteSlogan = string([]rune(s.SiteSlogan)[:120])
	}
	s.Announcement = payload.Site.Announcement
	if s.DefaultDailySearchLimit < 0 || s.DefaultDailySearchLimit > 100000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "默认每日搜索上限不合法", "code": "BAD_LIMIT"})
		return
	}
	if s.AnonymousWebSearchLimit < 0 || s.AnonymousWebSearchLimit > 100000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "匿名搜索每分钟上限不合法", "code": "BAD_LIMIT"})
		return
	}
	if s.AnonymousDailyLimit < 0 || s.AnonymousDailyLimit > 1000000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "匿名搜索单 IP 每日上限不合法", "code": "BAD_LIMIT"})
		return
	}
	if s.AnonymousGlobalDailyLimit < 0 || s.AnonymousGlobalDailyLimit > 100000000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "匿名搜索全站每日上限不合法", "code": "BAD_LIMIT"})
		return
	}
	if s.SessionTTLHours < 1 || s.SessionTTLHours > 8760 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话有效期需在 1-8760 小时之间", "code": "BAD_SESSION_TTL"})
		return
	}

	oldEmail := LoadEmailSettings()
	e := payload.Email
	if e.Password == "" {
		e.Password = oldEmail.Password
	}
	if e.Port <= 0 {
		e.Port = 587
	}
	if e.Enabled {
		if strings.TrimSpace(e.Host) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "启用 SMTP 时必须填写服务器地址", "code": "BAD_SMTP"})
			return
		}
		from := strings.TrimSpace(e.From)
		if from == "" {
			from = strings.TrimSpace(e.Username)
		}
		if from == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "启用 SMTP 时必须填写发件人地址", "code": "BAD_SMTP"})
			return
		}
		e.From = from
	}
	// 邮箱注册必须配合 SMTP 验证，这是硬性约束。
	if s.EmailRegisterEnabled && (!e.Enabled || strings.TrimSpace(e.Host) == "") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "开启邮箱注册前必须启用并配置 SMTP 邮件验证",
			"code":  "SMTP_REQUIRED",
		})
		return
	}
	if s.OAuthLoginEnabled || s.OAuthRegisterEnabled {
		providers, _ := ListOAuthProviders(true)
		configured := 0
		for _, p := range providers {
			if p.ClientID != "" && p.ClientSecret != "" && p.AuthURL != "" && p.TokenURL != "" && p.UserInfoURL != "" {
				configured++
			}
		}
		if configured == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "开启 OAuth 登录/注册前，请先配置至少一个可用的 OAuth 服务商",
				"code":  "OAUTH_NOT_CONFIGURED",
			})
			return
		}
	}

	if err := SaveSettings(s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存设置失败: " + err.Error()})
		return
	}
	if err := SaveEmailSettings(e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 SMTP 设置失败: " + err.Error()})
		return
	}
	InvalidateSettingsCache()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AdminTestEmail(c *gin.Context) {
	var body struct {
		To string `json:"to"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.To) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写收件邮箱", "code": "BAD_REQUEST"})
		return
	}
	e := LoadEmailSettings()
	if !e.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP 未启用", "code": "SMTP_DISABLED"})
		return
	}
	settings := LoadSettings()
	subject, text := BuildCodeMail(settings.SiteName, "123456", "test")
	subject = settings.SiteName + " - SMTP 测试"
	text = "这是一封来自 " + settings.SiteName + " 的测试邮件，收到即表示 SMTP 配置可用。"
	if err := SendMail(e, strings.TrimSpace(body.To), subject, text); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": "EMAIL_SEND_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AdminOverview(c *gin.Context) {
	o, err := GetOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func AdminStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "14"))
	stats, err := GetStats(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func AdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := ListUsers(c.Query("q"), c.Query("role"), c.Query("status"), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total, "page": page, "page_size": pageSize})
}

func AdminCreateUser(c *gin.Context) {
	var body struct {
		Username         string `json:"username"`
		Email            string `json:"email"`
		Password         string `json:"password"`
		Role             string `json:"role"`
		DailySearchLimit int    `json:"daily_search_limit"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	user, err := CreateUser(body.Username, body.Email, body.Password, body.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "CREATE_FAILED"})
		return
	}
	if body.DailySearchLimit != 0 {
		limit := body.DailySearchLimit
		user, err = UpdateUser(user.ID, UserUpdate{DailySearchLimit: &limit})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "UPDATE_FAILED"})
			return
		}
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func AdminUpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 无效"})
		return
	}
	var upd UserUpdate
	if err := c.ShouldBindJSON(&upd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if current := CurrentUser(c); current != nil && id == current.ID && upd.Status != nil && *upd.Status != StatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能禁用当前登录的管理员账号", "code": "CANNOT_DISABLE_SELF"})
		return
	}
	user, err := UpdateUser(id, upd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "UPDATE_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func AdminDeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户 ID 无效"})
		return
	}
	current := CurrentUser(c)
	if err := DeleteUser(id, current.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "DELETE_FAILED"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AdminSearchLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := ListSearchLogs(page, pageSize, c.Query("q"), c.Query("source"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": list, "total": total, "page": page, "page_size": pageSize})
}

func AdminListOAuthProviders(c *gin.Context) {
	list, err := ListOAuthProviders(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i := range list {
		list[i].ClientSecret = ""
	}
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func AdminSaveOAuthProvider(c *gin.Context) {
	var p OAuthProvider
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效", "code": "BAD_REQUEST"})
		return
	}
	if p.ID > 0 {
		old, err := GetOAuthProviderByID(p.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if strings.TrimSpace(p.ClientSecret) == "" {
			p.ClientSecret = old.ClientSecret
		}
	}
	if p.Enabled {
		missing := ""
		switch {
		case strings.TrimSpace(p.ClientID) == "":
			missing = "Client ID"
		case strings.TrimSpace(p.ClientSecret) == "":
			missing = "Client Secret"
		case strings.TrimSpace(p.AuthURL) == "":
			missing = "Auth URL"
		case strings.TrimSpace(p.TokenURL) == "":
			missing = "Token URL"
		case strings.TrimSpace(p.UserInfoURL) == "":
			missing = "UserInfo URL"
		}
		if missing != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("启用 OAuth 服务商前请填写 %s", missing), "code": "OAUTH_INCOMPLETE"})
			return
		}
	}
	if err := SaveOAuthProvider(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "code": "SAVE_FAILED"})
		return
	}
	p.ClientSecret = ""
	c.JSON(http.StatusOK, gin.H{"provider": p})
}

func AdminDeleteOAuthProvider(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务商 ID 无效"})
		return
	}
	if err := DeleteOAuthProvider(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
