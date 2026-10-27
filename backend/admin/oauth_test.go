package admin

import "testing"

func TestUpsertPrimaryOAuthProviderKeepsSingleRow(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	first, err := UpsertPrimaryOAuthProvider(&OAuthProvider{
		Name:         "GitHub",
		ClientID:     "client-1",
		ClientSecret: "secret-1",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Slug == "" {
		t.Fatal("新建服务商未生成 slug")
	}

	// 第二次保存即使传入不同 slug / ID，也只能更新同一条记录。
	second, err := UpsertPrimaryOAuthProvider(&OAuthProvider{
		ID:          99,
		Slug:        "google",
		Name:        "Google",
		Enabled:     true,
		ClientID:    "client-2",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("服务商被新建为第二条记录: first=%d second=%d", first.ID, second.ID)
	}
	if second.Name != "Google" {
		t.Fatalf("名称未更新: %q", second.Name)
	}
	if second.Slug != first.Slug {
		t.Fatalf("slug 被改变: %q -> %q", first.Slug, second.Slug)
	}
	if second.ClientSecret == "" {
		t.Fatal("更新时未保留已有 Client Secret")
	}

	list, err := ListOAuthProviders(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("OAuth 服务商数量为 %d，期望 1", len(list))
	}
}
