package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

// reactionStateLookupLimit bounds the batch size of the state endpoint so one
// request cannot scan an unbounded id list.
const reactionStateLookupLimit = 100

// ArticleReactionService owns the reader-interaction use cases. Every method
// requires an authenticated account; the middleware enforces that for the
// /v1/auth/me routes.
type ArticleReactionService interface {
	ToggleArticleReaction(c *gin.Context) model.ResultVO
	ListMyArticleReactions(c *gin.Context) model.ResultVO
	ListArticleReactionStates(c *gin.Context) model.ResultVO
}

type MyArticleReactionService struct {
	repo     port.ArticleReactionRepository
	articles port.ArticleRepository
	limiter  port.RateLimiter
}

func NewArticleReactionService(deps ArticleReactionServiceDeps) (*MyArticleReactionService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyArticleReactionService{repo: deps.Repo, articles: deps.Articles, limiter: deps.Limiter}, nil
}

func (s *MyArticleReactionService) ToggleArticleReaction(c *gin.Context) model.ResultVO {
	var vo model.ReactionToggleVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.ArticleId <= 0 || !port.IsReactionKind(vo.Reaction) {
		return model.ResultFromError(apperrors.Invalid("article_reaction.toggle", "unsupported reaction"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil))
	}
	if allowed, err := allowRateLimit(c.Request.Context(), s.limiter, "article-reaction:", strconv.Itoa(userInfoID), 30, time.Minute); err != nil {
		return model.ResultFromError(apperrors.Unavailable("article_reaction.rate_limit", err))
	} else if !allowed {
		return model.ResultFromError(apperrors.Invalid("article_reaction.rate_limit", "too many reactions"))
	}
	if err := s.ensureArticle(c.Request.Context(), vo.ArticleId); err != nil {
		return model.ResultFromError(err)
	}
	states, err := s.repo.States(c.Request.Context(), userInfoID, []int{vo.ArticleId})
	if err != nil {
		return model.ResultFromError(err)
	}
	// The client sends the state it wants, so a retry cannot flip the reaction
	// back; only a real difference reaches the ledger.
	active := states[vo.ArticleId][vo.Reaction]
	if active != vo.Active {
		if active, err = s.repo.Toggle(c.Request.Context(), vo.ArticleId, userInfoID, vo.Reaction); err != nil {
			return model.ResultFromError(err)
		}
	}
	counts, err := s.repo.Counts(c.Request.Context(), []int{vo.ArticleId})
	if err != nil {
		return model.ResultFromError(err)
	}
	totals := counts[vo.ArticleId]
	return model.ResultOkWithData(model.ReactionToggleDTO{
		Active:        active,
		LikeCount:     totals.LikeCount,
		FavoriteCount: totals.FavoriteCount,
	})
}

func (s *MyArticleReactionService) ListMyArticleReactions(c *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	reaction := c.Query("reaction")
	if !port.IsReactionKind(reaction) {
		return model.ResultFromError(apperrors.Invalid("article_reaction.list", "unsupported reaction"))
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil))
	}
	articleIDs, total, err := s.repo.ListArticleIDsByUser(c.Request.Context(), userInfoID, reaction, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(articleIDs) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: []*port.ArticleCard{}, Count: 0})
	}
	cards, err := s.articles.ListArticleCardsByIDs(c.Request.Context(), articleIDs)
	if err != nil {
		return model.ResultFromError(err)
	}
	ordered := orderArticleCards(articleIDs, cards)
	if err := s.attachReactionCounts(c.Request.Context(), ordered); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: ordered, Count: int(total)})
}

func (s *MyArticleReactionService) ListArticleReactionStates(c *gin.Context) model.ResultVO {
	raw := strings.TrimSpace(c.Query("articleIds"))
	if raw == "" {
		return model.ResultFromError(apperrors.Invalid("article_reaction.states", "articleIds is required"))
	}
	parts := strings.Split(raw, ",")
	if len(parts) > reactionStateLookupLimit {
		return model.ResultFromError(apperrors.Invalid("article_reaction.states", "too many article ids"))
	}
	articleIDs := make([]int, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return model.ResultFromError(apperrors.Invalid("article_reaction.states", "invalid article id"))
		}
		articleIDs = append(articleIDs, id)
	}
	userInfoID, ok := currentUserInfoID(c)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil))
	}
	states, err := s.repo.States(c.Request.Context(), userInfoID, articleIDs)
	if err != nil {
		return model.ResultFromError(err)
	}
	result := make([]model.ReactionStateDTO, 0, len(articleIDs))
	for _, id := range articleIDs {
		entry := states[id]
		result = append(result, model.ReactionStateDTO{ArticleId: id, Like: entry[port.ReactionLike], Favorite: entry[port.ReactionFavorite]})
	}
	return model.ResultOkWithData(result)
}

// ensureArticle rejects reactions on missing or recycled articles.
func (s *MyArticleReactionService) ensureArticle(ctx context.Context, articleID int) error {
	article, err := s.articles.GetArticleRecord(ctx, articleID)
	if err != nil {
		return err
	}
	if article.Id == 0 || article.IsDelete != 0 {
		return apperrors.NotFound("article_reaction.article")
	}
	return nil
}

func (s *MyArticleReactionService) attachReactionCounts(ctx context.Context, cards []*port.ArticleCard) error {
	if len(cards) == 0 {
		return nil
	}
	ids := make([]int, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.Id)
	}
	counts, err := s.repo.Counts(ctx, ids)
	if err != nil {
		return err
	}
	for _, card := range cards {
		totals := counts[card.Id]
		card.LikeCount = totals.LikeCount
		card.FavoriteCount = totals.FavoriteCount
	}
	return nil
}

// orderArticleCards restores the ledger order (newest reaction first) because
// the card query returns rows in id order.
func orderArticleCards(articleIDs []int, cards []*port.ArticleCard) []*port.ArticleCard {
	byID := make(map[int]*port.ArticleCard, len(cards))
	for _, card := range cards {
		byID[card.Id] = card
	}
	ordered := make([]*port.ArticleCard, 0, len(cards))
	for _, id := range articleIDs {
		if card, ok := byID[id]; ok {
			ordered = append(ordered, card)
		}
	}
	return ordered
}

// currentUserInfoID reads the profile id attached by the login middleware.
func currentUserInfoID(c *gin.Context) (int, bool) {
	value, ok := c.Get("userInfo")
	if !ok {
		return 0, false
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok || dto.UserInfoId <= 0 {
		return 0, false
	}
	return dto.UserInfoId, true
}
