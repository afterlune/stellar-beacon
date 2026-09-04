package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeAgentReviewPublisher struct {
	input  port.AgentReviewPublication
	result port.AgentReviewPublicationResult
	err    error
	calls  int
}

func (f *fakeAgentReviewPublisher) Publish(_ context.Context, input port.AgentReviewPublication) (port.AgentReviewPublicationResult, error) {
	f.calls++
	f.input = input
	return f.result, f.err
}

func TestAIStudioAgentApprovalPublishesThroughApplicationBoundary(t *testing.T) {
	reviewID := "agent-review-1"
	reviews := &fakeAIReviewRepository{reviews: map[string]port.AIReview{
		reviewID: {
			ID:              reviewID,
			TargetType:      "article",
			TargetID:        "42",
			Operation:       port.AgentBehaviorOperationComment,
			Content:         "候选评论",
			Status:          port.ReviewPending,
			RunID:           "run-1",
			AgentID:         "42",
			SourceArticleID: 42,
		},
	}}
	publisher := &fakeAgentReviewPublisher{result: port.AgentReviewPublicationResult{
		Status:    port.ReviewPublishSucceeded,
		ContentID: "comment-9",
	}}
	service, err := NewAIStudioService(AIStudioServiceDeps{Reviews: reviews, Publisher: publisher})
	if err != nil {
		t.Fatal(err)
	}
	ctx := aiStudioContext("POST", "/admin/ai/reviews/"+reviewID+"/approve", `{}`)
	ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
	result := service.AcceptReview(ctx)
	if !result.Flag {
		t.Fatalf("unexpected approval result: %+v", result)
	}
	stored := reviews.reviews[reviewID]
	if publisher.calls != 1 || publisher.input.Content != "候选评论" || stored.Status != port.ReviewApproved || stored.PublishStatus != port.ReviewPublishSucceeded || stored.PublishedContentID != "comment-9" {
		t.Fatalf("publisher calls=%d input=%+v review=%+v", publisher.calls, publisher.input, stored)
	}
}

func TestAIStudioAgentApprovalKeepsPendingReviewWhenPublicationFails(t *testing.T) {
	reviewID := "agent-review-2"
	reviews := &fakeAIReviewRepository{reviews: map[string]port.AIReview{
		reviewID: {
			ID:        reviewID,
			Operation: port.AgentBehaviorOperationTalk,
			Content:   "候选说说",
			Status:    port.ReviewPending,
			RunID:     "run-2",
			AgentID:   "42",
		},
	}}
	publisher := &fakeAgentReviewPublisher{err: apperrors.Unavailable("test.publisher", nil)}
	service, err := NewAIStudioService(AIStudioServiceDeps{Reviews: reviews, Publisher: publisher})
	if err != nil {
		t.Fatal(err)
	}
	ctx := aiStudioContext("POST", "/admin/ai/reviews/"+reviewID+"/approve", `{}`)
	ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
	result := service.AcceptReview(ctx)
	if result.Flag {
		t.Fatal("publication failure was reported as success")
	}
	stored := reviews.reviews[reviewID]
	if stored.Status != port.ReviewPending || stored.PublishStatus != port.ReviewPublishFailed || publisher.calls != 1 {
		t.Fatalf("publisher calls=%d review=%+v", publisher.calls, stored)
	}
}

func TestAIStudioExpiresPendingReviewThroughAuditedRepository(t *testing.T) {
	reviewID := "review-expired"
	reviews := &fakeAIReviewRepository{reviews: map[string]port.AIReview{
		reviewID: {ID: reviewID, Status: port.ReviewPending, ExpiresAt: time.Now().Add(-time.Minute)},
	}}
	service, err := NewAIStudioService(AIStudioServiceDeps{Reviews: reviews})
	if err != nil {
		t.Fatal(err)
	}
	ctx := aiStudioContext("POST", "/admin/ai/reviews/"+reviewID+"/expire", `{}`)
	ctx.Params = gin.Params{{Key: "id", Value: reviewID}}
	result := service.ExpireReview(ctx)
	if !result.Flag || reviews.reviews[reviewID].Status != port.ReviewExpired {
		t.Fatalf("unexpected expire result=%+v review=%+v", result, reviews.reviews[reviewID])
	}
	if len(reviews.actions) != 1 || reviews.actions[0].Action != port.ReviewActionExpired {
		t.Fatalf("expire actions=%+v", reviews.actions)
	}
}

type publisherArticleReaderFake struct {
	article entity.TArticle
	err     error
}

func (f publisherArticleReaderFake) GetAdminArticle(context.Context, int) (entity.TArticle, string, []string, error) {
	return f.article, "", nil, f.err
}

func TestNewAgentReviewPublisherRequiresAllApplicationCollaborators(t *testing.T) {
	if _, err := NewAgentReviewPublisher(AgentReviewPublisherDeps{}); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

type trackingAgentCommentRepository struct {
	fakeCommentRepository
	validatedArticle int
	created          entity.TComment
}

func (r *trackingAgentCommentRepository) ValidateTarget(_ context.Context, commentType, topicID int) error {
	if commentType != 1 {
		return apperrors.Invalid("test.comment", "unexpected comment type")
	}
	r.validatedArticle = topicID
	return nil
}

func (r *trackingAgentCommentRepository) Create(_ context.Context, comment entity.TComment) error {
	r.created = comment
	return nil
}

type trackingAgentTalkRepository struct {
	fakeTalkRepository
	created entity.TTalk
}

func (r *trackingAgentTalkRepository) SaveOrUpdate(_ context.Context, talk entity.TTalk) error {
	r.created = talk
	return nil
}

func TestAgentReviewPublisherUsesExistingApplicationPublishUseCases(t *testing.T) {
	commentRepo := &trackingAgentCommentRepository{}
	commentService, err := NewCommentService(CommentServiceDeps{Repo: commentRepo, Website: fakeBenetnaschInfoService{}})
	if err != nil {
		t.Fatal(err)
	}
	talkRepo := &trackingAgentTalkRepository{}
	talkService, err := NewTalkService(TalkServiceDeps{Repo: talkRepo, Comments: &fakeCommentRepository{}, Storage: fakeServiceStorage{}})
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := NewAgentReviewPublisher(AgentReviewPublisherDeps{
		Articles: publisherArticleReaderFake{article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0}},
		Comments: commentService,
		Talks:    talkService,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	commentResult, err := publisher.Publish(ctx, port.AgentReviewPublication{
		Review:  port.AIReview{AgentID: "42", Operation: port.AgentBehaviorOperationComment, SourceArticleID: 7},
		Content: "批准的评论",
	})
	if err != nil || commentResult.Status != port.ReviewPublishSucceeded || commentRepo.validatedArticle != 7 || commentRepo.created.UserId != 42 || commentRepo.created.TopicId != 7 || commentRepo.created.IsReview != 1 {
		t.Fatalf("comment publication result=%+v error=%v stored=%+v", commentResult, err, commentRepo.created)
	}
	talkResult, err := publisher.Publish(ctx, port.AgentReviewPublication{
		Review:  port.AIReview{AgentID: "42", Operation: port.AgentBehaviorOperationTalk, SourceArticleID: 7},
		Content: "批准的说说",
	})
	if err != nil || talkResult.Status != port.ReviewPublishSucceeded || talkRepo.created.UserId != 42 || talkRepo.created.Content != "批准的说说" || talkRepo.created.Status != 1 {
		t.Fatalf("talk publication result=%+v error=%v stored=%+v", talkResult, err, talkRepo.created)
	}
}

func TestAgentReviewPublisherSkipsWakeSuggestionAndBlocksPrivateSource(t *testing.T) {
	commentService, err := NewCommentService(CommentServiceDeps{Repo: &fakeCommentRepository{}, Website: fakeBenetnaschInfoService{}})
	if err != nil {
		t.Fatal(err)
	}
	talkService, err := NewTalkService(TalkServiceDeps{Repo: &fakeTalkRepository{}, Comments: &fakeCommentRepository{}, Storage: fakeServiceStorage{}})
	if err != nil {
		t.Fatal(err)
	}
	privatePublisher, err := NewAgentReviewPublisher(AgentReviewPublisherDeps{
		Articles: publisherArticleReaderFake{article: entity.TArticle{Id: 7, Status: 2, IsDelete: 0}},
		Comments: commentService,
		Talks:    talkService,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := privatePublisher.Publish(context.Background(), port.AgentReviewPublication{
		Review:  port.AIReview{AgentID: "42", Operation: port.AgentBehaviorOperationWake, SourceArticleID: 7},
		Content: "唤醒建议",
	}); err != nil || result.Status != port.ReviewPublishSkipped {
		t.Fatalf("wake publication result=%+v error=%v", result, err)
	}
	if _, err := privatePublisher.Publish(context.Background(), port.AgentReviewPublication{
		Review:  port.AIReview{AgentID: "42", Operation: port.AgentBehaviorOperationComment, SourceArticleID: 7},
		Content: "不应发布",
	}); err == nil || !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("private source was not blocked: %v", err)
	}
}
