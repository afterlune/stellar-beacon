package port

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// StudioFilter scopes owner-facing content lists. UserID is always supplied by
// the authenticated server context and never trusted from a request payload.
type StudioFilter struct {
	Current  int
	Size     int
	Status   int
	SeriesID int
	Keywords string
}

type StudioContentType string

const (
	StudioContentArticle StudioContentType = "article"
	StudioContentTalk    StudioContentType = "talk"
	StudioContentSeries  StudioContentType = "series"
)

const (
	StudioBatchScopeIDs    = "ids"
	StudioBatchScopeFilter = "filter"
)

type StudioBatchScope struct {
	Mode          string `json:"mode"`
	IDs           []int  `json:"ids,omitempty"`
	Status        int    `json:"status,omitempty"`
	Keywords      string `json:"keywords,omitempty"`
	SeriesID      int    `json:"seriesId,omitempty"`
	MaxID         int    `json:"maxId,omitempty"`
	ExcludeIDs    []int  `json:"excludeIds,omitempty"`
	ExpectedCount int    `json:"expectedCount,omitempty"`
}

type StudioBatchPreviewItem struct {
	ID               int    `json:"id"`
	Title            string `json:"title"`
	Status           int    `json:"status"`
	ModerationStatus string `json:"moderationStatus"`
}

type StudioBatchPreview struct {
	Count        int                      `json:"count"`
	MaxID        int                      `json:"maxId"`
	StatusCounts map[string]int           `json:"statusCounts"`
	HiddenCount  int                      `json:"hiddenCount"`
	Sample       []StudioBatchPreviewItem `json:"sample"`
}

type StudioAuditActor struct {
	UserID    int
	Nickname  string
	IPAddress string
	IPSource  string
}

type StudioBatchMutation struct {
	Affected   int   `json:"affected"`
	AuditID    int   `json:"auditId"`
	ContentIDs []int `json:"-"`
}

type StudioDashboard struct {
	ArticleCount   int `json:"articleCount"`
	DraftCount     int `json:"draftCount"`
	PrivateCount   int `json:"privateCount"`
	TalkCount      int `json:"talkCount"`
	SeriesCount    int `json:"seriesCount"`
	FavoriteCount  int `json:"favoriteCount"`
	FollowerCount  int `json:"followerCount"`
	FollowingCount int `json:"followingCount"`
}

type StudioProfile struct {
	Handle   string
	Nickname string
	Avatar   string
	Intro    string
	Website  string
}

// Public discovery sort keys. The service layer validates query parameters
// against these constants so an unexpected value can never reach a SQL
// builder; unknown values fall back to the documented default.
const (
	FeedSortLatest   = "latest"
	FeedSortHot      = "hot"
	FeedSortFeatured = "featured"
)

const (
	AuthorSortArticles  = "articles"
	AuthorSortFollowers = "followers"
	AuthorSortActive    = "active"
)

// TopicOverviewItem is one public taxonomy entry (category, tag or series)
// ranked by its recent reader activity.
type TopicOverviewItem struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Cover        string `json:"cover,omitempty"`
	ArticleCount int    `json:"articleCount"`
	HotScore     int    `json:"hotScore"`
}

// TopicOverview groups the public taxonomy surfaces rendered by the topic
// plaza. Each group is already ranked and truncated by the repository.
type TopicOverview struct {
	Categories []TopicOverviewItem `json:"categories"`
	Tags       []TopicOverviewItem `json:"tags"`
	Series     []TopicOverviewItem `json:"series"`
}
type AuthorCard struct {
	PublicAuthor
	ArticleCount    int  `json:"articleCount"`
	TalkCount       int  `json:"talkCount"`
	SeriesCount     int  `json:"seriesCount"`
	CollectionCount int  `json:"collectionCount"`
	FollowerCount   int  `json:"followerCount"`
	IsFollowing     bool `json:"isFollowing,omitempty"`
	// LastPublishedAt is only filled for the active-author ranking, where the
	// reader needs to see why an author is listed as recently active.
	LastPublishedAt *time.Time `json:"lastPublishedAt,omitempty"`
}

// PlatformRepository owns cross-owner discovery and owner-scoped editing. It
// deliberately keeps ownership predicates inside persistence methods so callers
// cannot forget a user_id filter.
type PlatformRepository interface {
	GetAuthorByHandle(ctx context.Context, handle string, viewerID int) (AuthorCard, error)
	ListAuthors(ctx context.Context, current, size, viewerID int, sort string) ([]*AuthorCard, int, error)
	StudioDashboard(ctx context.Context, userID int) (StudioDashboard, error)
	GetStudioProfile(ctx context.Context, userID int) (StudioProfile, error)
	UpdateAuthorProfile(ctx context.Context, userID int, handle, nickname, intro, website string) error

	ListFeedArticles(ctx context.Context, current, size int, featuredOnly bool) ([]*ArticleCard, int, error)
	ListFeedArticlesHot(ctx context.Context, current, size int) ([]*ArticleCard, int, error)
	ListFeedTalks(ctx context.Context, current, size int) ([]*Talk, int, error)
	ListAuthorArticles(ctx context.Context, userID, current, size int) ([]*ArticleCard, int, error)
	ListAuthorArticlesHot(ctx context.Context, userID, current, size int) ([]*ArticleCard, int, error)
	ListAuthorTalks(ctx context.Context, userID, current, size int) ([]*Talk, int, error)
	ListAuthorSeries(ctx context.Context, userID, current, size int) ([]*Series, int, error)
	ListTopicArticles(ctx context.Context, topic string, slug string, current, size int) ([]*ArticleCard, int, error)
	ListTopicOverview(ctx context.Context, size int) (TopicOverview, error)

	ListOwnedArticles(ctx context.Context, userID int, filter StudioFilter) ([]*ArticleAdmin, int, error)
	GetOwnedArticle(ctx context.Context, userID, articleID int) (ArticleAdminView, error)
	SaveOwnedArticle(ctx context.Context, userID int, article entity.TArticle, categoryID int, tagIDs []int) (entity.TArticle, error)
	DeleteOwnedArticles(ctx context.Context, userID int, ids []int) error

	ListOwnedTalks(ctx context.Context, userID int, filter StudioFilter) ([]*TalkAdmin, int, error)
	GetOwnedTalk(ctx context.Context, userID, talkID int) (entity.TTalk, error)
	SaveOwnedTalk(ctx context.Context, talk entity.TTalk) (entity.TTalk, error)
	DeleteOwnedTalks(ctx context.Context, userID int, ids []int) error

	ListOwnedSeries(ctx context.Context, userID int, filter StudioFilter) ([]*Series, int, error)
	GetOwnedSeries(ctx context.Context, userID, seriesID int) (entity.TSeries, error)
	SaveOwnedSeries(ctx context.Context, series entity.TSeries) (entity.TSeries, error)
	DeleteOwnedSeries(ctx context.Context, userID, seriesID int) error
	PreviewOwnedContent(ctx context.Context, userID int, contentType StudioContentType, filter StudioFilter) (StudioBatchPreview, error)
	BatchUpdateOwnedContentStatus(ctx context.Context, userID int, contentType StudioContentType, scope StudioBatchScope, status int, actor StudioAuditActor) (StudioBatchMutation, error)
	BatchDeleteOwnedContent(ctx context.Context, userID int, contentType StudioContentType, scope StudioBatchScope, actor StudioAuditActor) (StudioBatchMutation, error)
	RetryScheduledPublication(ctx context.Context, userID, articleID int, actor StudioAuditActor) (ScheduledPublish, error)

	ListOwnedCategories(ctx context.Context, userID int) ([]*Category, error)
	SaveOwnedCategory(ctx context.Context, category entity.TCategory) (entity.TCategory, error)
	DeleteOwnedCategory(ctx context.Context, userID, categoryID int) error

	ListOwnedTags(ctx context.Context, userID int) ([]*Tag, error)
	SaveOwnedTag(ctx context.Context, tag entity.TTag) (entity.TTag, error)
	DeleteOwnedTag(ctx context.Context, userID, tagID int) error

	ModerateContent(ctx context.Context, contentType string, id, adminID int, hidden bool, reason string) error
}
