package port

import (
	"context"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// StudioFilter scopes owner-facing content lists. UserID is always supplied by
// the authenticated server context and never trusted from a request payload.
type StudioFilter struct {
	Current  int
	Size     int
	Status   int
	Keywords string
}

type StudioDashboard struct {
	ArticleCount  int `json:"articleCount"`
	DraftCount    int `json:"draftCount"`
	PrivateCount  int `json:"privateCount"`
	TalkCount     int `json:"talkCount"`
	SeriesCount   int `json:"seriesCount"`
	FavoriteCount int `json:"favoriteCount"`
}

type StudioProfile struct {
	Handle   string
	Nickname string
	Avatar   string
	Intro    string
	Website  string
}
type AuthorCard struct {
	PublicAuthor
	ArticleCount int `json:"articleCount"`
	TalkCount    int `json:"talkCount"`
	SeriesCount  int `json:"seriesCount"`
}

// PlatformRepository owns cross-owner discovery and owner-scoped editing. It
// deliberately keeps ownership predicates inside persistence methods so callers
// cannot forget a user_id filter.
type PlatformRepository interface {
	GetAuthorByHandle(ctx context.Context, handle string) (AuthorCard, error)
	ListAuthors(ctx context.Context, current, size int) ([]*AuthorCard, int, error)
	StudioDashboard(ctx context.Context, userID int) (StudioDashboard, error)
	GetStudioProfile(ctx context.Context, userID int) (StudioProfile, error)
	UpdateAuthorProfile(ctx context.Context, userID int, handle, nickname, intro, website string) error

	ListFeedArticles(ctx context.Context, current, size int, featuredOnly bool) ([]*ArticleCard, int, error)
	ListFeedTalks(ctx context.Context, current, size int) ([]*Talk, int, error)
	ListAuthorArticles(ctx context.Context, userID, current, size int) ([]*ArticleCard, int, error)
	ListAuthorTalks(ctx context.Context, userID, current, size int) ([]*Talk, int, error)
	ListAuthorSeries(ctx context.Context, userID, current, size int) ([]*Series, int, error)
	ListTopicArticles(ctx context.Context, topic string, slug string, current, size int) ([]*ArticleCard, int, error)

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

	ListOwnedCategories(ctx context.Context, userID int) ([]*Category, error)
	SaveOwnedCategory(ctx context.Context, category entity.TCategory) (entity.TCategory, error)
	DeleteOwnedCategory(ctx context.Context, userID, categoryID int) error

	ListOwnedTags(ctx context.Context, userID int) ([]*Tag, error)
	SaveOwnedTag(ctx context.Context, tag entity.TTag) (entity.TTag, error)
	DeleteOwnedTag(ctx context.Context, userID, tagID int) error

	ModerateContent(ctx context.Context, contentType string, id, adminID int, hidden bool, reason string) error
}
