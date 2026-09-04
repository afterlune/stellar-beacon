package port

import "context"

// AgentReviewPublication is the application boundary for turning an already
// approved candidate into public content. The publisher receives the review
// snapshot and the human-approved content; it has no method for generating
// candidates or bypassing the review lifecycle.
type AgentReviewPublication struct {
	Review         AIReview
	Content        string
	ReviewerID     string
	IdempotencyKey string
}

type AgentReviewPublicationResult struct {
	Status    ReviewPublishStatus
	ContentID string
}

type AgentReviewPublisher interface {
	Publish(context.Context, AgentReviewPublication) (AgentReviewPublicationResult, error)
}
