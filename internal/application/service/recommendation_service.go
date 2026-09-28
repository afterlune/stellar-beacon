package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

const (
	recommendationDefaultSize = 12
	recommendationMaxSize     = 24
	recommendationMaxSeeds    = 20
	recommendationCursorV1    = 1
)

type RecommendationService interface {
	Query(c *gin.Context) model.ResultVO
	ListFeedback(c *gin.Context) model.ResultVO
	SaveFeedback(c *gin.Context) model.ResultVO
	DeleteFeedback(c *gin.Context) model.ResultVO
}

type MyRecommendationService struct {
	repo      port.RecommendationRepository
	reactions port.ArticleReactionRepository
}

func NewRecommendationService(repo port.RecommendationRepository, reactions port.ArticleReactionRepository) (*MyRecommendationService, error) {
	if repo == nil {
		return nil, missingServiceDependency("recommendation", "repository")
	}
	if reactions == nil {
		return nil, missingServiceDependency("recommendation", "reaction repository")
	}
	return &MyRecommendationService{repo: repo, reactions: reactions}, nil
}

func encodeRecommendationCursor(cursor *port.RecommendationCursor) (string, error) {
	if cursor == nil {
		return "", nil
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeRecommendationCursor(value string) (*port.RecommendationCursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	var cursor port.RecommendationCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, err
	}
	if cursor.Version != recommendationCursorV1 || cursor.Snapshot.IsZero() || cursor.ArticleID <= 0 || cursor.Window < 0 || cursor.Window > 1 || cursor.Score < 0 {
		return nil, apperrors.Invalid("recommendation.cursor", "invalid cursor")
	}
	return &cursor, nil
}

func normalizedRecommendationSeeds(values []int) ([]int, bool) {
	result := make([]int, 0, len(values))
	seen := map[int]struct{}{}
	for _, value := range values {
		if value <= 0 {
			return nil, false
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) > recommendationMaxSeeds {
		return nil, false
	}
	return result, true
}

func (s *MyRecommendationService) Query(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.RecommendationQueryVO
	if err := c.ShouldBindJSON(&vo); err != nil && err != io.EOF {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Size == 0 {
		vo.Size = recommendationDefaultSize
	}
	if vo.Size < 1 || vo.Size > recommendationMaxSize {
		return model.ResultFailWithMessage("每页数量必须在 1 到 24 之间")
	}
	seeds, valid := normalizedRecommendationSeeds(vo.SeedArticleIDs)
	if !valid {
		return model.ResultFailWithMessage("阅读种子最多 20 个且必须是有效文章")
	}
	cursor, err := decodeRecommendationCursor(vo.Cursor)
	if err != nil {
		return model.ResultFailWithMessage("推荐游标不正确")
	}
	// Content timestamps use PostgreSQL TIMESTAMP WITHOUT TIME ZONE and are
	// written in the application/database local timezone. Keep the ranking
	// snapshot in that same wall-clock timezone so today's content is not
	// treated as future-dated when the process timezone is not UTC.
	snapshot := time.Now()
	if cursor != nil {
		snapshot = cursor.Snapshot.In(time.Local)
	}
	page, err := s.repo.ListRecommendations(c.Request.Context(), port.RecommendationRequest{
		UserID: user.UserInfoId, SeedArticleIDs: seeds, Size: vo.Size, Snapshot: snapshot, Cursor: cursor,
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	s.attachRecommendationReactionCounts(c.Request.Context(), page.Items)
	nextCursor, err := encodeRecommendationCursor(page.NextCursor)
	if err != nil {
		return model.ResultFromError(apperrors.Unavailable("recommendation.cursor", err))
	}
	return model.ResultOkWithData(model.RecommendationFeedDTO{
		Items: page.Items, NextCursor: nextCursor, HasMore: page.HasMore, Personalized: page.Personalized,
	})
}

func (s *MyRecommendationService) attachRecommendationReactionCounts(ctx context.Context, items []port.RecommendationItem) {
	if len(items) == 0 || s.reactions == nil {
		return
	}
	ids := make([]int, 0, len(items))
	for _, item := range items {
		if item.Id > 0 {
			ids = append(ids, item.Id)
		}
	}
	counts, err := s.reactions.Counts(ctx, ids)
	if err != nil {
		return
	}
	for index := range items {
		if value, ok := counts[items[index].Id]; ok {
			items[index].LikeCount = value.LikeCount
			items[index].FavoriteCount = value.FavoriteCount
		}
	}
}

func (s *MyRecommendationService) ListFeedback(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(c)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	items, count, err := s.repo.ListFeedback(c.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: items, Count: count, Page: current, PageSize: size})
}

func (s *MyRecommendationService) SaveFeedback(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.RecommendationFeedbackVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	target := strings.ToLower(strings.TrimSpace(vo.TargetType))
	if !port.ValidRecommendationTarget(target) {
		return model.ResultFailWithMessage("推荐偏好类型不正确")
	}
	item, err := s.repo.UpsertFeedback(c.Request.Context(), user.UserInfoId, port.RecommendationFeedbackInput{
		TargetType: target, ArticleID: vo.ArticleID, AuthorID: vo.AuthorID,
		TopicType: strings.ToLower(strings.TrimSpace(vo.TopicType)), TopicKey: strings.TrimSpace(vo.TopicKey),
	})
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("推荐偏好目标不存在")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(item)
}

func (s *MyRecommendationService) DeleteFeedback(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	feedbackID, err := pathID(c, "feedbackId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.repo.DeleteFeedback(c.Request.Context(), user.UserInfoId, feedbackID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
