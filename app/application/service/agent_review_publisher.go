package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

// AgentReviewPublisherDeps wires autonomous publication to the existing
// comment and talk application services. The article reader performs a
// second visibility check immediately before any public write.
type AgentReviewPublisherDeps struct {
	Articles port.AgentArticleReader
	Comments *MyCommentService
	Talks    *MyTalkService
	Activity port.AgentActivityRepository
}

type AgentReviewPublisher struct {
	articles port.AgentArticleReader
	comments *MyCommentService
	talks    *MyTalkService
	activity port.AgentActivityRepository
}

var _ port.AgentReviewPublisher = (*AgentReviewPublisher)(nil)

func NewAgentReviewPublisher(deps AgentReviewPublisherDeps) (*AgentReviewPublisher, error) {
	if deps.Articles == nil || deps.Comments == nil || deps.Talks == nil {
		return nil, apperrors.Invalid("service.agent_review_publisher.dependencies", "articles, comments and talks are required")
	}
	return &AgentReviewPublisher{articles: deps.Articles, comments: deps.Comments, talks: deps.Talks, activity: deps.Activity}, nil
}

func (p *AgentReviewPublisher) Publish(ctx context.Context, input port.AgentReviewPublication) (port.AgentReviewPublicationResult, error) {
	if p == nil || p.articles == nil || p.comments == nil || p.talks == nil {
		return port.AgentReviewPublicationResult{}, apperrors.Unavailable("agent.review.publish", nil)
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.review.publish", "approved content is required")
	}
	actorID, err := strconv.Atoi(strings.TrimSpace(input.Review.AgentID))
	if err != nil || actorID <= 0 {
		return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.review.publish", "virtual actor must reference a positive user id")
	}

	switch input.Review.Operation {
	case port.AgentBehaviorOperationWake:
		// A wake suggestion is intentionally review-only. Approval records the
		// decision, but does not create public content.
		return port.AgentReviewPublicationResult{Status: port.ReviewPublishSkipped}, nil
	case port.AgentBehaviorOperationComment, port.AgentBehaviorOperationTalk:
		if input.Review.SourceArticleID <= 0 {
			return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.review.publish", "source article is required")
		}
		article, _, _, err := p.articles.GetAdminArticle(ctx, input.Review.SourceArticleID)
		if err != nil {
			return port.AgentReviewPublicationResult{}, err
		}
		if !port.IsPublicArticle(article.Status, article.IsDelete) {
			return port.AgentReviewPublicationResult{}, apperrors.Conflict("agent.review.publish", "source article is no longer public")
		}
		if input.Review.Operation == port.AgentBehaviorOperationComment {
			id, err := p.comments.PublishAgentComment(ctx, actorID, input.Review.SourceArticleID, content)
			if err != nil {
				return port.AgentReviewPublicationResult{}, err
			}
			p.recordActivity(ctx, input, "comment", strconv.Itoa(id), AgentActivityEmotionGentle)
			return port.AgentReviewPublicationResult{Status: port.ReviewPublishSucceeded, ContentID: strconv.Itoa(id)}, nil
		}
		id, err := p.talks.PublishAgentTalk(ctx, actorID, content)
		if err != nil {
			return port.AgentReviewPublicationResult{}, err
		}
		p.recordActivity(ctx, input, "talk", strconv.Itoa(id), AgentActivityEmotionMelancholic)
		return port.AgentReviewPublicationResult{Status: port.ReviewPublishSucceeded, ContentID: strconv.Itoa(id)}, nil
	default:
		return port.AgentReviewPublicationResult{}, apperrors.Invalid("agent.review.publish", "review operation is not publishable")
	}
}

const (
	AgentActivityEmotionGentle      = "gentle"
	AgentActivityEmotionMelancholic = "melancholic"
)

func (p *AgentReviewPublisher) recordActivity(ctx context.Context, input port.AgentReviewPublication, contentType, contentID, emotion string) {
	if p == nil || p.activity == nil {
		return
	}
	activityIDDigest := sha256.Sum256([]byte(strings.TrimSpace(input.Review.ID) + ":" + contentType))
	if err := p.activity.Append(ctx, port.AgentActivity{
		ID:          hex.EncodeToString(activityIDDigest[:]),
		Kind:        "agent.review.published",
		Emotion:     emotion,
		ContentType: contentType,
		ContentID:   contentID,
		LifeStage:   port.LifeStageGrowing,
		Metadata: map[string]string{
			"review_id": input.Review.ID,
			"run_id":    input.Review.RunID,
		},
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		slog.Warn("agent activity record failed after publication", "review_id", input.Review.ID, "error_code", "activity_record_failed")
	}
}
