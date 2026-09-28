package service

import (
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) Feed(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryFeedSort(c.Query("sort"))
	if !ok {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	kind := strings.TrimSpace(c.Query("type"))
	featuredOnly := c.Query("featured") == "1" || c.Query("featured") == "true"
	if kind == "talk" {
		talks, count, err := s.platformRepo().ListFeedTalks(c.Request.Context(), current, size)
		if err != nil {
			return model.ResultFromError(err)
		}
		if len(talks) == 0 {
			return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
		}
		s.attachTalkCommentCounts(c.Request.Context(), talks)
		return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
	}
	var articles []*port.ArticleCard
	var count int
	switch sort {
	case port.FeedSortHot:
		articles, count, err = s.platformRepo().ListFeedArticlesHot(c.Request.Context(), current, size)
	case port.FeedSortFeatured:
		articles, count, err = s.platformRepo().ListFeedArticles(c.Request.Context(), current, size, true)
	default:
		articles, count, err = s.platformRepo().ListFeedArticles(c.Request.Context(), current, size, featuredOnly)
	}
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(articles) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

// discoveryFeedSort validates the feed sort key. An empty value keeps the
// historical recency ordering so existing callers are unaffected.
func discoveryFeedSort(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", port.FeedSortLatest:
		return port.FeedSortLatest, true
	case port.FeedSortHot, port.FeedSortFeatured:
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}

func (s *MyPlatformService) Authors(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryAuthorSort(c.Query("sort"))
	if !ok {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	authors, count, err := s.platformRepo().ListAuthors(c.Request.Context(), current, size, optionalUserID(c), sort)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: authors, Count: count})
}

// discoveryAuthorSort validates the author-board ordering key. The historical
// article-count order stays the default for existing callers.
func discoveryAuthorSort(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", port.AuthorSortArticles:
		return port.AuthorSortArticles, true
	case port.AuthorSortFollowers, port.AuthorSortActive:
		return strings.ToLower(strings.TrimSpace(value)), true
	default:
		return "", false
	}
}
func (s *MyPlatformService) Author(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(author)
}

func (s *MyPlatformService) AuthorArticles(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	sort, ok := discoveryFeedSort(c.Query("sort"))
	if !ok || sort == port.FeedSortFeatured {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var articles []*port.ArticleCard
	var count int
	if sort == port.FeedSortHot {
		articles, count, err = s.platformRepo().ListAuthorArticlesHot(c.Request.Context(), author.Id, current, size)
	} else {
		articles, count, err = s.platformRepo().ListAuthorArticles(c.Request.Context(), author.Id, current, size)
	}
	if err != nil {
		return model.ResultFromError(err)
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

func (s *MyPlatformService) AuthorTalks(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	talks, count, err := s.platformRepo().ListAuthorTalks(c.Request.Context(), author.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: talks, Count: count})
}

func (s *MyPlatformService) AuthorSeries(c *gin.Context) model.ResultVO {
	author, err := s.platformRepo().GetAuthorByHandle(c.Request.Context(), c.Param("handle"), optionalUserID(c))
	if err != nil {
		return model.ResultFromError(err)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	series, count, err := s.platformRepo().ListAuthorSeries(c.Request.Context(), author.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: series, Count: count})
}

func (s *MyPlatformService) TopicArticles(c *gin.Context) model.ResultVO {
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	articles, count, err := s.platformRepo().ListTopicArticles(c.Request.Context(), c.Param("topic"), c.Param("slug"), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	s.attachArticleReactionCounts(c.Request.Context(), articles)
	return model.ResultOkWithData(model.PageResultDTO{Records: articles, Count: count})
}

// Topic overview sizes are clamped instead of rejected: the plaza is a
// presentational surface, so an odd size should degrade to a usable page.
const (
	topicOverviewSizeDefault = 12
	topicOverviewSizeMax     = 30
)

func (s *MyPlatformService) Topics(c *gin.Context) model.ResultVO {
	size := topicOverviewSizeDefault
	if raw := strings.TrimSpace(c.Query("size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			size = parsed
		}
	}
	if size > topicOverviewSizeMax {
		size = topicOverviewSizeMax
	}
	overview, err := s.platformRepo().ListTopicOverview(c.Request.Context(), size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(overview)
}

// attachArticleReactionCounts decorates discovery cards with like/favourite
// totals. Counts are presentational: a ledger failure is logged and leaves the
// zero value instead of failing the whole read.
func (s *MyPlatformService) attachArticleReactionCounts(ctx context.Context, cards []*port.ArticleCard) {
	if s.reactions == nil || len(cards) == 0 {
		return
	}
	ids := make([]int, 0, len(cards))
	for _, card := range cards {
		if card != nil && card.Id > 0 {
			ids = append(ids, card.Id)
		}
	}
	counts, err := s.reactions.Counts(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "load discovery reaction counts failed", "error", err)
		return
	}
	for _, card := range cards {
		if card == nil {
			continue
		}
		totals := counts[card.Id]
		card.LikeCount = totals.LikeCount
		card.FavoriteCount = totals.FavoriteCount
	}
}

// attachTalkCommentCounts decorates feed talks with their approved-comment
// totals using the same aggregate the talk service publishes.
func (s *MyPlatformService) attachTalkCommentCounts(ctx context.Context, talks []*port.Talk) {
	if s.comments == nil || len(talks) == 0 {
		return
	}
	ids := make([]int, 0, len(talks))
	for _, talk := range talks {
		if talk != nil && talk.Id > 0 {
			ids = append(ids, talk.Id)
		}
	}
	counts, err := s.comments.ListCommentCountsByTypeAndTopicIDs(ctx, commentTypeTalk, ids)
	if err != nil {
		slog.WarnContext(ctx, "load discovery comment counts failed", "error", err)
		return
	}
	byTopic := make(map[int]int, len(counts))
	for _, count := range counts {
		if count != nil {
			byTopic[count.Id] = count.CommentCount
		}
	}
	for _, talk := range talks {
		if talk == nil {
			continue
		}
		talk.CommentCount = byTopic[talk.Id]
	}
}
