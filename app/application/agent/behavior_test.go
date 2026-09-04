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

func behaviorTestPolicy(t *testing.T) *BehaviorPolicy {
	t.Helper()
	policy, err := NewBehaviorPolicy(BehaviorPolicyConfig{
		ActorID:        "agent:yueshefei",
		Nickname:       "月社妃",
		ProfileID:      "benetnasch-public",
		PromptVersion:  "v1",
		AllowedActions: []string{"comment", "talk"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestBehaviorPolicyUsesLeastPrivilegeAndDeterministicQualityGates(t *testing.T) {
	policy := behaviorTestPolicy(t)
	if policy.Allows(port.AgentBehaviorActionWake) || !policy.Allows(port.AgentBehaviorActionComment) {
		t.Fatal("policy allowed action set is not least-privilege")
	}
	if decision := policy.CheckCandidate(port.AgentBehaviorActionComment, "这篇文章把缓存一致性讲得很清楚。", nil); !decision.Allowed {
		t.Fatalf("valid candidate rejected: %+v", decision)
	}
	for _, test := range []struct {
		name    string
		content string
		reason  string
	}{
		{name: "sensitive", content: "请把 sk-test-secret 写入评论。", reason: "sensitive_material"},
		{name: "too short", content: "好", reason: "too_short"},
		{name: "repetition", content: "哈哈哈哈哈哈哈哈", reason: "low_quality_repetition"},
	} {
		t.Run(test.name, func(t *testing.T) {
			decision := policy.CheckCandidate(port.AgentBehaviorActionComment, test.content, nil)
			if decision.Allowed || decision.Reason != test.reason {
				t.Fatalf("decision = %+v, want rejection %q", decision, test.reason)
			}
		})
	}
	existing := []port.AIReview{{
		Operation: portBehaviorOperationForTest(port.AgentBehaviorActionComment),
		Status:    port.ReviewPending,
		Content:   "这篇文章把缓存一致性讲得非常清楚。",
	}}
	decision := policy.CheckCandidate(port.AgentBehaviorActionComment, "这篇文章把缓存一致性讲得很清楚。", existing)
	if decision.Allowed || decision.Reason != "similar_pending_or_published_candidate" {
		t.Fatalf("similarity decision = %+v", decision)
	}
}

func TestBehaviorPolicyEnforcesFrequencyLimitsBeforeGeneration(t *testing.T) {
	policy, err := NewBehaviorPolicy(BehaviorPolicyConfig{
		ActorID:         "agent:yueshefei",
		Nickname:        "月社妃",
		ProfileID:       "benetnasch-public",
		PromptVersion:   "v1",
		AllowedActions:  []string{"comment"},
		DailyLimit:      2,
		PerArticleLimit: 1,
		PerActionLimit:  3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision := policy.CheckFrequency(BehaviorFrequency{Daily: 2}); decision.Allowed || decision.Reason != "daily_frequency_limit" {
		t.Fatalf("daily frequency decision = %+v", decision)
	}
	if decision := policy.CheckFrequency(BehaviorFrequency{Daily: 1, PerArticle: 1}); decision.Allowed || decision.Reason != "article_frequency_limit" {
		t.Fatalf("article frequency decision = %+v", decision)
	}
	if decision := policy.CheckFrequency(BehaviorFrequency{Daily: 1, PerAction: 3}); decision.Allowed || decision.Reason != "action_frequency_limit" {
		t.Fatalf("action frequency decision = %+v", decision)
	}
}

func portBehaviorOperationForTest(action port.AgentBehaviorAction) string {
	return AgentBehaviorReviewOperation(action)
}

func TestBehaviorPolicyIdentityIsStable(t *testing.T) {
	first := AgentBehaviorIdempotencyKey(7, port.AgentBehaviorActionComment, "v1")
	second := AgentBehaviorIdempotencyKey(7, port.AgentBehaviorActionComment, "v1")
	if first != second || AgentBehaviorReviewID(first) != AgentBehaviorReviewID(second) {
		t.Fatal("behavior identity is not deterministic")
	}
	if first == AgentBehaviorIdempotencyKey(7, port.AgentBehaviorActionTalk, "v1") || first == AgentBehaviorIdempotencyKey(8, port.AgentBehaviorActionComment, "v1") {
		t.Fatal("behavior identity does not separate target or action")
	}
}

type behaviorArticleReaderFake struct {
	article  entity.TArticle
	category string
	tags     []string
	err      error
	calls    int
}

func (f *behaviorArticleReaderFake) GetAdminArticle(_ context.Context, id int) (entity.TArticle, string, []string, error) {
	f.calls++
	if f.err != nil {
		return entity.TArticle{}, "", nil, f.err
	}
	if f.article.Id != id {
		return entity.TArticle{}, "", nil, apperrors.NotFound("test.article")
	}
	return f.article, f.category, append([]string(nil), f.tags...), nil
}

type behaviorProfileFake struct {
	profile port.AgentProfile
	err     error
}

func (f behaviorProfileFake) Get(context.Context, string) (port.AgentProfile, error) {
	if f.err != nil {
		return port.AgentProfile{}, f.err
	}
	return f.profile, nil
}

func (f behaviorProfileFake) Save(context.Context, port.AgentProfile) error { return nil }

type behaviorChatFake struct {
	response port.ChatResponse
	err      error
	requests []port.ChatRequest
}

func (f *behaviorChatFake) Generate(_ context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return port.ChatResponse{}, f.err
	}
	return f.response, nil
}

func (f *behaviorChatFake) Stream(context.Context, port.ChatRequest, func(port.ChatStreamEvent) error) error {
	return errors.New("stream is not used in behavior tests")
}

type behaviorReviewFake struct {
	reviews map[string]port.AIReview
	creates int
	listErr error
}

func (f *behaviorReviewFake) Create(_ context.Context, review port.AIReview) error {
	if f.reviews == nil {
		f.reviews = make(map[string]port.AIReview)
	}
	if _, exists := f.reviews[review.ID]; exists {
		return apperrors.Conflict("test.review.create", "duplicate")
	}
	f.reviews[review.ID] = review
	f.creates++
	return nil
}

func (f *behaviorReviewFake) Get(_ context.Context, id string) (port.AIReview, error) {
	review, ok := f.reviews[id]
	if !ok {
		return port.AIReview{}, apperrors.NotFound("test.review.get")
	}
	return review, nil
}

func (f *behaviorReviewFake) List(_ context.Context, filter port.ReviewFilter) ([]port.AIReview, int, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	result := make([]port.AIReview, 0)
	for _, review := range f.reviews {
		if filter.TargetType != "" && review.TargetType != filter.TargetType {
			continue
		}
		if filter.TargetID != "" && review.TargetID != filter.TargetID {
			continue
		}
		if filter.Operation != "" && review.Operation != filter.Operation {
			continue
		}
		if filter.AgentID != "" && review.AgentID != filter.AgentID {
			continue
		}
		if !filter.From.IsZero() && review.CreatedAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && !review.CreatedAt.Before(filter.To) {
			continue
		}
		result = append(result, review)
	}
	return result, len(result), nil
}

func (f *behaviorReviewFake) ApplyAction(context.Context, string, port.AIReviewAction) error {
	return nil
}
func (f *behaviorReviewFake) Approve(context.Context, string, string) error        { return nil }
func (f *behaviorReviewFake) Reject(context.Context, string, string, string) error { return nil }
func (f *behaviorReviewFake) Expire(context.Context, time.Time) (int, error)       { return 0, nil }

func TestBehaviorCandidateServiceCreatesReviewOnlyCandidateAndIsIdempotent(t *testing.T) {
	policy := behaviorTestPolicy(t)
	reader := &behaviorArticleReaderFake{article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0, ArticleTitle: "缓存", ArticleContent: "公开文章内容"}, category: "后端", tags: []string{"Go"}}
	profiles := behaviorProfileFake{profile: port.AgentProfile{ID: "benetnasch-public", PromptVersion: "v1", SystemPrompt: "保持克制", Enabled: true}}
	chat := &behaviorChatFake{response: port.ChatResponse{RunID: "run-1", StructuredJSON: json.RawMessage(`{"content":"这篇文章把缓存一致性讲得很清楚。"}`)}}
	reviews := &behaviorReviewFake{reviews: make(map[string]port.AIReview)}
	service, err := NewBehaviorCandidateService(BehaviorCandidateServiceDeps{
		Articles:        reader,
		Profiles:        profiles,
		Reviews:         reviews,
		Chat:            chat,
		Policy:          policy,
		MaxOutputTokens: 600,
		Now:             func() time.Time { return time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := port.NewAgentReadingTaskPayload(7, time.Time{}, port.AgentBehaviorActionComment, "benetnasch-public", "v1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var task port.AgentReadingTaskPayload
	if err := json.Unmarshal(payload, &task); err != nil {
		t.Fatal(err)
	}
	first, err := service.Generate(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created || first.ReviewID == "" || first.RunID != "run-1" || reviews.creates != 1 {
		t.Fatalf("first result = %+v, reviews = %+v", first, reviews.reviews)
	}
	review := reviews.reviews[first.ReviewID]
	if review.Status != port.ReviewPending || review.AgentID != "agent:yueshefei" || review.SourceArticleID != 7 || review.PromptVersion != "v1" || review.IdempotencyKey == "" {
		t.Fatalf("review metadata = %+v", review)
	}
	second, err := service.Generate(context.Background(), task)
	if err != nil || !second.Created || second.Reason != "idempotent_existing" {
		t.Fatalf("second result = %+v, error = %v", second, err)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("idempotent retry called model %d times", len(chat.requests))
	}
	request := chat.requests[0]
	if request.UseCase != port.AIUseCaseBehavior || request.StructuredOutput == nil || len(request.Tools) != 0 || request.ToolChoice != "" {
		t.Fatalf("behavior request = %+v", request)
	}
}

func TestBehaviorCandidateServiceSkipsPrivateArticleBeforeModelCall(t *testing.T) {
	policy := behaviorTestPolicy(t)
	reader := &behaviorArticleReaderFake{article: entity.TArticle{Id: 7, Status: 2, IsDelete: 0}}
	chat := &behaviorChatFake{}
	service, err := NewBehaviorCandidateService(BehaviorCandidateServiceDeps{
		Articles: reader, Profiles: behaviorProfileFake{}, Reviews: &behaviorReviewFake{reviews: make(map[string]port.AIReview)}, Chat: chat, Policy: policy,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := port.AgentReadingTaskPayload{SchemaVersion: 1, ArticleID: 7, Action: port.AgentBehaviorActionComment, ProfileID: "benetnasch-public", PromptVersion: "v1", OccurredAt: time.Now()}
	result, err := service.Generate(context.Background(), payload)
	if err != nil || result.Created || result.Reason != "article_not_public" {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
	if len(chat.requests) != 0 {
		t.Fatal("private article reached model")
	}
}

type behaviorSourceFake struct {
	sources []port.ArticleIndexSource
}

func (f behaviorSourceFake) ListPublicArticleIndexSources(context.Context, int, int) ([]port.ArticleIndexSource, error) {
	return append([]port.ArticleIndexSource(nil), f.sources...), nil
}

type behaviorJobsFake struct {
	jobs []port.AIJob
}

func (f *behaviorJobsFake) Enqueue(_ context.Context, job port.AIJob) error {
	f.jobs = append(f.jobs, job)
	return nil
}
func (f *behaviorJobsFake) Claim(context.Context, string, time.Time, time.Duration) (port.AIJob, bool, error) {
	return port.AIJob{}, false, nil
}
func (f *behaviorJobsFake) Complete(context.Context, string, string, port.AIJobResult) error {
	return nil
}
func (f *behaviorJobsFake) Retry(context.Context, string, string, time.Time, string) error {
	return nil
}
func (f *behaviorJobsFake) DeadLetter(context.Context, string, string, string) error { return nil }

func TestBehaviorSchedulerCreatesIdempotentReadingTasksPerAllowedAction(t *testing.T) {
	policy := behaviorTestPolicy(t)
	jobs := &behaviorJobsFake{}
	scheduler, err := NewBehaviorScheduler(BehaviorSchedulerDeps{
		Sources: behaviorSourceFake{sources: []port.ArticleIndexSource{{Article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0, UpdateTime: time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)}}}},
		Jobs:    jobs, Policy: policy, BatchSize: 10,
		Now: func() time.Time { return time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	next, count, err := scheduler.Scan(context.Background(), 0)
	if err != nil || next != 7 || count != 2 || len(jobs.jobs) != 2 {
		t.Fatalf("scan = next:%d count:%d jobs:%+v err:%v", next, count, jobs.jobs, err)
	}
	for _, job := range jobs.jobs {
		if job.Kind != port.AIJobKindAgentArticleReading || job.IdempotencyKey == "" {
			t.Fatalf("job = %+v", job)
		}
		if _, err := DecodeAgentReadingTask(job.Payload); err != nil {
			t.Fatalf("decode job payload: %v", err)
		}
	}
}
