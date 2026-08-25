package shared

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/config"
	"fmt"
	"github.com/wneessen/go-mail"
	"html/template"
	"io"
)

func SendHtmlEmail(dto model.EmailDTO) error {
	cfg := new(config.Email).Email()

	m := mail.NewMsg()
	err := m.From(cfg.EmailAccount)
	if err != nil {
		return fmt.Errorf("set email sender: %w", err)
	}
	err = m.To(dto.Email)
	if err != nil {
		return fmt.Errorf("set email recipient: %w", err)
	}
	m.Subject(dto.Subject)
	t := template.New("common.html")
	parse, err := t.ParseFiles(dto.Template)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}
	err = m.SetBodyHTMLTemplate(parse, dto.CommentMap)
	if err != nil {
		return fmt.Errorf("render email template: %w", err)
	}
	client, err := mail.NewClient(cfg.SmtpName, mail.WithPort(cfg.SmtpPort), mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(cfg.EmailAccount), mail.WithPassword(cfg.Password), mail.WithSSL())
	if err != nil {
		return fmt.Errorf("create email client: %w", err)
	}
	err = client.DialAndSend(m)
	if err != nil && err != io.EOF {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
