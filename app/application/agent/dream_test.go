package agent

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type dreamArticleReaderFake struct {
	article entity.TArticle
	err     error
}

func (f dreamArticleReaderFake) GetAdminArticle(_ context.Context, id int) (entity.TArticle, string, []string, error) {
	if f.err != nil {
		return entity.TArticle{}, "", nil, f.err
	}
	if f.article.Id != id {
		return entity.TArticle{}, "", nil, apperrors.NotFound("test.dream.article")
	}
	return f.article, "工程", []string{"Go", "Agent"}, nil
}

type dreamProfileRepositoryFake struct {
	profile port.AgentProfile
}

func (f dreamProfileRepositoryFake) Get(context.Context, string) (port.AgentProfile, error) {
	return f.profile, nil
}

func (f dreamProfileRepositoryFake) Save(context.Context, port.AgentProfile) error { return nil }

type dreamChatFake struct {
	response port.ChatResponse
	calls    int
}

func (f *dreamChatFake) Generate(_ context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	f.calls++
	if request.UseCase != port.AIUseCaseDream || len(request.Tools) != 0 || request.StructuredOutput == nil {
		return port.ChatResponse{}, errors.New("unsafe dream request")
	}
	return f.response, nil
}

func (f *dreamChatFake) Stream(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error {
	return errors.New("stream is not used in dream tests")
}

type dreamReviewsFake struct {
	reviews map[string]port.AIReview
	creates int
}

func (f *dreamReviewsFake) Create(_ context.Context, review port.AIReview) error {
	if f.reviews == nil {
		f.reviews = make(map[string]port.AIReview)
	}
	if _, exists := f.reviews[review.ID]; exists {
		return apperrors.Conflict("test.dream.review", "duplicate")
	}
	f.reviews[review.ID] = review
	f.creates++
	return nil
}

func (f *dreamReviewsFake) Get(_ context.Context, id string) (port.AIReview, error) {
	review, ok := f.reviews[id]
	if !ok {
		return port.AIReview{}, apperrors.NotFound("test.dream.review")
	}
	return review, nil
}

func (f *dreamReviewsFake) List(context.Context, port.ReviewFilter) ([]port.AIReview, int, error) {
	result := make([]port.AIReview, 0, len(f.reviews))
	for _, review := range f.reviews {
		result = append(result, review)
	}
	return result, len(result), nil
}

func (f *dreamReviewsFake) ApplyAction(context.Context, string, port.AIReviewAction) error {
	return nil
}
func (f *dreamReviewsFake) Approve(context.Context, string, string) error        { return nil }
func (f *dreamReviewsFake) Reject(context.Context, string, string, string) error { return nil }
func (f *dreamReviewsFake) Expire(context.Context, time.Time) (int, error)       { return 0, nil }

type dreamRepositoryFake struct {
	entries map[string]port.DreamEntry
}

func (f *dreamRepositoryFake) Create(_ context.Context, entry port.DreamEntry) error {
	if f.entries == nil {
		f.entries = make(map[string]port.DreamEntry)
	}
	if current, ok := f.entries[entry.ReviewID]; ok {
		if current.ID == entry.ID && current.Content == entry.Content {
			return nil
		}
		return apperrors.Conflict("test.dream.create", "duplicate")
	}
	f.entries[entry.ReviewID] = entry
	return nil
}

func (f *dreamRepositoryFake) GetByReview(_ context.Context, reviewID string) (port.DreamEntry, error) {
	entry, ok := f.entries[reviewID]
	if !ok {
		return port.DreamEntry{}, apperrors.NotFound("test.dream.get")
	}
	return entry, nil
}

func (f *dreamRepositoryFake) Approve(_ context.Context, reviewID, content string, now time.Time) error {
	entry, err := f.GetByReview(context.Background(), reviewID)
	if err != nil {
		return err
	}
	entry.Status = port.DreamApproved
	entry.Content = content
	entry.UpdatedAt = now
	f.entries[reviewID] = entry
	return nil
}

func (f *dreamRepositoryFake) SyncReviewStatus(context.Context, string, port.DreamStatus, time.Time) error {
	return nil
}

func (f *dreamRepositoryFake) ListPublic(context.Context, int, int) ([]port.DreamEntry, int, error) {
	return nil, 0, nil
}
func (f *dreamRepositoryFake) ListPendingImages(context.Context, int, time.Time) ([]port.DreamEntry, error) {
	return nil, nil
}
func (f *dreamRepositoryFake) ClaimImage(context.Context, string, string, time.Time, time.Duration) (port.DreamEntry, bool, error) {
	return port.DreamEntry{}, false, nil
}
func (f *dreamRepositoryFake) CompleteImage(context.Context, string, string, port.DreamImageStatus, string, string, time.Time) error {
	return nil
}

func TestDreamCandidateServiceCreatesPendingReviewAndIsIdempotent(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	chat := &dreamChatFake{response: port.ChatResponse{
		RunID:          "dream-run-1",
		StructuredJSON: json.RawMessage(`{"title":"夜航","content":"我在文章的星光之间缓慢醒来。","imagePrompt":"quiet blue night sky"}`),
	}}
	reviews := &dreamReviewsFake{reviews: make(map[string]port.AIReview)}
	dreams := &dreamRepositoryFake{entries: make(map[string]port.DreamEntry)}
	service, err := NewDreamCandidateService(DreamCandidateServiceDeps{
		Articles: dreamArticleReaderFake{article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0, ArticleTitle: "缓存", ArticleContent: "公开文章内容"}},
		Profiles: dreamProfileRepositoryFake{profile: port.AgentProfile{ID: "benetnasch-public", Name: "Benetnasch", PromptVersion: "v1", SystemPrompt: "保持克制", Enabled: true}},
		Reviews:  reviews,
		Dreams:   dreams,
		Chat:     chat,
		ActorID:  "42", ProfileID: "benetnasch-public", PromptVersion: "v1",
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	payloadBytes, err := port.NewDreamTaskPayload([]int{7}, "benetnasch-public", "v1", "agent-dream:test", now)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := DecodeDreamTask(payloadBytes)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Generate(context.Background(), payload)
	if err != nil || !first.Created || first.ReviewID == "" || first.RunID != "dream-run-1" {
		t.Fatalf("first result=%+v err=%v", first, err)
	}
	review := reviews.reviews[first.ReviewID]
	if review.Status != port.ReviewPending || review.TargetType != port.DreamTargetType || review.Operation != port.AgentDreamOperation || review.AgentID != "42" {
		t.Fatalf("review=%+v", review)
	}
	entry := dreams.entries[review.ID]
	if entry.Status != port.DreamPendingReview || entry.Title != "夜航" || entry.ImageStatus != port.DreamImagePending {
		t.Fatalf("dream entry=%+v", entry)
	}
	second, err := service.Generate(context.Background(), payload)
	if err != nil || second.Reason != "idempotent_existing" || chat.calls != 1 || reviews.creates != 1 {
		t.Fatalf("second=%+v err=%v chatCalls=%d creates=%d", second, err, chat.calls, reviews.creates)
	}
}

func TestDreamCandidateServiceDoesNotSendPrivateArticleToModel(t *testing.T) {
	chat := &dreamChatFake{response: port.ChatResponse{StructuredJSON: json.RawMessage(`{"title":"x","content":"content","imagePrompt":""}`)}}
	service, err := NewDreamCandidateService(DreamCandidateServiceDeps{
		Articles: dreamArticleReaderFake{article: entity.TArticle{Id: 7, Status: 2}},
		Profiles: dreamProfileRepositoryFake{profile: port.AgentProfile{ID: "profile", PromptVersion: "v1", Enabled: true}},
		Reviews:  &dreamReviewsFake{reviews: make(map[string]port.AIReview)},
		Dreams:   &dreamRepositoryFake{entries: make(map[string]port.DreamEntry)},
		Chat:     chat, ActorID: "42", ProfileID: "profile", PromptVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Generate(context.Background(), port.DreamTaskPayload{SchemaVersion: 1, SeedArticleIDs: []int{7}, ProfileID: "profile", PromptVersion: "v1", IdempotencyKey: "key", OccurredAt: time.Now()})
	if err != nil || result.Reason != "source_article_not_public" || chat.calls != 0 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, chat.calls)
	}
}

func TestDreamSchedulerUsesDailyDeterministicIdempotency(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	jobs := &behaviorJobsFake{}
	scheduler, err := NewDreamScheduler(DreamSchedulerDeps{
		Sources: behaviorSourceFake{sources: []port.ArticleIndexSource{{Article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0}}}},
		Jobs:    jobs, ProfileID: "benetnasch-public", PromptVersion: "v1", BatchSize: 10,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	next, count, err := scheduler.Scan(context.Background(), 0)
	if err != nil || next != 7 || count != 1 || len(jobs.jobs) != 1 {
		t.Fatalf("scan next=%d count=%d jobs=%+v err=%v", next, count, jobs.jobs, err)
	}
	payload, err := DecodeDreamTask(jobs.jobs[0].Payload)
	if err != nil || payload.IdempotencyKey != DreamTaskIdempotencyKey(now, 7, "benetnasch-public", "v1") {
		t.Fatalf("payload=%+v err=%v", payload, err)
	}
}
