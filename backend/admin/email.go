package admin

import (
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// CreateEmailCode 生成 6 位验证码并写入数据库（只保存哈希）。
func CreateEmailCode(email, purpose string, ttl time.Duration) (string, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return "", err
	}
	if purpose == "" {
		purpose = "register"
	}
	code := randomCode()
	now := time.Now().UTC()
	_, err := DB.Exec(
		`INSERT INTO email_codes (email, code_hash, purpose, expires_at, used, created_at) VALUES (?, ?, ?, ?, 0, ?)`,
		email, sha256Hex(email+":"+code+":"+purpose), purpose,
		now.Add(ttl).Format(time.RFC3339), now.Format(time.RFC3339),
	)
	if err != nil {
		return "", err
	}
	return code, nil
}

func VerifyEmailCode(email, code, purpose string) error {
	email = NormalizeEmail(email)
	code = strings.TrimSpace(code)
	if purpose == "" {
		purpose = "register"
	}
	if len(code) != 6 {
		return fmt.Errorf("验证码格式不正确")
	}
	var id int64
	var expiresAt string
	var used int
	err := DB.QueryRow(
		`SELECT id, expires_at, used FROM email_codes WHERE email = ? AND code_hash = ? AND purpose = ?
		 ORDER BY id DESC LIMIT 1`,
		email, sha256Hex(email+":"+code+":"+purpose), purpose,
	).Scan(&id, &expiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("验证码错误")
	}
	if err != nil {
		return err
	}
	if used == 1 {
		return fmt.Errorf("验证码已使用")
	}
	if expired(expiresAt) {
		return fmt.Errorf("验证码已过期")
	}
	_, _ = DB.Exec(`UPDATE email_codes SET used = 1 WHERE id = ?`, id)
	return nil
}

// EmailCodeSendAllowed 简单频控：同一邮箱 10 分钟内最多 3 次。
// IP 维度由 AccessGate 的匿名限流兜底。
func EmailCodeSendAllowed(email string) (bool, string) {
	since := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	var byEmail int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM email_codes WHERE email = ? AND created_at >= ?`, NormalizeEmail(email), since).Scan(&byEmail)
	if byEmail >= 3 {
		return false, "验证码发送过于频繁，请稍后再试"
	}
	return true, ""
}

func parseAddress(s string) (string, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", err
	}
	return addr.Address, nil
}

func randomCode() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	n := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return fmt.Sprintf("%06d", n%1000000)
}

// SendMail 支持 465 隐式 TLS 与 587 STARTTLS。
func SendMail(cfg EmailSettings, to, subject, body string) error {
	if cfg.Host == "" || cfg.Port <= 0 {
		return fmt.Errorf("SMTP 未配置")
	}
	from := strings.TrimSpace(cfg.From)
	if from == "" {
		from = strings.TrimSpace(cfg.Username)
	}
	if from == "" {
		return fmt.Errorf("发件人地址未配置")
	}
	if _, err := parseAddress(from); err != nil {
		return fmt.Errorf("发件人地址不正确: %w", err)
	}
	if _, err := parseAddress(to); err != nil {
		return fmt.Errorf("收件人地址不正确: %w", err)
	}

	fromHeader := from
	if cfg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("UTF-8", cfg.FromName), from)
	}
	msg := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}

	// 465 端口使用隐式 TLS；其余端口交给 net/smtp 自动 STARTTLS。
	if cfg.Port == 465 {
		return sendImplicitTLS(cfg.Host, addr, auth, from, to, []byte(msg))
	}
	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("发送失败: %w", err)
	}
	return nil
}

func sendImplicitTLS(host, addr string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("连接 SMTP 失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP 认证失败: %w", err)
			}
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}

func BuildCodeMail(siteName, code, purpose string) (string, string) {
	if siteName == "" {
		siteName = "PanSou"
	}
	action := "注册"
	if purpose == "reset" {
		action = "重置密码"
	}
	subject := fmt.Sprintf("%s - %s验证码", siteName, action)
	body := fmt.Sprintf("你正在%s %s。\n\n验证码：%s\n\n验证码 10 分钟内有效。如果这不是你的操作，请忽略本邮件。",
		action, siteName, code)
	return subject, body
}
