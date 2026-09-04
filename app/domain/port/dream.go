package port

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	// AIJobKindDreamCandidate is a review-only generation task. Its worker
	// never writes public articles, comments, or talks.
	AIJobKindDreamCandidate = "agent.dream.generate"
	AgentDreamOperation     = "agent.dream"
	DreamTargetType         = "dream"
	MaxDreamSeedArticles    = 5
)

// DreamTaskPayload contains only article identifiers and versioned profile
// metadata. The worker reads the current public article snapshot again, so a
// retry cannot send stale or private content to a model.
type DreamTaskPayload struct {
	SchemaVersion  int       `json:"schemaVersion"`
	SeedArticleIDs []int     `json:"seedArticleIds"`
	ProfileID      string    `json:"profileId"`
	PromptVersion  string    `json:"promptVersion"`
	IdempotencyKey string    `json:"idempotencyKey"`
	OccurredAt     time.Time `json:"occurredAt"`
}

func (p DreamTaskPayload) Validate() error {
	if p.SchemaVersion <= 0 {
		return errors.New("dream task schema version must be positive")
	}
	if len(p.SeedArticleIDs) == 0 || len(p.SeedArticleIDs) > MaxDreamSeedArticles {
		return errors.New("dream task seed article count is invalid")
	}
	seen := make(map[int]struct{}, len(p.SeedArticleIDs))
	for _, articleID := range p.SeedArticleIDs {
		if articleID <= 0 {
			return errors.New("dream task article id must be positive")
		}
		if _, exists := seen[articleID]; exists {
			return errors.New("dream task seed articles must be unique")
		}
		seen[articleID] = struct{}{}
	}
	if strings.TrimSpace(p.ProfileID) == "" || len(p.ProfileID) > 128 || strings.ContainsAny(p.ProfileID, "\r\n\x00") {
		return errors.New("dream task profile id is invalid")
	}
	if strings.TrimSpace(p.PromptVersion) == "" || len(p.PromptVersion) > 128 || strings.ContainsAny(p.PromptVersion, "\r\n\x00") {
		return errors.New("dream task prompt version is invalid")
	}
	if strings.TrimSpace(p.IdempotencyKey) == "" || len(p.IdempotencyKey) > 255 || strings.ContainsAny(p.IdempotencyKey, "\r\n\x00") {
		return errors.New("dream task idempotency key is invalid")
	}
	if p.OccurredAt.IsZero() {
		return errors.New("dream task occurrence time is required")
	}
	return nil
}

func NewDreamTaskPayload(seedArticleIDs []int, profileID, promptVersion, idempotencyKey string, occurredAt time.Time) ([]byte, error) {
	payload := DreamTaskPayload{
		SchemaVersion:  1,
		SeedArticleIDs: append([]int(nil), seedArticleIDs...),
		ProfileID:      strings.TrimSpace(profileID),
		PromptVersion:  strings.TrimSpace(promptVersion),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
		OccurredAt:     occurredAt.UTC(),
	}
	if err := payload.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

type DreamCandidateResult struct {
	Created  bool   `json:"created"`
	ReviewID string `json:"reviewId,omitempty"`
	RunID    string `json:"runId,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// DreamCandidateGenerator creates a candidate that remains pending until a
// human approves it in AI Studio.
type DreamCandidateGenerator interface {
	Generate(context.Context, DreamTaskPayload) (DreamCandidateResult, error)
}

type DreamStatus string

const (
	DreamPendingReview DreamStatus = "pending_review"
	DreamApproved      DreamStatus = "approved"
	DreamRejected      DreamStatus = "rejected"
	DreamExpired       DreamStatus = "expired"
)

type DreamImageStatus string

const (
	DreamImagePending     DreamImageStatus = "pending"
	DreamImageProcessing  DreamImageStatus = "processing"
	DreamImageReady       DreamImageStatus = "ready"
	DreamImagePlaceholder DreamImageStatus = "placeholder"
	DreamImageFailed      DreamImageStatus = "failed"
)

// DreamEntry is the public projection of an approved dream plus the private
// generation metadata needed by the asynchronous image worker.
type DreamEntry struct {
	ID               string
	ReviewID         string
	Title            string
	Content          string
	ImagePrompt      string
	SourceArticleIDs []int
	Status           DreamStatus
	ImageStatus      DreamImageStatus
	ImageURL         string
	ImageError       string
	ImageLeaseOwner  string
	ImageLeaseUntil  time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type DreamPage struct {
	Records []DreamEntry
	Count   int
}

type DreamRepository interface {
	Create(context.Context, DreamEntry) error
	GetByReview(context.Context, string) (DreamEntry, error)
	Approve(context.Context, string, string, time.Time) error
	SyncReviewStatus(context.Context, string, DreamStatus, time.Time) error
	ListPublic(context.Context, int, int) ([]DreamEntry, int, error)
	ListPendingImages(context.Context, int, time.Time) ([]DreamEntry, error)
	ClaimImage(context.Context, string, string, time.Time, time.Duration) (DreamEntry, bool, error)
	CompleteImage(context.Context, string, string, DreamImageStatus, string, string, time.Time) error
}

type ImageRequest struct {
	Prompt   string
	Metadata map[string]string
}

type ImageResponse struct {
	URL      string
	Provider string
}

// ImageGateway is deliberately provider-neutral. The first rollout may leave
// it nil and use the local placeholder fallback; adding a provider later does
// not change the dream review or public read contracts.
type ImageGateway interface {
	Generate(context.Context, ImageRequest) (ImageResponse, error)
}
