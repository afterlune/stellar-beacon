package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
)

const (
	aiReviewTargetArticle = "article"
	aiReviewTargetDraft   = "draft"
	maxReviewTargetIDLen  = 128
	maxReviewActionRunID  = 128
	maxReviewTextRunes    = 100_000
)

type AIStudioService interface {
	PreviewWriting(port.Request) port.ResultVO
	ListReviews(port.Request) port.ResultVO
	GetReview(port.Request) port.ResultVO
	AcceptReview(port.Request) port.ResultVO
	PartiallyAcceptReview(port.Request) port.ResultVO
	RejectReview(port.Request) port.ResultVO
	RegenerateReview(port.Request) port.ResultVO
	ExpireReview(port.Request) port.ResultVO
}

type AIStudioServiceDeps struct {
	Writing        port.WritingGateway
	Reviews        port.AIReviewRepository
	ReviewPolicy   port.AgentReviewPolicyRepository
	ReviewPolicyID string
	Publisher      port.AgentReviewPublisher
	Dreams         port.DreamRepository
}

type MyAIStudioService struct {
	writing        port.WritingGateway
	reviews        port.AIReviewRepository
	reviewPolicy   port.AgentReviewPolicyRepository
	reviewPolicyID string
	publisher      port.AgentReviewPublisher
	dreams         port.DreamRepository
}

var _ AIStudioService = (*MyAIStudioService)(nil)

func NewAIStudioService(deps AIStudioServiceDeps) (*MyAIStudioService, error) {
	if deps.Reviews == nil {
		return nil, apperrors.Unavailable("service.ai_studio.reviews", errors.New("review repository is not configured"))
	}
	return &MyAIStudioService{
		writing:        deps.Writing,
		reviews:        deps.Reviews,
		reviewPolicy:   deps.ReviewPolicy,
		reviewPolicyID: strings.TrimSpace(deps.ReviewPolicyID),
		publisher:      deps.Publisher,
		dreams:         deps.Dreams,
	}, nil
}

type writingPreviewRequest struct {
	ArticleID   int    `json:"articleId"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	Operation   string `json:"operation" binding:"required"`
	Title       string `json:"title"`
	Content     string `json:"content" binding:"required"`
	Instruction string `json:"instruction"`
}

func (s *MyAIStudioService) PreviewWriting(c port.Request) port.ResultVO {
	if s == nil || s.writing == nil || s.reviews == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.writing", nil))
	}
	var request writingPreviewRequest
	if err := c.BindJSON(&request); err != nil {
		return port.ResultFromError(apperrors.Invalid("agent.writing.request", "writing request is invalid"))
	}
	policy, err := s.currentReviewPolicy(c.Context())
	if err != nil {
		return port.ResultFromError(err)
	}
	actorID, err := reviewActorID(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	sessionID, err := reviewSessionID(c, actorID)
	if err != nil {
		return port.ResultFromError(err)
	}
	targetType, targetID, err := normalizeReviewTarget(request.TargetType, request.TargetID, request.ArticleID)
	if err != nil {
		return port.ResultFromError(err)
	}
	operation, err := port.NormalizeWritingOperation(request.Operation)
	if err != nil {
		return port.ResultFromError(apperrors.Invalid("agent.writing.operation", err.Error()))
	}
	response, err := s.writing.Generate(c.Context(), port.WritingRequest{
		Operation:   operation,
		Title:       request.Title,
		Content:     request.Content,
		Instruction: request.Instruction,
	})
	if err != nil {
		return port.ResultFromError(err)
	}
	runID := strings.TrimSpace(response.RunID)
	if runID == "" {
		runID = uuid.NewString()
	}
	now := time.Now().UTC()
	review := port.AIReview{
		ID:            uuid.NewString(),
		TargetType:    targetType,
		TargetID:      targetID,
		SessionID:     sessionID,
		ContentDigest: port.ReviewContentDigest(response.Preview),
		BoundAt:       now,
		Operation:     string(operation),
		Content:       response.Preview,
		Diff:          response.Diff,
		Status:        port.ReviewPending,
		RunID:         runID,
		ReviewerID:    actorID,
		CreatedAt:     now,
		UpdatedAt:     now,
		ExpiresAt:     now.Add(policy.ReviewTTL),
	}
	if err := s.reviews.Create(c.Context(), review); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(port.AIWritingPreviewDTO{
		ReviewID:  review.ID,
		RunID:     runID,
		Operation: string(operation),
		Preview:   response.Preview,
		Diff:      response.Diff,
	})
}

func (s *MyAIStudioService) currentReviewPolicy(ctx context.Context) (port.AgentReviewPolicy, error) {
	policy := port.DefaultAgentReviewPolicy()
	if s == nil || s.reviewPolicy == nil {
		return policy, nil
	}
	stored, err := s.reviewPolicy.Get(ctx, strings.TrimSpace(s.reviewPolicyID))
	if err != nil {
		return port.AgentReviewPolicy{}, err
	}
	policy, err = port.NormalizeAgentReviewPolicy(stored)
	if err != nil {
		return port.AgentReviewPolicy{}, apperrors.Unavailable("agent.review_policy.validate", err)
	}
	return policy, nil
}

func (s *MyAIStudioService) GetReview(c port.Request) port.ResultVO {
	if s == nil || s.reviews == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.review", nil))
	}
	reviewID, err := reviewIDParam(c, "agent.review.get")
	if err != nil {
		return port.ResultFromError(err)
	}
	review, err := s.reviews.Get(c.Context(), reviewID)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(port.NewAIReviewDTO(review))
}

func (s *MyAIStudioService) ListReviews(c port.Request) port.ResultVO {
	if s == nil || s.reviews == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.review", nil))
	}
	current, size, err := pageQuery(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	status := port.ReviewStatus(strings.TrimSpace(c.Query("status")))
	if status != "" {
		if _, err := normalizeReviewStatus(status); err != nil {
			return port.ResultFromError(err)
		}
	}
	reviews, count, err := s.reviews.List(c.Context(), port.ReviewFilter{
		Status:  status,
		Current: current,
		Size:    size,
	})
	if err != nil {
		return port.ResultFromError(err)
	}
	dtos := make([]port.AIReviewDTO, 0, len(reviews))
	for _, review := range reviews {
		dtos = append(dtos, port.NewAIReviewDTO(review))
	}
	if len(dtos) == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: []port.AIReviewDTO{}, Count: 0})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: count})
}

type reviewActionRequest struct {
	Content        string `json:"content"`
	RejectReason   string `json:"rejectReason"`
	RunID          string `json:"runId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (s *MyAIStudioService) AcceptReview(c port.Request) port.ResultVO {
	return s.applyReviewAction(c, port.ReviewActionAccepted)
}

func (s *MyAIStudioService) PartiallyAcceptReview(c port.Request) port.ResultVO {
	return s.applyReviewAction(c, port.ReviewActionPartiallyAccepted)
}

func (s *MyAIStudioService) RejectReview(c port.Request) port.ResultVO {
	return s.applyReviewAction(c, port.ReviewActionRejected)
}

func (s *MyAIStudioService) RegenerateReview(c port.Request) port.ResultVO {
	return s.applyReviewAction(c, port.ReviewActionRegenerated)
}

func (s *MyAIStudioService) ExpireReview(c port.Request) port.ResultVO {
	if s == nil || s.reviews == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.review", nil))
	}
	reviewID, err := reviewIDParam(c, "agent.review.expire")
	if err != nil {
		return port.ResultFromError(err)
	}
	actorID, err := reviewActorID(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	publicationRepo, ok := s.reviews.(port.AIReviewPublicationRepository)
	if !ok {
		return port.ResultFromError(apperrors.Unavailable("agent.review.expire", errors.New("review expiration is not supported")))
	}
	review, err := s.reviews.Get(c.Context(), reviewID)
	if err != nil {
		return port.ResultFromError(err)
	}
	now := time.Now().UTC()
	if err := publicationRepo.ExpireOne(c.Context(), reviewID, actorID, now); err != nil {
		return port.ResultFromError(err)
	}
	if review.Operation == port.AgentDreamOperation && s.dreams != nil {
		if err := s.dreams.SyncReviewStatus(c.Context(), reviewID, port.DreamExpired, now); err != nil && !apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultFromError(err)
		}
	}
	return port.ResultOk()
}

func (s *MyAIStudioService) applyReviewAction(c port.Request, action port.ReviewActionType) port.ResultVO {
	if s == nil || s.reviews == nil {
		return port.ResultFromError(apperrors.Unavailable("agent.review", nil))
	}
	reviewID, err := reviewIDParam(c, "agent.review.action")
	if err != nil {
		return port.ResultFromError(err)
	}
	actorID, err := reviewActorID(c)
	if err != nil {
		return port.ResultFromError(err)
	}
	var request reviewActionRequest
	if c.HTTPRequest().ContentLength != 0 {
		if err := c.BindJSON(&request); err != nil {
			return port.ResultFromError(apperrors.Invalid("agent.review.action", "review action is invalid"))
		}
	}
	request.Content = strings.TrimSpace(request.Content)
	request.RejectReason = strings.TrimSpace(request.RejectReason)
	request.RunID = strings.TrimSpace(request.RunID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	if len([]rune(request.Content)) > maxReviewTextRunes || len([]rune(request.RejectReason)) > maxReviewTextRunes || len(request.RunID) > maxReviewActionRunID || len(request.IdempotencyKey) > 255 {
		return port.ResultFromError(apperrors.Invalid("agent.review.action", "review action value is too long"))
	}
	if action == port.ReviewActionPartiallyAccepted && request.Content == "" {
		return port.ResultFromError(apperrors.Invalid("agent.review.partial", "accepted content is required"))
	}
	if action == port.ReviewActionRejected && request.RejectReason == "" {
		return port.ResultFromError(apperrors.Invalid("agent.review.reject", "reject reason is required"))
	}
	review, err := s.reviews.Get(c.Context(), reviewID)
	if err != nil {
		return port.ResultFromError(err)
	}
	if review.Status == port.ReviewPending && !review.ExpiresAt.IsZero() && !time.Now().UTC().Before(review.ExpiresAt) {
		if publicationRepo, ok := s.reviews.(port.AIReviewPublicationRepository); ok {
			if err := publicationRepo.ExpireOne(c.Context(), reviewID, actorID, time.Now().UTC()); err != nil {
				return port.ResultFromError(err)
			}
		}
		return port.ResultFromError(apperrors.Conflict("agent.review.action", "review has expired"))
	}
	if request.RunID == "" {
		request.RunID = review.RunID
	}
	if action == port.ReviewActionRegenerated && request.RunID == "" && request.IdempotencyKey == "" {
		return port.ResultFromError(apperrors.Invalid("agent.review.regenerate", "run id or idempotency key is required"))
	}
	content := request.Content
	if action == port.ReviewActionRejected {
		content = request.RejectReason
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = fmt.Sprintf("review:%s:%s:%s", reviewID, action, request.RunID)
	}
	if request.IdempotencyKey == "" {
		return port.ResultFromError(apperrors.Invalid("agent.review.action", "idempotency key is required"))
	}
	sessionID, err := reviewSessionID(c, actorID)
	if err != nil {
		return port.ResultFromError(err)
	}
	if isAgentReview(review) && (action == port.ReviewActionAccepted || action == port.ReviewActionPartiallyAccepted) && review.Status != port.ReviewPending {
		if (review.PublishStatus == port.ReviewPublishSucceeded || review.PublishStatus == port.ReviewPublishSkipped) && review.PublicationKey == request.IdempotencyKey {
			return port.ResultOk()
		}
		return port.ResultFromError(apperrors.Conflict("agent.review.publish", "review is no longer pending"))
	}
	contentDigest := strings.TrimSpace(review.ContentDigest)
	if contentDigest == "" {
		contentDigest = port.ReviewContentDigest(review.Content)
	}
	actionInput := port.AIReviewAction{
		ID:             uuid.NewString(),
		ReviewID:       reviewID,
		SessionID:      sessionID,
		TargetType:     review.TargetType,
		TargetID:       review.TargetID,
		ContentDigest:  contentDigest,
		ExpiresAt:      review.ExpiresAt,
		Action:         action,
		ActorID:        actorID,
		Content:        content,
		RunID:          request.RunID,
		IdempotencyKey: request.IdempotencyKey,
		CreatedAt:      time.Now().UTC(),
	}
	if (action == port.ReviewActionAccepted || action == port.ReviewActionPartiallyAccepted) && isAgentReview(review) {
		if s.publisher == nil {
			return port.ResultFromError(apperrors.Unavailable("agent.review.publish", errors.New("agent review publisher is not configured")))
		}
		publicationRepo, ok := s.reviews.(port.AIReviewPublicationRepository)
		if !ok {
			return port.ResultFromError(apperrors.Unavailable("agent.review.publish", errors.New("review publication recording is not supported")))
		}
		claimRepo, ok := s.reviews.(port.AIReviewPublicationClaimRepository)
		if !ok {
			return port.ResultFromError(apperrors.Unavailable("agent.review.publish", errors.New("review publication claiming is not supported")))
		}
		claimed, err := claimRepo.ClaimPublication(c.Context(), reviewID, actionInput)
		if err != nil {
			return port.ResultFromError(err)
		}
		if !claimed {
			latest, getErr := s.reviews.Get(c.Context(), reviewID)
			if getErr == nil && latest.PublicationKey == request.IdempotencyKey &&
				(latest.PublishStatus == port.ReviewPublishSucceeded || latest.PublishStatus == port.ReviewPublishSkipped) {
				return port.ResultOk()
			}
			return port.ResultFromError(apperrors.Conflict("agent.review.publish", "publication is already processing"))
		}
		publication, publishErr := s.publisher.Publish(c.Context(), port.AgentReviewPublication{
			Review:         review,
			Content:        publicationContent(review, actionInput),
			ReviewerID:     actorID,
			IdempotencyKey: request.IdempotencyKey,
		})
		if publishErr != nil {
			failureErr := publicationRepo.RecordPublicationFailure(c.Context(), reviewID, actionInput, "publication_failed")
			if failureErr != nil {
				return port.ResultFromError(failureErr)
			}
			return port.ResultFromError(publishErr)
		}
		if _, err := port.NormalizeReviewPublishStatus(string(publication.Status)); err != nil {
			return port.ResultFromError(apperrors.Invalid("agent.review.publish", "publisher returned invalid status"))
		}
		if publication.Status == port.ReviewPublishSucceeded && strings.TrimSpace(publication.ContentID) == "" {
			return port.ResultFromError(apperrors.Invalid("agent.review.publish", "successful publication must return a content id"))
		}
		if err := publicationRepo.FinalizePublication(c.Context(), reviewID, actionInput, port.AIReviewPublication{
			Status:         publication.Status,
			ContentID:      publication.ContentID,
			PublicationKey: request.IdempotencyKey,
			PublishedAt:    actionInput.CreatedAt,
		}); err != nil {
			return port.ResultFromError(err)
		}
		return port.ResultOk()
	}
	if err := s.reviews.ApplyAction(c.Context(), reviewID, actionInput); err != nil {
		return port.ResultFromError(err)
	}
	if action == port.ReviewActionRejected && review.Operation == port.AgentDreamOperation && s.dreams != nil {
		if err := s.dreams.SyncReviewStatus(c.Context(), reviewID, port.DreamRejected, actionInput.CreatedAt); err != nil && !apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultFromError(err)
		}
	}
	return port.ResultOk()
}

func reviewIDParam(c port.Request, operation string) (string, error) {
	reviewID := strings.TrimSpace(c.Param("id"))
	if reviewID == "" || len(reviewID) > 64 {
		return "", apperrors.Invalid(operation, "review id is invalid")
	}
	return reviewID, nil
}

func isAgentReview(review port.AIReview) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(review.Operation)), "agent.") && strings.TrimSpace(review.AgentID) != ""
}

func publicationContent(review port.AIReview, action port.AIReviewAction) string {
	if action.Action == port.ReviewActionPartiallyAccepted && strings.TrimSpace(action.Content) != "" {
		return action.Content
	}
	return review.Content
}

func reviewActorID(c port.Request) (string, error) {
	value, ok := c.Get("userInfo")
	if !ok {
		return "", apperrors.New(apperrors.KindUnauthorized, "agent.review.actor", nil)
	}
	dto, ok := value.(port.UserDetailsDTO)
	if !ok || dto.UserInfoId <= 0 {
		return "", apperrors.New(apperrors.KindUnauthorized, "agent.review.actor", nil)
	}
	return strconv.Itoa(dto.UserInfoId), nil
}

func reviewSessionID(c port.Request, actorID string) (string, error) {
	value := ""
	if c != nil {
		value = strings.TrimSpace(c.GetHeader("X-Agent-Session-ID"))
		if value == "" {
			value = strings.TrimSpace(c.GetHeader("X-Session-ID"))
		}
	}
	if value == "" {
		value = "admin:" + strings.TrimSpace(actorID)
	}
	if len(value) > port.MaxAgentSessionIDLength || strings.ContainsAny(value, "\r\n\x00") {
		return "", apperrors.Invalid("agent.review.session", "review session id is invalid")
	}
	return value, nil
}

func normalizeReviewTarget(targetType, targetID string, articleID int) (string, string, error) {
	targetType = strings.ToLower(strings.TrimSpace(targetType))
	targetID = strings.TrimSpace(targetID)
	if targetType == "" {
		if articleID > 0 {
			targetType = aiReviewTargetArticle
		} else {
			targetType = aiReviewTargetDraft
		}
	}
	switch targetType {
	case aiReviewTargetArticle:
		if articleID > 0 {
			if targetID != "" && targetID != strconv.Itoa(articleID) {
				return "", "", apperrors.Invalid("agent.review.target", "article target does not match article id")
			}
			targetID = strconv.Itoa(articleID)
		}
		if targetID == "" {
			return "", "", apperrors.Invalid("agent.review.target", "article target id is required")
		}
	case aiReviewTargetDraft:
		if targetID == "" {
			targetID = "draft"
		}
	default:
		return "", "", apperrors.Invalid("agent.review.target", "review target type is invalid")
	}
	if len(targetID) > maxReviewTargetIDLen || strings.ContainsAny(targetID, "\r\n\x00") {
		return "", "", apperrors.Invalid("agent.review.target", "review target id is invalid")
	}
	return targetType, targetID, nil
}

func normalizeReviewStatus(status port.ReviewStatus) (port.ReviewStatus, error) {
	switch status {
	case port.ReviewPending, port.ReviewApproved, port.ReviewPartiallyApproved, port.ReviewRejected, port.ReviewExpired:
		return status, nil
	default:
		return "", apperrors.Invalid("agent.review.status", "review status is invalid")
	}
}

func pageQuery(c port.Request) (int, int, error) {
	current := 1
	size := 20
	var err error
	if value := strings.TrimSpace(c.Query("current")); value != "" {
		current, err = strconv.Atoi(value)
		if err != nil || current < 1 {
			return 0, 0, apperrors.Invalid("agent.review.page", "current must be a positive integer")
		}
	}
	if value := strings.TrimSpace(c.Query("size")); value != "" {
		size, err = strconv.Atoi(value)
		if err != nil || size < 1 || size > 100 {
			return 0, 0, apperrors.Invalid("agent.review.page", "size must be between 1 and 100")
		}
	}
	return current, size, nil
}
