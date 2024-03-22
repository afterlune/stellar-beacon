package shared

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/config"
	"benetnasch/app/infrastructure/zlog"
	"github.com/wneessen/go-mail"
	"html/template"
	"io"
)

func SendHtmlEmail(dto model.EmailDTO) {
	m := mail.NewMsg()
	err := m.From(config.SmtpEmail)
	if err != nil {
		zlog.Error(err.Error())
	}
	err = m.To(dto.Email)
	if err != nil {
		zlog.Error(err.Error())
	}
	m.Subject(dto.Subject)
	t := template.New("common.html")
	parse, err := t.ParseFiles(dto.Template)
	if err != nil {
		zlog.Error(err.Error())
	}
	err = m.SetBodyHTMLTemplate(parse, dto.CommentMap)
	if err != nil {
		zlog.Error(err.Error())
	}
	client, err := mail.NewClient(config.SmtpName, mail.WithPort(config.SmtpPort), mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(config.SmtpEmail), mail.WithPassword(config.SmtpPassword), mail.WithSSL())
	mail.WithDebugLog()
	if err != nil {
		zlog.Error(err.Error())
	}
	err = client.DialAndSend(m)
	if err != nil && err != io.EOF {
		zlog.Error(err.Error())
	}
}
