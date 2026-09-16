package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"html/template"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wneessen/go-mail"
)

type SMTPMailer struct {
	config config.Email
}

func NewSMTPMailer(conf *config.Email) *SMTPMailer {
	return &SMTPMailer{config: *conf}
}

func (m *SMTPMailer) SendHTML(ctx context.Context, message port.EmailMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg := mail.NewMsg()
	if err := msg.From(m.config.EmailAccount); err != nil {
		return fmt.Errorf("set email sender: %w", err)
	}
	if err := msg.To(message.To); err != nil {
		return fmt.Errorf("set email recipient: %w", err)
	}
	msg.Subject(message.Subject)
	templateName := filepath.Base(message.Template)
	tmpl, err := template.New(templateName).ParseFiles(message.Template)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}
	if err := msg.SetBodyHTMLTemplate(tmpl, message.CommentMap); err != nil {
		return fmt.Errorf("render email template: %w", err)
	}
	options := []mail.Option{mail.WithPort(m.config.SmtpPort)}
	if m.config.Auth {
		options = append(options,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(m.config.EmailAccount),
			mail.WithPassword(m.config.Password),
		)
	}
	if m.config.TLS {
		options = append(options, mail.WithSSL())
	} else {
		// go-mail defaults to mandatory STARTTLS; Mailpit's integration
		// listener is intentionally plaintext, so opt out explicitly.
		options = append(options, mail.WithTLSPolicy(mail.NoTLS))
	}
	client, err := mail.NewClient(m.config.SmtpName, options...)
	if err != nil {
		return fmt.Errorf("create email client: %w", err)
	}
	if err := client.DialAndSendWithContext(ctx, msg); err != nil && err != io.EOF {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// Check verifies the configured SMTP endpoint without sending a message.
// The result is safe to expose in the admin console: credentials are never
// returned and connectivity is checked with a short timeout.
func (m *SMTPMailer) Check(ctx context.Context) port.MailerHealth {
	checkedAt := time.Now()
	status := port.MailerHealth{
		Host:      m.config.SmtpName,
		Port:      m.config.SmtpPort,
		TLS:       m.config.TLS,
		Auth:      m.config.Auth,
		CheckedAt: checkedAt,
	}
	host := strings.TrimSpace(m.config.SmtpName)
	status.Configured = host != "" && m.config.SmtpPort > 0 && strings.TrimSpace(m.config.EmailAccount) != ""
	if m.config.Auth && strings.TrimSpace(m.config.Password) == "" {
		status.Configured = false
	}
	if !status.Configured {
		status.Message = "SMTP 配置不完整"
		return status
	}

	checkCtx := ctx
	if checkCtx == nil {
		checkCtx = context.Background()
	}
	checkCtx, cancel := context.WithTimeout(checkCtx, 5*time.Second)
	defer cancel()
	address := net.JoinHostPort(host, strconv.Itoa(m.config.SmtpPort))
	var (
		connection net.Conn
		err        error
	)
	if m.config.TLS {
		connection, err = (&tls.Dialer{
			NetDialer: &net.Dialer{Timeout: 5 * time.Second},
			Config:    &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12},
		}).DialContext(checkCtx, "tcp", address)
	} else {
		connection, err = (&net.Dialer{Timeout: 5 * time.Second}).DialContext(checkCtx, "tcp", address)
	}
	if err != nil {
		status.Message = "SMTP 服务不可达"
		return status
	}
	_ = connection.Close()
	status.Reachable = true
	status.Message = "SMTP 服务正常"
	return status
}

var _ port.Mailer = (*SMTPMailer)(nil)
