package port

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// ArticleRepository is the application-facing contract for article reads.
// It owns domain read models and propagates persistence failures instead of
// turning them into empty result sets.
type ArticleRepository interface {
	ListTopAndFeaturedArticles(ctx context.Context) ([]*ArticleCard, error)
	ListArticles(ctx context.Context, current, size int) ([]*ArticleCard, int, error)
	GetArticlesByCategoryID(ctx context.Context, current, size, categoryID int) ([]*ArticleCard, int, error)
	GetArticlesByCategoryName(ctx context.Context, current, size int, name string) ([]*ArticleCard, int, error)
	ListArticleCardsByIDs(ctx context.Context, articleIDs []int) ([]*ArticleCard, error)
	ListArticleCardsBySeries(ctx context.Context, seriesID int) ([]*ArticleCard, error)
	ListRelatedArticles(ctx context.Context, articleID, categoryID, seriesID, limit int) ([]*ArticleCard, error)
	GetArticleByID(ctx context.Context, articleID int) (Article, error)
	GetPreArticleByID(ctx context.Context, articleID int) (ArticleCard, error)
	GetNextArticleByID(ctx context.Context, articleID int) (ArticleCard, error)
	GetFirstArticle(ctx context.Context) (ArticleCard, error)
	GetLastArticle(ctx context.Context) (ArticleCard, error)
	ListArticlesByTagID(ctx context.Context, current, size, tagID int) ([]*ArticleCard, int, error)
	ListArticlesByTagName(ctx context.Context, current, size int, name string) ([]*ArticleCard, int, error)
	ListArchives(ctx context.Context, current, size int) ([]ArticleCard, int, error)
	CountArticleAdmins(ctx context.Context, filter ArticleFilter) (int, error)
	ListArticlesAdmin(ctx context.Context, filter ArticleFilter) ([]*ArticleAdmin, error)
	ListArticleStatistics(ctx context.Context) ([]ArticleStatistics, error)
	GetArticleSearchDocument(ctx context.Context, articleID int) (ArticleSearch, bool, error)
	ListPublicArticleSearchDocuments(ctx context.Context) ([]ArticleSearch, error)
	GetArticleRecord(ctx context.Context, articleID int) (entity.TArticle, error)
	SaveOrUpdate(ctx context.Context, article entity.TArticle, categoryName string, tagNames []string) (entity.TArticle, error)
	UpdateTopAndFeatured(ctx context.Context, articleID, isTop, isFeatured int) (entity.TArticle, error)
	UpdateDelete(ctx context.Context, ids []int, isDelete int) error
	Delete(ctx context.Context, ids []int) error
	GetAdminArticle(ctx context.Context, articleID int) (entity.TArticle, string, []string, error)
	Export(ctx context.Context, ids []int) ([]entity.TArticle, error)
}

const (
	ScheduledNotificationPending    = "pending"
	ScheduledNotificationQueued     = "queued"
	ScheduledNotificationFailed     = "failed"
	ScheduledNotificationSuppressed = "suppressed"
)

type ScheduledPublish struct {
	RecordID             int
	ArticleID            int
	UserID               int
	ScheduledAt          time.Time
	PublishedAt          time.Time
	ModerationStatus     string
	NotificationState    string
	NotificationAttempts int
	NextRetryAt          *time.Time
	LastError            string
}

type ScheduledPublishRepository interface {
	PublishDueScheduledArticles(ctx context.Context, now time.Time, limit int) ([]ScheduledPublish, error)
	ListRetryableScheduledPublishes(ctx context.Context, now time.Time, limit int) ([]ScheduledPublish, error)
	MarkScheduledNotificationQueued(ctx context.Context, recordID int, at time.Time) error
	MarkScheduledNotificationFailed(ctx context.Context, recordID int, message string, retryAt *time.Time) error
	MarkScheduledNotificationSuppressed(ctx context.Context, recordID int, message string) error
}
