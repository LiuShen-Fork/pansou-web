package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// OAuth 登录只下发会话 Cookie，前端启动时必须能仅凭 Cookie 识别登录态。
func TestAttachUserAcceptsSessionCookie(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user, err := CreateUser("cookie-user", "cookie@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := CreateSession(user.ID, "127.0.0.1", "test-agent", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AttachUser())
	r.GET("/who", func(c *gin.Context) {
		u := CurrentUser(c)
		if u == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"username": u.Username})
	})

	req := httptest.NewRequest(http.MethodGet, "/who", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Cookie 会话未被识别: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// 无效 Bearer 不应回退到 Cookie：前端需要在收到 401 后清除本地旧令牌再重试。
func TestAttachUserDoesNotFallBackToCookieForInvalidBearer(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user, err := CreateUser("cookie-user2", "cookie2@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := CreateSession(user.ID, "127.0.0.1", "test-agent", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AttachUser())
	r.GET("/who", func(c *gin.Context) {
		if CurrentUser(c) == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/who", nil)
	req.Header.Set("Authorization", "Bearer stale-token")
	req.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("无效 Bearer 不应回退到 Cookie: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
