package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type PlatformService interface {
	Feed(c *gin.Context) model.ResultVO
	Authors(c *gin.Context) model.ResultVO
	Author(c *gin.Context) model.ResultVO
	AuthorArticles(c *gin.Context) model.ResultVO
	AuthorTalks(c *gin.Context) model.ResultVO
	AuthorSeries(c *gin.Context) model.ResultVO
	TopicArticles(c *gin.Context) model.ResultVO
	Topics(c *gin.Context) model.ResultVO
	Dashboard(c *gin.Context) model.ResultVO
	SyncActivation(c *gin.Context) model.ResultVO
	GetProfile(c *gin.Context) model.ResultVO
	UpdateProfile(c *gin.Context) model.ResultVO

	ListOwnedArticles(c *gin.Context) model.ResultVO
	GetOwnedArticle(c *gin.Context) model.ResultVO
	SaveOwnedArticle(c *gin.Context) model.ResultVO
	DeleteOwnedArticles(c *gin.Context) model.ResultVO

	ListOwnedTalks(c *gin.Context) model.ResultVO
	GetOwnedTalk(c *gin.Context) model.ResultVO
	SaveOwnedTalk(c *gin.Context) model.ResultVO
	DeleteOwnedTalks(c *gin.Context) model.ResultVO

	ListOwnedSeries(c *gin.Context) model.ResultVO
	GetOwnedSeries(c *gin.Context) model.ResultVO
	SaveOwnedSeries(c *gin.Context) model.ResultVO
	DeleteOwnedSeries(c *gin.Context) model.ResultVO
	BatchPreviewContent(c *gin.Context) model.ResultVO
	BatchUpdateContentStatus(c *gin.Context) model.ResultVO
	BatchDeleteContent(c *gin.Context) model.ResultVO
	RetryScheduledPublication(c *gin.Context) model.ResultVO

	ListOwnedCategories(c *gin.Context) model.ResultVO
	SaveOwnedCategory(c *gin.Context) model.ResultVO
	DeleteOwnedCategory(c *gin.Context) model.ResultVO
	ListOwnedTags(c *gin.Context) model.ResultVO
	SaveOwnedTag(c *gin.Context) model.ResultVO
	DeleteOwnedTag(c *gin.Context) model.ResultVO

	Moderate(c *gin.Context) model.ResultVO
	DistributeArticle(c *gin.Context) model.ResultVO
	Upload(c *gin.Context) model.ResultVO
}

// reactionCounter and commentCounter are the narrow read shapes the public
// discovery payloads need. They are optional dependencies: when a deployment
// does not wire them the feed still renders, only the counters stay at zero.
type reactionCounter interface {
	Counts(ctx context.Context, articleIDs []int) (map[int]port.ReactionCounts, error)
}

type commentCounter interface {
	ListCommentCountsByTypeAndTopicIDs(ctx context.Context, commentType int, topicIDs []int) ([]*port.CommentCount, error)
}

type PlatformServiceDeps struct {
	Repo        port.PlatformRepository
	Articles    port.ArticleRepository
	Newsletter  articleDistributor
	Storage     port.ObjectStorage
	Cache       port.Cache
	Reactions   reactionCounter
	Comments    commentCounter
	SearchIndex ArticleSearchMaintainer
}

type articleDistributor interface {
	EnqueueArticle(context.Context, int) error
}

type MyPlatformService struct {
	repo        port.PlatformRepository
	articles    port.ArticleRepository
	newsletter  articleDistributor
	storage     port.ObjectStorage
	cache       port.Cache
	reactions   reactionCounter
	comments    commentCounter
	searchIndex ArticleSearchMaintainer
}

func NewPlatformService(deps PlatformServiceDeps) (*MyPlatformService, error) {
	if deps.Repo == nil || deps.Articles == nil || deps.Storage == nil {
		return nil, fmt.Errorf("platform service dependencies are incomplete")
	}
	return &MyPlatformService{
		repo: deps.Repo, articles: deps.Articles, newsletter: deps.Newsletter,
		storage: deps.Storage, cache: deps.Cache, reactions: deps.Reactions, comments: deps.Comments,
		searchIndex: deps.SearchIndex,
	}, nil
}

func (s *MyPlatformService) syncArticleSearch(ctx context.Context, articleIDs ...int) {
	if s.searchIndex == nil || len(articleIDs) == 0 {
		return
	}
	if err := s.searchIndex.Sync(ctx, articleIDs...); err != nil {
		slog.WarnContext(ctx, "sync platform article search index failed", "articleIds", articleIDs, "error", err)
	}
}

func (s *MyPlatformService) platformRepo() port.PlatformRepository { return s.repo }
