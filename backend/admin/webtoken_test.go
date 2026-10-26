package admin

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWebTokenBindsIPAndUA(t *testing.T) {
	token, expires := issueWebToken("203.0.113.9", "Mozilla/5.0 test")
	if token == "" || expires <= time.Now().Unix() {
		t.Fatal("签发的网页凭证无效")
	}
	if !ValidateWebToken(token, "203.0.113.9", "Mozilla/5.0 test") {
		t.Fatal("相同 IP 与 UA 的凭证未通过校验")
	}
	if ValidateWebToken(token, "203.0.113.10", "Mozilla/5.0 test") {
		t.Fatal("凭证错误地接受了不同 IP")
	}
	if ValidateWebToken(token, "203.0.113.9", "Another UA") {
		t.Fatal("凭证错误地接受了不同 UA")
	}
	if ValidateWebToken(token+"x", "203.0.113.9", "Mozilla/5.0 test") {
		t.Fatal("被篡改的凭证未校验失败")
	}
	if ValidateWebToken("", "203.0.113.9", "Mozilla/5.0 test") {
		t.Fatal("空凭证未校验失败")
	}
}

func TestSameOriginRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest("GET", "http://example.com/api/web/token", nil)
	req.Host = "example.com"
	c.Request = req

	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if sameOriginRequest(c) {
		t.Fatal("跨站请求被误判为同源")
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if !sameOriginRequest(c) {
		t.Fatal("同源请求未通过校验")
	}

	req.Header.Del("Sec-Fetch-Site")
	req.Header.Set("Origin", "https://evil.example")
	if sameOriginRequest(c) {
		t.Fatal("不同 Origin 未校验失败")
	}
	req.Header.Set("Origin", "http://example.com")
	if !sameOriginRequest(c) {
		t.Fatal("相符 Origin 未通过校验")
	}
}

func TestSearchQuotaCounting(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user, err := CreateUser("quota", "quota@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	now := Now()
	logs := []struct {
		userID int64
		source string
		ip     string
	}{
		{user.ID, "api", "198.51.100.1"},
		{user.ID, "api", "198.51.100.1"},
		{user.ID, "web", "198.51.100.1"},
		{0, "web", "198.51.100.2"},
		{0, "api", "198.51.100.2"},
	}
	for _, l := range logs {
		if _, err := DB.Exec(
			`INSERT INTO search_logs (user_id, username, source, keyword, ip, created_at) VALUES (?, '', ?, 'k', ?, ?)`,
			l.userID, l.source, l.ip, now,
		); err != nil {
			t.Fatal(err)
		}
	}

	if n, err := UserAPISearchCountToday(user.ID); err != nil || n != 2 {
		t.Fatalf("用户 API 计数 = %d, err=%v, 期望 2", n, err)
	}
	if n, err := AnonymousIPSearchCountToday("198.51.100.2"); err != nil || n != 2 {
		t.Fatalf("匿名 IP 计数 = %d, err=%v, 期望 2", n, err)
	}
	if n, err := AnonymousGlobalSearchCountToday(); err != nil || n != 2 {
		t.Fatalf("匿名全站计数 = %d, err=%v, 期望 2", n, err)
	}
	if limit := EffectiveAPIDailyLimit(user, DefaultSettings()); limit != 0 {
		t.Fatalf("默认 API 额度 = %d, 期望 0（不限制）", limit)
	}
	apiLimit := 5
	if _, err := UpdateUser(user.ID, UserUpdate{APIDailySearchLimit: &apiLimit}); err != nil {
		t.Fatal(err)
	}
	updated, err := GetUserByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if limit := EffectiveAPIDailyLimit(updated, DefaultSettings()); limit != 5 {
		t.Fatalf("独立 API 额度 = %d, 期望 5", limit)
	}
}
