package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type fakeRecommendationRepository struct {
	port.RecommendationRepository
	request      port.RecommendationRequest
	page         port.RecommendationPage
	feedback     port.RecommendationFeedbackInput
	feedbackPage []port.RecommendationFeedback
	deleted      int
}

func (f *fakeRecommendationRepository) ListRecommendations(_ context.Context, request port.RecommendationRequest) (port.RecommendationPage, error) {
	f.request = request
	return f.page, nil
}

func (f *fakeRecommendationRepository) ListFeedback(context.Context, int, int, int) ([]port.RecommendationFeedback, int, error) {
	return f.feedbackPage, len(f.feedbackPage), nil
}

func (f *fakeRecommendationRepository) UpsertFeedback(_ context.Context, _ int, input port.RecommendationFeedbackInput) (port.RecommendationFeedback, error) {
	f.feedback = input
	return port.RecommendationFeedback{ID: 1, TargetType: input.TargetType, Label: "fixture"}, nil
}

func (f *fakeRecommendationRepository) DeleteFeedback(_ context.Context, _ int, feedbackID int) error {
	f.deleted = feedbackID
	return nil
}

func TestRecommendationQueryNormalizesSeedsAndAttachesCounts(t *testing.T) {
	repo := &fakeRecommendationRepository{page: port.RecommendationPage{
		Items: []port.RecommendationItem{{ArticleCard: port.ArticleCard{Id: 4}}}, Personalized: true,
		NextCursor: &port.RecommendationCursor{Version: 1, Snapshot: time.Now().UTC(), Window: 0, Score: 2, PublishedAt: time.Now().UTC(), ArticleID: 4},
	}}
	reactions := &fakeArticleReactionRepository{counts: map[int]port.ReactionCounts{4: {LikeCount: 3, FavoriteCount: 2}}}
	service, err := NewRecommendationService(repo, reactions)
	if err != nil {
		t.Fatal(err)
	}
	result := service.Query(platformTestContext(http.MethodPost, "/v1/auth/me/recommendations/query", `{"size":6,"seedArticleIds":[4,4,5]}`))
	if !result.Flag {
		t.Fatalf("query failed: %+v", result)
	}
	if repo.request.UserID != 7 || repo.request.Size != 6 || len(repo.request.SeedArticleIDs) != 2 || repo.request.SeedArticleIDs[0] != 4 {
		t.Fatalf("unexpected recommendation request: %+v", repo.request)
	}
	page, ok := result.Data.(model.RecommendationFeedDTO)
	if !ok || page.Items[0].LikeCount != 3 || page.Items[0].FavoriteCount != 2 || page.NextCursor == "" {
		t.Fatalf("unexpected recommendation response: %+v", result.Data)
	}
}

func TestRecommendationQueryRejectsTooManySeeds(t *testing.T) {
	repo := &fakeRecommendationRepository{}
	service, err := NewRecommendationService(repo, &fakeArticleReactionRepository{})
	if err != nil {
		t.Fatal(err)
	}
	seeds := make([]int, 21)
	for index := range seeds {
		seeds[index] = index + 1
	}
	payload, err := json.Marshal(model.RecommendationQueryVO{Size: 12, SeedArticleIDs: seeds})
	if err != nil {
		t.Fatal(err)
	}
	body := string(payload)
	if result := service.Query(platformTestContext(http.MethodPost, "/v1/auth/me/recommendations/query", body)); result.Flag {
		t.Fatal("more than twenty seeds must be rejected")
	}
}

func TestRecommendationFeedbackMapsExplicitTargets(t *testing.T) {
	repo := &fakeRecommendationRepository{}
	service, err := NewRecommendationService(repo, &fakeArticleReactionRepository{})
	if err != nil {
		t.Fatal(err)
	}
	result := service.SaveFeedback(platformTestContext(http.MethodPut, "/v1/auth/me/recommendation-feedback", `{"targetType":"topic","topicType":"tag","topicKey":" Go "}`))
	if !result.Flag || repo.feedback.TargetType != port.RecommendationTargetTopic || repo.feedback.TopicKey != "Go" {
		t.Fatalf("unexpected feedback mapping: result=%+v feedback=%+v", result, repo.feedback)
	}
	ctx := platformTestContext(http.MethodDelete, "/v1/auth/me/recommendation-feedback/9", "")
	ctx.Params = []gin.Param{{Key: "feedbackId", Value: "9"}}
	if result := service.DeleteFeedback(ctx); !result.Flag || repo.deleted != 9 {
		t.Fatalf("delete feedback failed: result=%+v deleted=%d", result, repo.deleted)
	}
}
