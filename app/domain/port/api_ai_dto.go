package port

import (
	"time"
)

// AIWritingPreviewDTO is the stable JSON envelope returned by AI Studio. A
// preview has a review ID so every later decision can be audited against the
// exact generated text and local RunID.
type AIWritingPreviewDTO struct {
	ReviewID  string `json:"reviewId"`
	RunID     string `json:"runId"`
	Operation string `json:"operation"`
	Preview   string `json:"preview"`
	Diff      string `json:"diff"`
}

// AIVisionPreviewDTO is the safe result of an administrator vision request.
// The image input is deliberately absent from the response; the generated
// text is represented by a pending review before any later operator action.
type AIVisionPreviewDTO struct {
	ReviewID  string `json:"reviewId"`
	RunID     string `json:"runId"`
	Operation string `json:"operation"`
	Preview   string `json:"preview"`
}

type AIReviewDTO struct {
	ID                 string    `json:"id"`
	TargetType         string    `json:"targetType"`
	TargetID           string    `json:"targetId"`
	Operation          string    `json:"operation"`
	Content            string    `json:"content"`
	Diff               string    `json:"diff"`
	Status             string    `json:"status"`
	RunID              string    `json:"runId"`
	ReviewerID         string    `json:"reviewerId"`
	AgentID            string    `json:"agentId,omitempty"`
	PromptVersion      string    `json:"promptVersion,omitempty"`
	SourceArticleID    int       `json:"sourceArticleId,omitempty"`
	PublishStatus      string    `json:"publishStatus"`
	PublishError       string    `json:"publishError,omitempty"`
	PublishedContentID string    `json:"publishedContentId,omitempty"`
	PublishedAt        time.Time `json:"publishedAt,omitempty"`
	RejectReason       string    `json:"rejectReason"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	ExpiresAt          time.Time `json:"expiresAt"`
}

// AgentProfileDTO is the admin-facing, non-secret persona configuration. It
// deliberately contains no provider credentials or internal prompt refs.
type AgentProfileDTO struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	PromptVersion string            `json:"promptVersion"`
	SystemPrompt  string            `json:"systemPrompt"`
	Opening       string            `json:"opening"`
	RhythmPrompts map[string]string `json:"rhythmPrompts"`
	Enabled       bool              `json:"enabled"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// AgentProfileUpdateVO uses pointer fields so PATCH can update one persona
// field without accidentally clearing the other safety-sensitive text.
type AgentProfileUpdateVO struct {
	Name          *string `json:"name"`
	PromptVersion *string `json:"promptVersion"`
	SystemPrompt  *string `json:"systemPrompt"`
	Opening       *string `json:"opening"`
	AwakePrompt   *string `json:"awakePrompt"`
	DuskPrompt    *string `json:"duskPrompt"`
	NightPrompt   *string `json:"nightPrompt"`
	Enabled       *bool   `json:"enabled"`
}

// AgentReviewPolicyDTO exposes review gates without exposing provider
// credentials. ReviewRequired is read-only at the API boundary and is always
// true; administrators can tune limits but cannot disable human review.
type AgentReviewPolicyDTO struct {
	ID                  string    `json:"id"`
	Version             int64     `json:"version"`
	ReviewRequired      bool      `json:"reviewRequired"`
	ReviewTTLSeconds    int64     `json:"reviewTtlSeconds"`
	MaxCandidateRunes   int       `json:"maxCandidateRunes"`
	SimilarityThreshold float64   `json:"similarityThreshold"`
	DailyLimit          int       `json:"dailyLimit"`
	PerArticleLimit     int       `json:"perArticleLimit"`
	PerActionLimit      int       `json:"perActionLimit"`
	AllowedActions      []string  `json:"allowedActions"`
	SensitivePatterns   []string  `json:"sensitivePatterns"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// AgentReviewPolicyUpdateVO is a version-aware partial update. Version is
// optional for compatibility, but clients that send it get optimistic
// concurrency protection and cannot overwrite a newer policy silently.
type AgentReviewPolicyUpdateVO struct {
	Version             *int64    `json:"version"`
	ReviewTTLSeconds    *int64    `json:"reviewTtlSeconds"`
	MaxCandidateRunes   *int      `json:"maxCandidateRunes"`
	SimilarityThreshold *float64  `json:"similarityThreshold"`
	DailyLimit          *int      `json:"dailyLimit"`
	PerArticleLimit     *int      `json:"perArticleLimit"`
	PerActionLimit      *int      `json:"perActionLimit"`
	AllowedActions      *[]string `json:"allowedActions"`
	SensitivePatterns   *[]string `json:"sensitivePatterns"`
}

// ContentGalaxyPointDTO intentionally contains only public display data. It
// must never expose the source embedding, model version, or PCA input.
type ContentGalaxyPointDTO struct {
	ArticleID int       `json:"articleId"`
	LifeStage string    `json:"lifeStage"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ContentGalaxyPageDTO struct {
	Records   []ContentGalaxyPointDTO `json:"records"`
	Count     int                     `json:"count"`
	HasMore   bool                    `json:"hasMore"`
	NextSince string                  `json:"nextSince,omitempty"`
}

type DreamDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	ImageURL    string    `json:"imageUrl,omitempty"`
	ImageStatus string    `json:"imageStatus"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type DreamPageDTO struct {
	Records []DreamDTO `json:"records"`
	Count   int        `json:"count"`
	HasMore bool       `json:"hasMore"`
}

// TimeCapsuleDTO never contains the owner account ID or any profile fields.
// Content is populated only for the owner's draft and after delivery.
type TimeCapsuleDTO struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content,omitempty"`
	DeliverAt        time.Time `json:"deliverAt"`
	Status           string    `json:"status"`
	SealedAt         time.Time `json:"sealedAt,omitempty"`
	DeliveredAt      time.Time `json:"deliveredAt,omitempty"`
	DeliveryAttempts int       `json:"deliveryAttempts"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// RadioEpisodeDTO is a public, text-only episode. It intentionally contains
// no system prompt, profile internals, visitor identity, or model metadata.
type RadioEpisodeDTO struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Script          string    `json:"script"`
	Phase           string    `json:"phase"`
	TTS             bool      `json:"tts"`
	SourceArticleID int       `json:"sourceArticleId,omitempty"`
	PublishedAt     time.Time `json:"publishedAt"`
}

type RadioPageDTO struct {
	Records []RadioEpisodeDTO `json:"records"`
	Count   int               `json:"count"`
}

// VideoDTO is the safe public video projection. EmbedURL is populated only
// for an allowlisted external source; local objects use URL with a validated
// object-storage reference.
type VideoDTO struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Source      string    `json:"source"`
	URL         string    `json:"url"`
	EmbedURL    string    `json:"embedUrl,omitempty"`
	MIMEType    string    `json:"mimeType,omitempty"`
	SizeBytes   int64     `json:"sizeBytes,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type VideoPageDTO struct {
	Records      []VideoDTO `json:"records"`
	Count        int        `json:"count"`
	HasMore      bool       `json:"hasMore"`
	FrameOrigins []string   `json:"frameOrigins,omitempty"`
}

func NewAIReviewDTO(review AIReview) AIReviewDTO {
	return AIReviewDTO{
		ID:                 review.ID,
		TargetType:         review.TargetType,
		TargetID:           review.TargetID,
		Operation:          review.Operation,
		Content:            review.Content,
		Diff:               review.Diff,
		Status:             string(review.Status),
		RunID:              review.RunID,
		ReviewerID:         review.ReviewerID,
		AgentID:            review.AgentID,
		PromptVersion:      review.PromptVersion,
		SourceArticleID:    review.SourceArticleID,
		PublishStatus:      string(review.PublishStatus),
		PublishError:       review.PublishError,
		PublishedContentID: review.PublishedContentID,
		PublishedAt:        review.PublishedAt,
		RejectReason:       review.RejectReason,
		CreatedAt:          review.CreatedAt,
		UpdatedAt:          review.UpdatedAt,
		ExpiresAt:          review.ExpiresAt,
	}
}
