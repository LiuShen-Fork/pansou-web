package admin

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// TestListUsersDoesNotDeadlock 固定住一个真实事故：
// ListUsers 在遍历结果集时调用 UserSearchCountToday，而连接池只有 1 条连接，
// 第二条查询永远等不到连接，最终把整个进程（包括登录接口）卡死。
// 这里刻意把池设为 1，确保实现先关闭结果集再查统计。
func TestListUsersDoesNotDeadlock(t *testing.T) {
	oldDB := DB
	defer func() { DB = oldDB }()

	path := filepath.Join(t.TempDir(), "admin.db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	DB = db
	if err := migrate(); err != nil {
		t.Fatal(err)
	}

	now := Now()
	for _, name := range []string{"alice", "bob"} {
		if _, err := DB.Exec(
			`INSERT INTO users (username, email, password_hash, role, status, must_change_password, daily_search_limit, created_at, updated_at)
			 VALUES (?, '', '', 'user', 'active', 0, 0, ?, ?)`, name, now, now,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := DB.Exec(
		`INSERT INTO search_logs (user_id, username, source, keyword, ip, created_at)
		 VALUES (1, 'alice', 'api', 'test', '127.0.0.1', ?)`, now,
	); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		list, total, err := ListUsers("", "", "", 1, 20)
		if err == nil && (total != 2 || len(list) != 2) {
			t.Errorf("ListUsers 返回 %d 个用户（total=%d），期望 2", len(list), total)
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ListUsers 失败: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ListUsers 超时：疑似再次出现单连接死锁")
	}
}
