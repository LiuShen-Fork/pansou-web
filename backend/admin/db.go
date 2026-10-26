package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var (
	DB *sql.DB

	initOnce sync.Once
	initErr  error
)

// Enabled 表示后台账户库已经初始化完成。
func Enabled() bool { return DB != nil }

// Init 打开（并迁移）管理后台数据库，然后确保默认管理员存在。
// 默认路径为 data/pansou-admin.db，可通过 ADMIN_DB_PATH 覆盖；
// 容器内由镜像设置为 /app/data/pansou-admin.db。
func Init() error {
	initOnce.Do(func() {
		path := strings.TrimSpace(os.Getenv("ADMIN_DB_PATH"))
		if path == "" {
			dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
			if dataDir == "" {
				dataDir = "data"
			}
			path = filepath.Join(dataDir, "pansou-admin.db")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			initErr = fmt.Errorf("创建后台数据库目录失败: %w", err)
			return
		}

		dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", filepath.ToSlash(path))
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			initErr = fmt.Errorf("打开后台数据库失败: %w", err)
			return
		}
		// SQLite 在 WAL 模式下支持多读单写。这里不能只留 1 条连接：
		// 管理后台存在“遍历结果集时再查一次”的场景，单连接会自我死锁。
		db.SetMaxOpenConns(8)
		db.SetMaxIdleConns(8)
		db.SetConnMaxLifetime(0)
		if err := db.Ping(); err != nil {
			_ = db.Close()
			initErr = fmt.Errorf("连接后台数据库失败: %w", err)
			return
		}

		DB = db
		if err := migrate(); err != nil {
			initErr = err
			return
		}
		if err := ensureDefaultAdmin(); err != nil {
			initErr = err
			return
		}
		if err := ensureDefaultSettings(); err != nil {
			initErr = err
			return
		}
		log.Printf("[admin] 账户数据库已就绪: %s", path)
	})
	return initErr
}

func migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL COLLATE NOCASE UNIQUE,
			email TEXT NOT NULL DEFAULT '' COLLATE NOCASE,
			password_hash TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'user',
			status TEXT NOT NULL DEFAULT 'active',
			must_change_password INTEGER NOT NULL DEFAULT 0,
			daily_search_limit INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_login_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email <> ''`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiry ON sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			prefix TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL COLLATE NOCASE UNIQUE,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			client_id TEXT NOT NULL DEFAULT '',
			client_secret TEXT NOT NULL DEFAULT '',
			auth_url TEXT NOT NULL DEFAULT '',
			token_url TEXT NOT NULL DEFAULT '',
			userinfo_url TEXT NOT NULL DEFAULT '',
			scopes TEXT NOT NULL DEFAULT 'openid profile email',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id INTEGER NOT NULL REFERENCES oauth_providers(id) ON DELETE CASCADE,
			subject TEXT NOT NULL,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			email TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(provider_id, subject)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user ON oauth_accounts(user_id)`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			state TEXT PRIMARY KEY,
			provider_id INTEGER NOT NULL,
			mode TEXT NOT NULL,
			user_id INTEGER NOT NULL DEFAULT 0,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS email_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			code_hash TEXT NOT NULL,
			purpose TEXT NOT NULL DEFAULT 'register',
			expires_at TEXT NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_codes_lookup ON email_codes(email, purpose, created_at)`,
		`CREATE TABLE IF NOT EXISTS login_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			account TEXT NOT NULL,
			success INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_ip ON login_attempts(ip, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_login_attempts_account ON login_attempts(account, created_at)`,
		`CREATE TABLE IF NOT EXISTS search_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL DEFAULT 0,
			username TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			keyword TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_search_logs_user_time ON search_logs(user_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_search_logs_time ON search_logs(created_at)`,
	}
	for _, stmt := range statements {
		if _, err := DB.Exec(stmt); err != nil {
			return fmt.Errorf("迁移后台数据库失败: %w", err)
		}
	}
	return nil
}

func ensureDefaultSettings() error {
	if err := GetSetting("auth", &Settings{}); err == nil {
		return nil
	}
	if err := SaveSettings(DefaultSettings()); err != nil {
		return err
	}
	if err := SaveEmailSettings(DefaultEmailSettings()); err != nil {
		return err
	}
	return nil
}

func ensureDefaultAdmin() error {
	// 救援开关：忘记密码时设置 PANSOU_RESET_ADMIN=1 启动一次，
	// 会把第一个管理员重置为默认密码并强制改密，然后正常启动即可。
	if os.Getenv("PANSOU_RESET_ADMIN") == "1" {
		var id int64
		if err := DB.QueryRow(`SELECT id FROM users WHERE role = ? ORDER BY id LIMIT 1`, RoleAdmin).Scan(&id); err == nil {
			hash, hashErr := HashPassword(DefaultAdminPassword)
			if hashErr != nil {
				return hashErr
			}
			now := Now()
			if _, err := DB.Exec(
				`UPDATE users SET password_hash = ?, must_change_password = 1, status = ?, updated_at = ? WHERE id = ?`,
				hash, StatusActive, now, id,
			); err != nil {
				return err
			}
			DeleteUserSessions(id)
			log.Printf("[admin] PANSOU_RESET_ADMIN=1：已重置管理员 ID=%d 的密码为默认密码，请登录后立即修改", id)
		}
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = ?`, RoleAdmin).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := HashPassword(DefaultAdminPassword)
	if err != nil {
		return err
	}
	now := Now()
	res, err := DB.Exec(
		`INSERT INTO users (username, email, password_hash, role, status, must_change_password, daily_search_limit, created_at, updated_at)
		 VALUES (?, '', ?, ?, ?, 1, 0, ?, ?)`,
		DefaultAdminUsername, hash, RoleAdmin, StatusActive, now, now,
	)
	if err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}
	_, _ = res.LastInsertId()
	log.Printf("[admin] 已创建默认管理员 %s，首次登录后请立即修改密码", DefaultAdminUsername)
	return nil
}

func Now() string { return time.Now().UTC().Format(time.RFC3339) }

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func isNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
