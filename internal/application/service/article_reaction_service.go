package service

import (
	"context"
	"strconv"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

// reactionStateLookupLimit bounds one batch lookup to avoid an unbounded scan.
const reactionStateLookupLimit = 100

type ArticleReactionService interface {
	ToggleArticleReaction(context.Context, int, int, string, bool) (port.ReactionToggleResult, error)
	ListMyArticleReactions(context.Context, int, int, int, string) ([]*port.ArticleCard, int64, error)
	ListArticleReactionStates(context.Context, int, []int) ([]port.ArticleReactionState, error)
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

func (s *MyArticleReactionService) ToggleArticleReaction(ctx context.Context, userInfoID, articleID int, reaction string, desired bool) (port.ReactionToggleResult, error) {
	if articleID <= 0 || !port.IsReactionKind(reaction) {
		return port.ReactionToggleResult{}, apperrors.Invalid("article_reaction.toggle", "unsupported reaction")
	}
	if userInfoID <= 0 {
		return port.ReactionToggleResult{}, apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil)
	}
	allowed, err := allowRateLimit(ctx, s.limiter, "article-reaction:", strconv.Itoa(userInfoID), 30, time.Minute)
	if err != nil {
		return port.ReactionToggleResult{}, apperrors.Unavailable("article_reaction.rate_limit", err)
	}
	if !allowed {
		return port.ReactionToggleResult{}, apperrors.Invalid("article_reaction.rate_limit", "too many reactions")
	}
	if err := s.ensureArticle(ctx, articleID); err != nil {
		return port.ReactionToggleResult{}, err
	}
	states, err := s.repo.States(ctx, userInfoID, []int{articleID})
	if err != nil {
		return port.ReactionToggleResult{}, err
	}
	active := states[articleID][reaction]
	if active != desired {
		if active, err = s.repo.Toggle(ctx, articleID, userInfoID, reaction); err != nil {
			return port.ReactionToggleResult{}, err
		}
	}
	counts, err := s.repo.Counts(ctx, []int{articleID})
	if err != nil {
		return port.ReactionToggleResult{}, err
	}
	totals := counts[articleID]
	return port.ReactionToggleResult{Active: active, LikeCount: totals.LikeCount, FavoriteCount: totals.FavoriteCount}, nil
}

func (s *MyArticleReactionService) ListMyArticleReactions(ctx context.Context, userInfoID, current, size int, reaction string) ([]*port.ArticleCard, int64, error) {
	if !port.IsReactionKind(reaction) {
		return nil, 0, apperrors.Invalid("article_reaction.list", "unsupported reaction")
	}
	if userInfoID <= 0 {
		return nil, 0, apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil)
	}
	articleIDs, total, err := s.repo.ListArticleIDsByUser(ctx, userInfoID, reaction, current, size)
	if err != nil {
		return nil, 0, err
	}
	if len(articleIDs) == 0 {
		return []*port.ArticleCard{}, 0, nil
	}
	cards, err := s.articles.ListArticleCardsByIDs(ctx, articleIDs)
	if err != nil {
		return nil, 0, err
	}
	ordered := orderArticleCards(articleIDs, cards)
	if err := s.attachReactionCounts(ctx, ordered); err != nil {
		return nil, 0, err
	}
	return ordered, total, nil
}

func (s *MyArticleReactionService) ListArticleReactionStates(ctx context.Context, userInfoID int, articleIDs []int) ([]port.ArticleReactionState, error) {
	if len(articleIDs) == 0 {
		return nil, apperrors.Invalid("article_reaction.states", "articleIds is required")
	}
	if len(articleIDs) > reactionStateLookupLimit {
		return nil, apperrors.Invalid("article_reaction.states", "too many article ids")
	}
	for _, id := range articleIDs {
		if id <= 0 {
			return nil, apperrors.Invalid("article_reaction.states", "invalid article id")
		}
	}
	if userInfoID <= 0 {
		return nil, apperrors.New(apperrors.KindUnauthorized, "article_reaction.user", nil)
	}
	states, err := s.repo.States(ctx, userInfoID, articleIDs)
	if err != nil {
		return nil, err
	}
	result := make([]port.ArticleReactionState, 0, len(articleIDs))
	for _, id := range articleIDs {
		entry := states[id]
		result = append(result, port.ArticleReactionState{ArticleId: id, Like: entry[port.ReactionLike], Favorite: entry[port.ReactionFavorite]})
	}
	return result, nil
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
