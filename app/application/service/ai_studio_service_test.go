package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type fakeWritingGateway struct {
	response port.WritingResponse
	err      error
	requests []port.WritingRequest
}

func (f *fakeWritingGateway) Generate(_ context.Context, request port.WritingRequest) (port.WritingResponse, error) {
	f.requests = append(f.requests, request)
	return f.response, f.err
}

type fakeAIReviewRepository struct {
	reviews      map[string]port.AIReview
	actions      []port.AIReviewAction
	expireOneErr error
}

type fakeAIReviewPolicyRepository struct {
	policy port.AgentReviewPolicy
	err    error
}

func (f *fakeAIReviewPolicyRepository) Get(context.Context, string) (port.AgentReviewPolicy, error) {
	if f.err != nil {
		return port.AgentReviewPolicy{}, f.err
	}
	return f.policy, nil
}

func (f *fakeAIReviewPolicyRepository) Save(context.Context, port.AgentReviewPolicy) error {
	return f.err
}

func (f *fakeAIReviewRepository) Create(_ context.Context, review port.AIReview) error {
	if f.reviews == nil {
		f.reviews = make(map[string]port.AIReview)
	}
	if _, exists := f.reviews[review.ID]; exists {
		return apperrors.Conflict("ai_review.create", "review already exists")
	}
	f.reviews[review.ID] = review
	return nil
}

func (f *fakeAIReviewRepository) Get(_ context.Context, id string) (port.AIReview, error) {
	review, ok := f.reviews[id]
	if !ok {
		return port.AIReview{}, apperrors.NotFound("ai_review.get")
	}
	return review, nil
}

func (f *fakeAIReviewRepository) List(_ context.Context, filter port.ReviewFilter) ([]port.AIReview, int, error) {
	result := make([]port.AIReview, 0)
	for _, review := range f.reviews {
		if filter.Status != "" && review.Status != filter.Status {
			continue
		}
		result = append(result, review)
	}
	return result, len(result), nil
}

func (f *fakeAIReviewRepository) ApplyAction(_ context.Context, id string, action port.AIReviewAction) error {
	review, ok := f.reviews[id]
	if !ok {
		return apperrors.NotFound("ai_review.action")
	}
	if review.Status != port.ReviewPending {
		return apperrors.Conflict("ai_review.action", "review is not pending")
	}
	switch action.Action {
	case port.ReviewActionAccepted:
		review.Status = port.ReviewApproved
	case port.ReviewActionPartiallyAccepted:
		review.Status = port.ReviewPartiallyApproved
	case port.ReviewActionRejected:
		review.Status = port.ReviewRejected
		review.RejectReason = action.Content
	case port.ReviewActionRegenerated:
	default:
		return apperrors.Invalid("ai_review.action", "unsupported action")
	}
	review.ReviewerID = action.ActorID
	review.UpdatedAt = action.CreatedAt
	f.reviews[id] = review
	f.actions = append(f.actions, action)
	return nil
}

func (f *fakeAIReviewRepository) Approve(context.Context, string, string) error { return nil }
func (f *fakeAIReviewRepository) Reject(context.Context, string, string, string) error {
	return nil
}
func (f *fakeAIReviewRepository) Expire(context.Context, time.Time) (int, error) { return 0, nil }

func (f *fakeAIReviewRepository) ClaimPublication(_ context.Context, id string, action port.AIReviewAction) (bool, error) {
	review, ok := f.reviews[id]
	if !ok {
		return false, apperrors.NotFound("ai_review.claim_publication")
	}
	if review.Status != port.ReviewPending {
		return false, apperrors.Conflict("ai_review.claim_publication", "review is no longer pending")
	}
	if review.PublishStatus == port.ReviewPublishProcessing {
		if review.PublicationKey == action.IdempotencyKey {
			return false, nil
		}
		return false, apperrors.Conflict("ai_review.claim_publication", "publication is already processing")
	}
	review.PublishStatus = port.ReviewPublishProcessing
	review.PublicationKey = action.IdempotencyKey
	f.reviews[id] = review
	return true, nil
}

func (f *fakeAIReviewRepository) FinalizePublication(_ context.Context, id string, action port.AIReviewAction, publication port.AIReviewPublication) error {
	review, ok := f.reviews[id]
	if !ok {
		return apperrors.NotFound("ai_review.finalize_publication")
	}
	if review.Status != port.ReviewPending {
		if review.PublicationKey == publication.PublicationKey && review.PublishStatus == publication.Status {
			return nil
		}
		return apperrors.Conflict("ai_review.finalize_publication", "review is no longer pending")
	}
	if action.Action == port.ReviewActionPartiallyAccepted {
		review.Status = port.ReviewPartiallyApproved
	} else {
		review.Status = port.ReviewApproved
	}
	review.ReviewerID = action.ActorID
	review.PublishStatus = publication.Status
	review.PublishedContentID = publication.ContentID
	review.PublicationKey = publication.PublicationKey
	review.PublishedAt = publication.PublishedAt
	review.UpdatedAt = action.CreatedAt
	f.reviews[id] = review
	f.actions = append(f.actions, action)
	return nil
}

func (f *fakeAIReviewRepository) RecordPublicationFailure(_ context.Context, id string, action port.AIReviewAction, reason string) error {
	review, ok := f.reviews[id]
	if !ok {
		return apperrors.NotFound("ai_review.publish_failure")
	}
	review.PublishStatus = port.ReviewPublishFailed
	review.PublishError = reason
	review.PublicationKey = action.IdempotencyKey
	review.UpdatedAt = action.CreatedAt
	f.reviews[id] = review
	return nil
}

func (f *fakeAIReviewRepository) ExpireOne(_ context.Context, id, actorID string, now time.Time) error {
	if f.expireOneErr != nil {
		return f.expireOneErr
	}
	review, ok := f.reviews[id]
	if !ok {
		return apperrors.NotFound("ai_review.expire_one")
	}
	if review.Status == port.ReviewExpired {
		return nil
	}
	if review.Status != port.ReviewPending || review.ExpiresAt.IsZero() || review.ExpiresAt.After(now) {
		return apperrors.Conflict("ai_review.expire_one", "review cannot be expired")
	}
	review.Status = port.ReviewExpired
	review.UpdatedAt = now
	f.reviews[id] = review
	f.actions = append(f.actions, port.AIReviewAction{ReviewID: id, Action: port.ReviewActionExpired, ActorID: actorID})
	return nil
}

func newAIStudioServiceForTest(t *testing.T, writing port.WritingGateway, reviews *fakeAIReviewRepository) *MyAIStudioService {
	t.Helper()
	service, err := NewAIStudioService(AIStudioServiceDeps{Writing: writing, Reviews: reviews})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func aiStudioContext(method, target, body string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	return serviceTestRequest{ginContextForServiceTest: c}
}

func TestAIStudioPreviewCreatesPendingReviewWithoutSavingArticle(t *testing.T) {
	writing := &fakeWritingGateway{response: port.WritingResponse{
		Operation: port.WritingOperationPolish,
		RunID:     "run-1",
		Preview:   "生成后的内容",
		Diff:      "-旧内容\n+生成后的内容",
	}}
	reviews := &fakeAIReviewRepository{}
	service := newAIStudioServiceForTest(t, writing, reviews)
	result := service.PreviewWriting(aiStudioContext(http.MethodPost, "/admin/ai/writing/preview", `{"articleId":42,"operation":"polish","title":"标题","content":"旧内容"}`))
	if !result.Flag {
		t.Fatalf("unexpected preview result: %+v", result)
	}
	preview, ok := result.Data.(model.AIWritingPreviewDTO)
	if !ok || preview.ReviewID == "" || preview.RunID != "run-1" || preview.Preview != "生成后的内容" {
		t.Fatalf("preview = %#v", result.Data)
	}
	if len(writing.requests) != 1 || writing.requests[0].Operation != port.WritingOperationPolish {
		t.Fatalf("writing requests = %#v", writing.requests)
	}
	review, ok := reviews.reviews[preview.ReviewID]
	if !ok || review.Status != port.ReviewPending || review.TargetType != "article" || review.TargetID != "42" || review.ReviewerID != "7" {
		t.Fatalf("stored review = %#v", review)
	}
}

func TestAIStudioPreviewUsesCurrentReviewPolicyTTL(t *testing.T) {
	writing := &fakeWritingGateway{response: port.WritingResponse{
		Operation: port.WritingOperationPolish,
		RunID:     "run-policy",
		Preview:   "策略有效期测试内容",
	}}
	reviews := &fakeAIReviewRepository{}
	policy := port.DefaultAgentReviewPolicy()
	policy.ReviewTTL = 2 * time.Hour
	service, err := NewAIStudioService(AIStudioServiceDeps{
		Writing:      writing,
		Reviews:      reviews,
		ReviewPolicy: &fakeAIReviewPolicyRepository{policy: policy},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := service.PreviewWriting(aiStudioContext(http.MethodPost, "/admin/ai/writing/preview", `{"operation":"polish","content":"旧内容"}`))
	if !result.Flag {
		t.Fatalf("unexpected preview result: %+v", result)
	}
	preview := result.Data.(model.AIWritingPreviewDTO)
	stored := reviews.reviews[preview.ReviewID]
	remaining := time.Until(stored.ExpiresAt)
	if remaining < 2*time.Hour-time.Minute || remaining > 2*time.Hour+time.Minute {
		t.Fatalf("review expiry = %v, want about two hours", remaining)
	}
}

func TestAIStudioActionsAreAuditedAndDoNotSaveArticle(t *testing.T) {
	for _, test := range []struct {
		name       string
		actionPath string
		body       string
		status     port.ReviewStatus
		content    string
	}{
		{name: "accept", actionPath: "approve", status: port.ReviewApproved},
		{name: "partial", actionPath: "partial", body: `{"content":"人工保留的片段"}`, status: port.ReviewPartiallyApproved, content: "人工保留的片段"},
		{name: "reject", actionPath: "reject", body: `{"rejectReason":"事实不准确"}`, status: port.ReviewRejected, content: "事实不准确"},
		{name: "regenerate", actionPath: "regenerate", body: `{"runId":"run-2"}`, status: port.ReviewPending},
	} {
		t.Run(test.name, func(t *testing.T) {
			reviewID := "review-" + test.name
			reviews := &fakeAIReviewRepository{reviews: map[string]port.AIReview{
				reviewID: {ID: reviewID, Status: port.ReviewPending, RunID: "run-1"},
			}}
			service := newAIStudioServiceForTest(t, &fakeWritingGateway{}, reviews)
			ctx := aiStudioContext(http.MethodPost, "/admin/ai/reviews/"+reviewID+"/"+test.actionPath, test.body)
			ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
			result := service.applyReviewAction(ctx, func() port.ReviewActionType {
				switch test.actionPath {
				case "approve":
					return port.ReviewActionAccepted
				case "partial":
					return port.ReviewActionPartiallyAccepted
				case "reject":
					return port.ReviewActionRejected
				default:
					return port.ReviewActionRegenerated
				}
			}())
			if !result.Flag || len(reviews.actions) != 1 {
				t.Fatalf("action result=%+v actions=%+v", result, reviews.actions)
			}
			stored := reviews.reviews[reviewID]
			if stored.Status != test.status || stored.ReviewerID != "7" {
				t.Fatalf("stored review = %+v", stored)
			}
			if test.content != "" && reviews.actions[0].Content != test.content {
				t.Fatalf("action content = %q, want %q", reviews.actions[0].Content, test.content)
			}
		})
	}
}

func TestAIStudioPropagatesExpiredReviewPersistenceFailure(t *testing.T) {
	reviewID := "expired-review"
	wantErr := apperrors.Unavailable("ai_review.expire_one", errors.New("database unavailable"))
	reviews := &fakeAIReviewRepository{
		reviews: map[string]port.AIReview{
			reviewID: {
				ID:        reviewID,
				Status:    port.ReviewPending,
				ExpiresAt: time.Now().UTC().Add(-time.Minute),
			},
		},
		expireOneErr: wantErr,
	}
	service := newAIStudioServiceForTest(t, &fakeWritingGateway{}, reviews)
	ctx := aiStudioContext(http.MethodPost, "/admin/ai/reviews/"+reviewID+"/approve", `{}`)
	ctx.Params = gin.Params{{Key: "id", Value: reviewID}}

	result := service.AcceptReview(ctx)
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("expired persistence failure result = %+v", result)
	}
	if reviews.reviews[reviewID].Status != port.ReviewPending {
		t.Fatalf("review status changed after failed expiration persistence: %s", reviews.reviews[reviewID].Status)
	}
}

func TestAIStudioRejectsInvalidReviewInputsBeforeGatewayOrRepository(t *testing.T) {
	writing := &fakeWritingGateway{}
	reviews := &fakeAIReviewRepository{}
	service := newAIStudioServiceForTest(t, writing, reviews)
	result := service.PreviewWriting(aiStudioContext(http.MethodPost, "/admin/ai/writing/preview", `{"operation":"invalid","content":"content"}`))
	if result.Flag || len(writing.requests) != 0 || len(reviews.reviews) != 0 {
		t.Fatalf("invalid preview was not rejected: %+v", result)
	}

	result = service.PartiallyAcceptReview(aiStudioContext(http.MethodPost, "/admin/ai/reviews/missing/partial", `{}`))
	if result.Flag || !apperrors.IsKind(apperrors.Invalid("test", "x"), apperrors.KindValidation) {
		t.Fatalf("invalid partial action result: %+v", result)
	}
}

func TestAIReviewDTOIsJSONCompatible(t *testing.T) {
	payload, err := json.Marshal(model.NewAIReviewDTO(port.AIReview{ID: "review-1", Status: port.ReviewPending}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"id":"review-1"`) || !strings.Contains(string(payload), `"status":"pending"`) {
		t.Fatalf("unexpected review JSON: %s", payload)
	}
}

func TestAIStudioDoesNotRepublishWhenPublicationClaimIsAlreadyHeld(t *testing.T) {
	reviewID := "agent-review-processing"
	publicationKey := "review:" + reviewID + ":accepted:run-1"
	reviews := &fakeAIReviewRepository{reviews: map[string]port.AIReview{
		reviewID: {
			ID:             reviewID,
			Status:         port.ReviewPending,
			AgentID:        "42",
			Operation:      port.AgentBehaviorOperationTalk,
			RunID:          "run-1",
			PublishStatus:  port.ReviewPublishProcessing,
			PublicationKey: publicationKey,
		},
	}}
	publisher := &fakeAgentReviewPublisher{result: port.AgentReviewPublicationResult{Status: port.ReviewPublishSucceeded, ContentID: "talk-1"}}
	service, err := NewAIStudioService(AIStudioServiceDeps{Reviews: reviews, Publisher: publisher})
	if err != nil {
		t.Fatal(err)
	}
	ctx := aiStudioContext(http.MethodPost, "/admin/ai/reviews/"+reviewID+"/approve", `{}`)
	ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
	result := service.AcceptReview(ctx)
	if result.Flag || publisher.calls != 0 {
		t.Fatalf("claim was bypassed: result=%+v calls=%d", result, publisher.calls)
	}
}
