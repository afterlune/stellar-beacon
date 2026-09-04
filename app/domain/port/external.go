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

// ObjectRef is the public result of an object-storage upload.
type ObjectRef struct {
	Key string
	URL string
}

// ObjectStorage hides the provider-specific object SDK from application code.
type ObjectStorage interface {
	Put(context.Context, string, io.Reader) (ObjectRef, error)
}

// ArticleSearchSource is provenance attached to a semantic or hybrid hit.
// ArticleSearch already contains the legacy article ID/title fields; this
// nested value keeps the original article and chunk context explicit.
type ArticleSearchSource struct {
	ArticleID       int      `json:"articleId"`
	Title           string   `json:"title"`
	URL             string   `json:"url,omitempty"`
	ChunkID         string   `json:"chunkId,omitempty"`
	ChunkIndex      int      `json:"chunkIndex"`
	Category        string   `json:"category,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	PublishedAtUnix int64    `json:"publishedAtUnix,omitempty"`
}

// ArticleSearchRelevance is deliberately provider-neutral. Score is the
// ranking score returned by the configured search index, while Mode and Index
// explain which retrieval path produced it.
type ArticleSearchRelevance struct {
	Score float64    `json:"score"`
	Mode  SearchMode `json:"mode"`
	Index string     `json:"index,omitempty"`
}

// ArticleSearchHit is a typed search result. Highlighted fields preserve the
// existing MeiliSearch response behavior without exposing raw SDK maps;
// Source and Relevance are additive fields for the enriched search response.
type ArticleSearchHit struct {
	ArticleSearch
	HighlightedTitle   string
	HighlightedContent string
	Source             *ArticleSearchSource
	Relevance          *ArticleSearchRelevance
}

// ArticleSearchResult is the HTTP-facing article search shape. The legacy
// article fields remain embedded at the top level; the additional fields are
// additive so existing clients can continue reading highlighted title/content
// from articleTitle/articleContent.
type ArticleSearchResult struct {
	ArticleSearch
	HighlightedTitle   string                  `json:"highlightedTitle,omitempty"`
	HighlightedContent string                  `json:"highlightedContent,omitempty"`
	Source             *ArticleSearchSource    `json:"source,omitempty"`
	Relevance          *ArticleSearchRelevance `json:"relevance,omitempty"`
}

// ArticleSearcher provides article search to the application layer.
type ArticleSearcher interface {
	Search(context.Context, string) ([]ArticleSearchHit, error)
}

// ArticleModeSearcher is an optional extension of the legacy article search
// contract. Implementations must keep Search compatible with the existing
// keyword endpoint and only use the additional modes when explicitly asked.
type ArticleModeSearcher interface {
	ArticleSearcher
	SearchWithMode(context.Context, string, SearchMode) ([]ArticleSearchHit, error)
}

// ArticleFilteredModeSearcher is an optional extension used when callers
// need structured category, tag, year, or time-range filters. Implementations
// must preserve the legacy behavior when the filter is empty.
type ArticleFilteredModeSearcher interface {
	ArticleModeSearcher
	SearchWithModeAndFilter(context.Context, string, SearchMode, KnowledgeFilter) ([]ArticleSearchHit, error)
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
