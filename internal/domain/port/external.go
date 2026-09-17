package port

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// ErrCacheMiss is returned when a cache key or hash field does not exist.
// Callers can decide whether the cache is optional for their operation.
var ErrCacheMiss = errors.New("cache miss")

// Cache is the application-facing cache contract. It deliberately exposes
// only the Redis primitives currently used by the application and keeps the
// concrete client out of application services.
type Cache interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, any, time.Duration) error
	SetNX(context.Context, string, any, time.Duration) (bool, error)
	IncrementWithExpiry(context.Context, string, time.Duration) (int64, error)
	Expire(context.Context, string, time.Duration) (bool, error)
	Delete(context.Context, string) error

	HGet(context.Context, string, string) (string, error)
	HGetAll(context.Context, string) (map[string]string, error)
	HSet(context.Context, string, string, any, time.Duration) error
	HDel(context.Context, string, string) error
	HIncrBy(context.Context, string, string, int64) (int64, error)

	SIsMember(context.Context, string, any) (bool, error)
	SAdd(context.Context, string, ...any) (int64, error)

	IncrBy(context.Context, string, int64) (int64, error)
	ZIncrBy(context.Context, string, float64, string) (float64, error)
	ZScore(context.Context, string, string) (float64, error)
	ZRevRangeWithScores(context.Context, string, int64, int64) (map[string]float64, error)
	ZRangeWithScores(context.Context, string) (map[string]float64, error)
}

// RateLimiter provides short-lived counters for public actions. Keys are
// expected to be opaque hashes; implementations must not persist them beyond
// the configured window.
type RateLimiter interface {
	Allow(context.Context, string, int64, time.Duration) (bool, error)
}

// ObjectRef is the public result of an object-storage upload.
type ObjectRef struct {
	Key string
	URL string
}

// ObjectStorage hides the provider-specific object SDK from application code.
type ObjectStorage interface {
	Put(context.Context, string, io.Reader) (ObjectRef, error)
}

// ObjectInfo describes an object that can be browsed from the administration
// media center. The core upload contract intentionally stays small; providers
// opt into MediaStorage when they support listing and deletion.
type ObjectInfo struct {
	Key          string
	URL          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

// MediaStorage is an optional extension of ObjectStorage used by the admin
// media library. Keeping it separate means existing upload-only fakes and
// providers remain valid for the rest of the application.
type MediaStorage interface {
	ObjectStorage
	List(context.Context, string) ([]ObjectInfo, error)
	Delete(context.Context, []string) error
}

// ArticleSearchHit is a typed search result. Highlighted fields preserve the
// existing MeiliSearch response behavior without exposing raw SDK maps.
type ArticleSearchHit struct {
	ArticleSearch
	HighlightedTitle   string
	HighlightedContent string
}

// ArticleSearcher provides article search to the application layer.
type ArticleSearcher interface {
	Search(context.Context, string) ([]ArticleSearchHit, error)
}

// EmailMessage is the provider-neutral email command used by services.
type EmailMessage struct {
	To         string
	Subject    string
	Template   string
	CommentMap map[string]any
}

// Mailer sends HTML email using an infrastructure-specific provider.
type Mailer interface {
	SendHTML(context.Context, EmailMessage) error
}

// MailerHealthChecker is an optional operational extension. It deliberately
// stays separate from Mailer so existing test doubles and other providers do
// not need to implement an SMTP-specific diagnostic operation.
type MailerHealthChecker interface {
	Check(context.Context) MailerHealth
}

type MailerHealth struct {
	Configured bool
	Reachable  bool
	Host       string
	Port       int
	TLS        bool
	Auth       bool
	CheckedAt  time.Time
	Message    string
}

// NewsletterEnqueuer is the small collaboration needed by article
// publication. Keeping it separate from the mailer prevents article saves
// from depending on SMTP availability.
type NewsletterEnqueuer interface {
	EnqueueArticle(context.Context, int) error
}

// CommentNotification is one queued comment-notification email. It carries the
// rendered inputs rather than entities so the delivery layer stays independent
// of the persistence model.
type CommentNotification struct {
	CommentID    int
	ArticleID    int
	RecipientID  int
	Recipient    string
	Nickname     string
	ReplyAuthor  string
	ArticleTitle string
	ArticleURL   string
	CommentBody  string
}

// CommentNotifier queues comment notifications. Comment writes must not wait
// for, or fail because of, the mail transport.
type CommentNotifier interface {
	EnqueueComment(CommentNotification) error
}

// VisitorIdentity contains request-derived information used by rate limiting
// and unique-visitor accounting.
type VisitorIdentity struct {
	IP             string
	Region         string
	Browser        string
	BrowserVersion string
	OS             string
	Fingerprint    string
	IsBot          bool
}

// VisitorResolver keeps IP/UA parsing and the IP-region database in infra.
type VisitorResolver interface {
	Resolve(context.Context, *http.Request) (VisitorIdentity, error)
}
