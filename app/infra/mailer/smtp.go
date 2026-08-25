package mailer

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"fmt"
	"html/template"
	"io"

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
	tmpl, err := template.New("common.html").ParseFiles(message.Template)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}
	if err := msg.SetBodyHTMLTemplate(tmpl, message.CommentMap); err != nil {
		return fmt.Errorf("render email template: %w", err)
	}
	client, err := mail.NewClient(
		m.config.SmtpName,
		mail.WithPort(m.config.SmtpPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(m.config.EmailAccount),
		mail.WithPassword(m.config.Password),
		mail.WithSSL(),
	)
	if err != nil {
		return fmt.Errorf("create email client: %w", err)
	}
	if err := client.DialAndSendWithContext(ctx, msg); err != nil && err != io.EOF {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

var _ port.Mailer = (*SMTPMailer)(nil)
