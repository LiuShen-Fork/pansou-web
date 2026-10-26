package admin

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()
	oldDB := DB
	path := filepath.Join(t.TempDir(), "admin.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	DB = db
	if err := migrate(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return func() {
		db.Close()
		DB = oldDB
	}
}

func TestAPITokenLifecycleAndLimit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	user, err := CreateUser("alice", "alice@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	other, err := CreateUser("bob", "bob@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}

	var rawTokens []string
	for i := 0; i < MaxAPITokensPerUser; i++ {
		item, raw, err := CreateAPIToken(user.ID, "")
		if err != nil {
			t.Fatalf("创建第 %d 个令牌失败: %v", i+1, err)
		}
		if item.Name != defaultTokenName {
			t.Fatalf("默认令牌名称错误: %q", item.Name)
		}
		if u, err := GetUserByAPIToken(raw); err != nil || u.ID != user.ID {
			t.Fatalf("令牌未解析到正确用户: u=%v err=%v", u, err)
		}
		rawTokens = append(rawTokens, raw)
	}

	if _, _, err := CreateAPIToken(user.ID, "超出上限"); err == nil {
		t.Fatal("超过上限后仍能创建令牌")
	}
	list, err := ListAPITokens(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != MaxAPITokensPerUser {
		t.Fatalf("令牌数量为 %d，期望 %d", len(list), MaxAPITokensPerUser)
	}
	if list[0].LastUsedAt == "" {
		t.Fatal("使用令牌后 last_used_at 未更新")
	}

	// 删除一个后可重新创建
	if err := DeleteAPIToken(user.ID, list[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateAPIToken(user.ID, "补充令牌"); err != nil {
		t.Fatalf("删除后重新创建失败: %v", err)
	}

	// 越权删除：用户不能删除别人的令牌
	otherTokens, err := ListAPITokens(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := DeleteAPIToken(other.ID, list[1].ID); err == nil {
		t.Fatal("用户删除了不属于自己的令牌")
	}
	if len(otherTokens) != 0 {
		t.Fatalf("新用户不应有令牌，实际 %d 个", len(otherTokens))
	}

	// 令牌只存哈希：数据库里不应出现明文
	for _, raw := range rawTokens {
		var count int
		if err := DB.QueryRow(`SELECT COUNT(*) FROM api_tokens WHERE token_hash = ?`, raw).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("数据库中出现了令牌明文")
		}
	}
}

func TestAdminIsGloballyUnique(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	admin, err := CreateUser("admin", "admin@example.com", "password123", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateUser("admin2", "admin2@example.com", "password123", RoleAdmin); err == nil {
		t.Fatal("创建了第二个管理员")
	}
	user, err := CreateUser("alice", "alice@example.com", "password123", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	role := RoleAdmin
	if _, err := UpdateUser(user.ID, UserUpdate{Role: &role}); err == nil {
		t.Fatal("普通用户被提升为管理员")
	}
	role = RoleUser
	if _, err := UpdateUser(admin.ID, UserUpdate{Role: &role}); err == nil {
		t.Fatal("唯一的管理员被降级")
	}
	if err := DeleteUser(admin.ID, 0); err == nil {
		t.Fatal("唯一的管理员被删除")
	}
}
