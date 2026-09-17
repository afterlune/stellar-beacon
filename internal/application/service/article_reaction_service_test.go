package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type fakeReactionRepository struct {
	active   map[string]bool
	counts   map[int]port.ReactionCounts
	toggles  int
	listIDs  []int
	listSize int64
}

func reactionKey(articleID, userInfoID int, reaction string) string {
	return fmt.Sprintf("%s:%d:%d", reaction, articleID, userInfoID)
}

func (f *fakeReactionRepository) Toggle(_ context.Context, articleID, userInfoID int, reaction string) (bool, error) {
	f.toggles++
	if f.active == nil {
		f.active = map[string]bool{}
	}
	key := reactionKey(articleID, userInfoID, reaction)
	f.active[key] = !f.active[key]
	return f.active[key], nil
}

func (f *fakeReactionRepository) Counts(_ context.Context, articleIDs []int) (map[int]port.ReactionCounts, error) {
	result := make(map[int]port.ReactionCounts, len(articleIDs))
	for _, id := range articleIDs {
		result[id] = f.counts[id]
	}
	return result, nil
}

func (f *fakeReactionRepository) States(_ context.Context, userInfoID int, articleIDs []int) (map[int]map[string]bool, error) {
	result := make(map[int]map[string]bool, len(articleIDs))
	for _, id := range articleIDs {
		entry := map[string]bool{}
		for _, kind := range port.ReactionKinds {
			if f.active[reactionKey(id, userInfoID, kind)] {
				entry[kind] = true
			}
		}
		result[id] = entry
	}
	return result, nil
}

func (f *fakeReactionRepository) ListArticleIDsByUser(context.Context, int, string, int, int) ([]int, int64, error) {
	return f.listIDs, f.listSize, nil
}

type fakeReactionArticles struct {
	fakeArticleRepository
	record entity.TArticle
	cards  []*port.ArticleCard
}

func (f *fakeReactionArticles) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.record, nil
}

func (f *fakeReactionArticles) ListArticleCardsByIDs(context.Context, []int) ([]*port.ArticleCard, error) {
	return f.cards, nil
}

type denyingRateLimiter struct{}

func (denyingRateLimiter) Allow(context.Context, string, int64, time.Duration) (bool, error) {
	return false, nil
}

func mustReactionContext(t *testing.T, method, target, body string, userInfoID int) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if body == "" {
		c.Request = httptest.NewRequest(method, target, nil)
	} else {
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if userInfoID > 0 {
		c.Set("userInfo", model.UserDetailsDTO{Id: userInfoID, UserInfoId: userInfoID})
	}
	return c
}

func mustReactionService(t *testing.T, repo port.ArticleReactionRepository, articles port.ArticleRepository, limiter port.RateLimiter) *MyArticleReactionService {
	t.Helper()
	service, err := NewArticleReactionService(ArticleReactionServiceDeps{Repo: repo, Articles: articles, Limiter: limiter})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestToggleArticleReactionIsIdempotentForTheRequestedState(t *testing.T) {
	repo := &fakeReactionRepository{counts: map[int]port.ReactionCounts{7: {LikeCount: 1}}}
	articles := &fakeReactionArticles{record: entity.TArticle{Id: 7}}
	service := mustReactionService(t, repo, articles, nil)

	first := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"like","active":true}`, 3))
	if !first.Flag {
		t.Fatalf("expected success, got %+v", first)
	}
	dto, ok := first.Data.(model.ReactionToggleDTO)
	if !ok || !dto.Active || dto.LikeCount != 1 {
		t.Fatalf("unexpected payload: %+v", first.Data)
	}

	// The desired state is already present: no second ledger write.
	second := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"like","active":true}`, 3))
	if !second.Flag {
		t.Fatalf("expected success, got %+v", second)
	}
	if repo.toggles != 1 {
		t.Fatalf("expected exactly one ledger write, got %d", repo.toggles)
	}

	third := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"like","active":false}`, 3))
	if !third.Flag {
		t.Fatalf("expected success, got %+v", third)
	}
	if active := third.Data.(model.ReactionToggleDTO).Active; active {
		t.Fatal("expected the reaction to be removed")
	}
}

func TestToggleArticleReactionRejectsUnsupportedInput(t *testing.T) {
	service := mustReactionService(t, &fakeReactionRepository{}, &fakeReactionArticles{record: entity.TArticle{Id: 7}}, nil)

	if result := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"clap","active":true}`, 3)); result.Flag {
		t.Fatal("unsupported reaction must fail")
	}
	if result := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"like","active":true}`, 0)); result.Flag {
		t.Fatal("anonymous requests must fail")
	}
}

func TestToggleArticleReactionRejectsMissingArticle(t *testing.T) {
	articles := &fakeReactionArticles{record: entity.TArticle{Id: 0}}
	service := mustReactionService(t, &fakeReactionRepository{}, articles, nil)

	result := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":9,"reaction":"like","active":true}`, 3))
	if result.Flag {
		t.Fatal("reactions on a missing article must fail")
	}
}

func TestToggleArticleReactionEnforcesRateLimit(t *testing.T) {
	articles := &fakeReactionArticles{record: entity.TArticle{Id: 7}}
	service := mustReactionService(t, &fakeReactionRepository{}, articles, denyingRateLimiter{})

	result := service.ToggleArticleReaction(mustReactionContext(t, http.MethodPut, "/v1/auth/me/reactions", `{"articleId":7,"reaction":"like","active":true}`, 3))
	if result.Flag {
		t.Fatal("rate-limited requests must fail")
	}
}

func TestArticleReactionServiceRejectsMissingDependencies(t *testing.T) {
	if _, err := NewArticleReactionService(ArticleReactionServiceDeps{}); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
