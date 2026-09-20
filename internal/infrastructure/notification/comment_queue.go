package notification

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
)

const (
	commentQueueCapacity = 256
	commentQueueWorkers  = 2
	commentRetryAttempts = 3

	commentReplyTemplate   = "template/comment_reply.html"
	commentArticleTemplate = "template/comment_article.html"
)

// Sender delivers one rendered notification. The queue owns retries and the
// bounded lifecycle; the sender only has to render and send.
type Sender func(context.Context, port.CommentNotification) error

// CommentQueue is a bounded asynchronous sink for comment notification emails.
// Comment writes never wait for SMTP and never fail because of it.
type CommentQueue struct {
	ctx    context.Context
	ch     chan port.CommentNotification
	send   Sender
	mu     sync.RWMutex
	closed bool
	once   sync.Once
	wg     sync.WaitGroup
}

var defaultCommentQueue atomic.Pointer[CommentQueue]

// StartCommentQueue installs a process-wide queue and returns it.
func StartCommentQueue(ctx context.Context, send Sender) *CommentQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	queue := newCommentQueue(ctx, send)
	queue.wg.Add(commentQueueWorkers)
	for i := 0; i < commentQueueWorkers; i++ {
		go queue.worker()
	}
	if previous := defaultCommentQueue.Swap(queue); previous != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = previous.Stop(shutdownCtx)
		cancel()
	}
	return queue
}

func newCommentQueue(ctx context.Context, send Sender) *CommentQueue {
	if ctx == nil {
		ctx = context.Background()
	}
	return &CommentQueue{
		ctx:  ctx,
		ch:   make(chan port.CommentNotification, commentQueueCapacity),
		send: send,
	}
}

// Stop closes the queue, drains pending notifications, and waits for ctx.
func (q *CommentQueue) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	q.once.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.ch)
		q.mu.Unlock()
	})
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		defaultCommentQueue.CompareAndSwap(q, nil)
		return nil
	case <-ctx.Done():
		return errors.Join(ctx.Err(), errors.New("comment notification queue shutdown timed out"))
	}
}

// StopCommentQueue stops the process-wide queue when one is installed.
func StopCommentQueue(ctx context.Context) error {
	queue := defaultCommentQueue.Load()
	if queue == nil {
		return nil
	}
	return queue.Stop(ctx)
}

func (q *CommentQueue) enqueue(item port.CommentNotification) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return false
	}
	select {
	case q.ch <- item:
		return true
	default:
		slog.Warn("comment notification dropped: queue is full", "capacity", commentQueueCapacity)
		return false
	}
}

func (q *CommentQueue) worker() {
	defer q.wg.Done()
	for {
		select {
		case item, ok := <-q.ch:
			if !ok {
				return
			}
			q.deliver(item)
		case <-q.ctx.Done():
			return
		}
	}
}

func (q *CommentQueue) deliver(item port.CommentNotification) {
	if q.send == nil {
		return
	}
	var err error
	for attempt := 1; attempt <= commentRetryAttempts; attempt++ {
		if err = q.send(q.ctx, item); err == nil {
			return
		}
		if attempt == commentRetryAttempts {
			slog.Error("comment notification failed after retries", "attempts", attempt, "commentId", item.CommentID, "error", err)
			return
		}
		timer := time.NewTimer(time.Duration(1<<(attempt-1)) * 200 * time.Millisecond)
		select {
		case <-timer.C:
		case <-q.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}
	}
}

// Notifier returns the domain port backed by the process-wide queue.
func Notifier() port.CommentNotifier { return queuedNotifier{} }

type queuedNotifier struct{}

func (queuedNotifier) EnqueueComment(item port.CommentNotification) error {
	queue := defaultCommentQueue.Load()
	if queue == nil {
		slog.Warn("comment notification dropped: queue is not started")
		return nil
	}
	queue.enqueue(item)
	return nil
}

// NewMailerSender renders the notification template and hands it to the mailer.
func NewMailerSender(mailer port.Mailer) Sender {
	return func(ctx context.Context, item port.CommentNotification) error {
		if mailer == nil {
			return errors.New("mailer is not configured")
		}
		templateName := commentArticleTemplate
		if item.ArticleID > 0 && item.ReplyAuthor != "" {
			templateName = commentReplyTemplate
		}
		return mailer.SendHTML(ctx, port.EmailMessage{
			To:       item.Recipient,
			Subject:  commentSubject(item),
			Template: templatePath(templateName),
			CommentMap: map[string]any{
				"nickname":       item.Nickname,
				"replyAuthor":    item.ReplyAuthor,
				"articleTitle":   item.ArticleTitle,
				"articleUrl":     item.ArticleURL,
				"commentContent": item.CommentBody,
			},
		})
	}
}

func commentSubject(item port.CommentNotification) string {
	if item.ReplyAuthor != "" {
		return item.ReplyAuthor + " 回复了你的评论"
	}
	return "你的内容收到了新评论"
}

func templatePath(name string) string {
	path, err := config.ResourcePath(name)
	if err != nil {
		return name
	}
	return filepath.ToSlash(path)
}
