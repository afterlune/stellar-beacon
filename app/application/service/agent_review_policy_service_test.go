package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type reviewPolicyRepositoryFake struct {
	policy port.AgentReviewPolicy
	saved  port.AgentReviewPolicy
}

func (f *reviewPolicyRepositoryFake) Get(context.Context, string) (port.AgentReviewPolicy, error) {
	return f.policy, nil
}

func (f *reviewPolicyRepositoryFake) Save(_ context.Context, policy port.AgentReviewPolicy) error {
	f.saved = policy
	policy.Version++
	f.policy = policy
	return nil
}

func reviewPolicyContext(method, body string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, "/admin/ai/review-policy", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return serviceTestRequest{ginContextForServiceTest: c}
}

func TestAgentReviewPolicyServiceUpdatesVersionedSafetyGates(t *testing.T) {
	policy := port.DefaultAgentReviewPolicy()
	repo := &reviewPolicyRepositoryFake{policy: policy}
	service := NewAgentReviewPolicyService(repo, policy.ID)
	body, _ := json.Marshal(map[string]any{
		"version":             policy.Version,
		"reviewTtlSeconds":    int64(2 * time.Hour / time.Second),
		"maxCandidateRunes":   800,
		"similarityThreshold": 0.9,
		"dailyLimit":          5,
		"perArticleLimit":     2,
		"perActionLimit":      4,
		"allowedActions":      []string{"comment", "talk"},
		"sensitivePatterns":   []string{"internal-token"},
	})
	result := service.Update(reviewPolicyContext(http.MethodPatch, string(body)))
	if !result.Flag {
		t.Fatalf("Update() failed: %+v", result)
	}
	dto, ok := result.Data.(model.AgentReviewPolicyDTO)
	if !ok || dto.Version != 2 || dto.ReviewTTLSeconds != int64(2*time.Hour/time.Second) || !dto.ReviewRequired || dto.SimilarityThreshold != 0.9 {
		t.Fatalf("updated policy = %#v", result.Data)
	}
	if repo.saved.ReviewRequired == false || len(repo.saved.AllowedActions) != 2 {
		t.Fatalf("saved policy bypassed safety invariant: %+v", repo.saved)
	}
}

func TestAgentReviewPolicyServiceRejectsInvalidValues(t *testing.T) {
	policy := port.DefaultAgentReviewPolicy()
	service := NewAgentReviewPolicyService(&reviewPolicyRepositoryFake{policy: policy}, policy.ID)
	result := service.Update(reviewPolicyContext(http.MethodPatch, `{"version":1,"reviewTtlSeconds":60,"similarityThreshold":0}`))
	if result.Flag {
		t.Fatalf("invalid policy unexpectedly succeeded: %+v", result)
	}
}
