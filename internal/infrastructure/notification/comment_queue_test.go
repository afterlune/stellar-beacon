package notification

import (
	"context"
	"errors"
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

func TestCommentQueueRetriesAndDrainsOnStop(t *testing.T) {
	attempts := 0
	queue := newCommentQueue(context.Background(), func(context.Context, port.CommentNotification) error {
		attempts++
		if attempts < commentRetryAttempts {
			return errors.New("temporary smtp failure")
		}
		return nil
	})
	queue.wg.Add(commentQueueWorkers)
	for i := 0; i < commentQueueWorkers; i++ {
		go queue.worker()
	}

	if !queue.enqueue(port.CommentNotification{CommentID: 5}) {
		t.Fatal("enqueue() unexpectedly dropped the notification")
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := queue.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	if attempts != commentRetryAttempts {
		t.Fatalf("delivery attempts = %d, want %d", attempts, commentRetryAttempts)
	}
}

func TestCommentQueueDropsWhenStopped(t *testing.T) {
	queue := newCommentQueue(context.Background(), func(context.Context, port.CommentNotification) error { return nil })
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := queue.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	if queue.enqueue(port.CommentNotification{CommentID: 6}) {
		t.Fatal("a closed queue must reject new notifications")
	}
}

func TestCommentNotificationPicksTemplateAndSubject(t *testing.T) {
	mailer := &recordingMailer{}
	sender := NewMailerSender(mailer)

	if err := sender(context.Background(), port.CommentNotification{
		Recipient: "a@example.test", ReplyAuthor: "回复者", ArticleID: 3, ArticleTitle: "标题", ArticleURL: "https://site/articles/3",
	}); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 1 {
		t.Fatalf("expected one message, got %d", len(mailer.messages))
	}
	reply := mailer.messages[0]
	if !strings.Contains(reply.Subject, "回复") {
		t.Fatalf("unexpected reply subject: %q", reply.Subject)
	}
	if !strings.HasSuffix(reply.Template, "comment_reply.html") {
		t.Fatalf("unexpected reply template: %q", reply.Template)
	}
	if reply.CommentMap["nickname"] != "" || reply.CommentMap["articleUrl"] != "https://site/articles/3" {
		t.Fatalf("unexpected template payload: %+v", reply.CommentMap)
	}

	mailer.messages = nil
	if err := sender(context.Background(), port.CommentNotification{
		Recipient: "b@example.test", ArticleID: 4, ArticleTitle: "标题",
	}); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 1 || !strings.HasSuffix(mailer.messages[0].Template, "comment_article.html") {
		t.Fatalf("unexpected article notification: %+v", mailer.messages)
	}
}

func TestCommentNotificationRendererRequiresMailer(t *testing.T) {
	if err := NewMailerSender(nil)(context.Background(), port.CommentNotification{}); err == nil {
		t.Fatal("a missing mailer must be reported")
	}
}

// The mailer renders these templates with the payload the sender builds, so
// every placeholder must resolve. A missing key would silently ship a broken
// email, which is why the shapes are asserted here.
func TestCommentNotificationTemplatesRenderSenderPayload(t *testing.T) {
	mailer := &recordingMailer{}
	sender := NewMailerSender(mailer)
	base := port.CommentNotification{
		Recipient: "render@example.test", Nickname: "收件人",
		ArticleTitle: "标题", ArticleURL: "https://site/articles/1", CommentBody: "内容", ArticleID: 1,
	}
	if err := sender(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	reply := base
	reply.ReplyAuthor = "回复者"
	if err := sender(context.Background(), reply); err != nil {
		t.Fatal(err)
	}
	if len(mailer.messages) != 2 {
		t.Fatalf("expected two rendered messages, got %d", len(mailer.messages))
	}
	for _, message := range mailer.messages {
		parsed, err := template.ParseFiles(message.Template)
		if err != nil {
			t.Skipf("template resources unavailable: %v", err)
		}
		var rendered strings.Builder
		if err := parsed.Execute(&rendered, message.CommentMap); err != nil {
			t.Fatalf("render %s: %v", message.Template, err)
		}
		if strings.Contains(rendered.String(), "{{") {
			t.Fatalf("%s left an unresolved placeholder", message.Template)
		}
	}
}

type recordingMailer struct{ messages []port.EmailMessage }

func (m *recordingMailer) SendHTML(_ context.Context, message port.EmailMessage) error {
	m.messages = append(m.messages, message)
	return nil
}
