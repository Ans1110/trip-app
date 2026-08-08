package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"strings"

	"github.com/Ans1110/trip-app/pkg/config"
	"go.uber.org/zap"
)

type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, name, token string) error
	SendPasswordResetEmail(ctx context.Context, to, name, token string) error
	SendMFACodeEmail(ctx context.Context, to, name, code string) error
}

func New(cfg config.NotificationConfig, logger *zap.Logger) Mailer {
	webURL := strings.TrimRight(cfg.WebURL, "/")
	if webURL == "" {
		webURL = "http://localhost:3000"
	}

	log := logger.With(zap.String("module", "mailer"))

	if !smtpConfigured(cfg.SMTP) {
		log.Info("mailer initialized: log mode (SMTP not configured)",
			zap.String("web_url", webURL),
		)
		return &logMailer{logger: log, webURL: webURL}
	}

	log.Info("mailer initialized: SMTP mode",
		zap.String("web_url", webURL),
		zap.String("smtp_host", cfg.SMTP.Host),
		zap.Int("smtp_port", cfg.SMTP.Port),
		zap.String("from", cfg.SMTP.From),
	)
	return &smtpMailer{cfg: cfg.SMTP, webURL: webURL, logger: log}
}

func smtpConfigured(s config.SMTPConfig) bool {
	if s.Host == "" || s.Host == "smtp.example.com" {
		return false
	}
	if s.User == "" || s.Password == "" {
		return false
	}
	return true
}

type logMailer struct {
	logger *zap.Logger
	webURL string
}

func (m *logMailer) SendVerificationEmail(_ context.Context, to, name, token string) error {
	link := buildLink(m.webURL, "/verify-email", token)
	m.logger.Info("verification email (dev log)",
		zap.String("to", to),
		zap.String("name", name),
		zap.String("link", link),
	)
	return nil
}

func (m *logMailer) SendPasswordResetEmail(_ context.Context, to, name, token string) error {
	link := buildLink(m.webURL, "/reset-password", token)
	m.logger.Info("password reset email (dev log)",
		zap.String("to", to),
		zap.String("name", name),
		zap.String("link", link),
	)
	return nil
}

func (m *logMailer) SendMFACodeEmail(_ context.Context, to, name, code string) error {
	m.logger.Info("mfa code email (dev log)",
		zap.String("to", to),
		zap.String("name", name),
		zap.String("code", code),
	)
	return nil
}

type smtpMailer struct {
	cfg    config.SMTPConfig
	webURL string
	logger *zap.Logger
}

func (m *smtpMailer) SendVerificationEmail(_ context.Context, to, name, token string) error {
	link := buildLink(m.webURL, "/verify-email", token)
	return m.send(to, "Verify your TripCraft email", renderVerification(name, link))
}

func (m *smtpMailer) SendPasswordResetEmail(_ context.Context, to, name, token string) error {
	link := buildLink(m.webURL, "/reset-password", token)
	return m.send(to, "Reset your TripCraft password", renderPasswordReset(name, link))
}

func (m *smtpMailer) SendMFACodeEmail(_ context.Context, to, name, code string) error {
	return m.send(to, "Your TripCraft sign-in code", renderMFACode(name, code))
}

func (m *smtpMailer) send(to, subject, htmlBody string) error {
	from := m.cfg.From
	if from == "" {
		from = m.cfg.User
	}
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)

	headers := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}, "\r\n")
	msg := []byte(headers + "\r\n\r\n" + htmlBody)

	if err := smtp.SendMail(addr, auth, from, []string{to}, msg); err != nil {
		m.logger.Warn("smtp send failed",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func buildLink(base, path, token string) string {
	return fmt.Sprintf("%s%s?token=%s", base, path, url.QueryEscape(token))
}

func renderVerification(name, link string) string {
	greeting := greet(name)
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#0B100D;color:#ECEFEA;font-family:Helvetica,Arial,sans-serif;">
  <div style="max-width:560px;margin:0 auto;padding:40px 24px;">
    <h1 style="font-family:Georgia,serif;font-weight:400;font-size:28px;margin:0 0 24px;">Welcome to TripCraft</h1>
    <p style="margin:0 0 8px;">%s,</p>
    <p style="margin:0 0 24px;color:#8B9A8E;">Confirm your email to start planning trips.</p>
    <p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#7FB68A;color:#0B100D;text-decoration:none;padding:12px 22px;border-radius:999px;font-weight:600;">Verify email</a></p>
    <p style="margin:0 0 8px;color:#8B9A8E;font-size:13px;">Or paste this link:</p>
    <p style="margin:0 0 24px;color:#A8E0B4;font-size:13px;word-break:break-all;">%s</p>
    <p style="margin:24px 0 0;color:#6B7A6F;font-size:12px;">If you didn't sign up, you can ignore this email.</p>
  </div>
</body></html>`, greeting, link, link)
}

func renderPasswordReset(name, link string) string {
	greeting := greet(name)
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#0B100D;color:#ECEFEA;font-family:Helvetica,Arial,sans-serif;">
  <div style="max-width:560px;margin:0 auto;padding:40px 24px;">
    <h1 style="font-family:Georgia,serif;font-weight:400;font-size:28px;margin:0 0 24px;">Reset your password</h1>
    <p style="margin:0 0 8px;">%s,</p>
    <p style="margin:0 0 24px;color:#8B9A8E;">Tap the button to choose a new password. The link expires in 30 minutes.</p>
    <p style="margin:0 0 24px;"><a href="%s" style="display:inline-block;background:#7FB68A;color:#0B100D;text-decoration:none;padding:12px 22px;border-radius:999px;font-weight:600;">Reset password</a></p>
    <p style="margin:0 0 8px;color:#8B9A8E;font-size:13px;">Or paste this link:</p>
    <p style="margin:0 0 24px;color:#A8E0B4;font-size:13px;word-break:break-all;">%s</p>
    <p style="margin:24px 0 0;color:#6B7A6F;font-size:12px;">If you didn't request this, you can safely ignore it.</p>
  </div>
</body></html>`, greeting, link, link)
}

func renderMFACode(name, code string) string {
	greeting := greet(name)
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#0B100D;color:#ECEFEA;font-family:Helvetica,Arial,sans-serif;">
  <div style="max-width:560px;margin:0 auto;padding:40px 24px;">
    <h1 style="font-family:Georgia,serif;font-weight:400;font-size:28px;margin:0 0 24px;">Your sign-in code</h1>
    <p style="margin:0 0 8px;">%s,</p>
    <p style="margin:0 0 24px;color:#8B9A8E;">Use this code to finish signing in. It expires in a few minutes.</p>
    <p style="margin:0 0 24px;font-size:32px;letter-spacing:12px;font-weight:600;color:#A8E0B4;">%s</p>
    <p style="margin:24px 0 0;color:#6B7A6F;font-size:12px;">If you didn't try to sign in, you can ignore this email.</p>
  </div>
</body></html>`, greeting, code)
}

func greet(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "Hi"
	}
	return "Hi " + n
}
