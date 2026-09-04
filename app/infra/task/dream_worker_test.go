package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type dreamWorkerSchedulerFake struct{}

func (dreamWorkerSchedulerFake) Scan(context.Context, int) (int, int, error) { return 0, 0, nil }

type dreamWorkerGeneratorFake struct {
	payloads []port.DreamTaskPayload
	outcome  port.DreamCandidateResult
	err      error
}

func (f *dreamWorkerGeneratorFake) Generate(_ context.Context, payload port.DreamTaskPayload) (port.DreamCandidateResult, error) {
	f.payloads = append(f.payloads, payload)
	return f.outcome, f.err
}

func dreamWorkerPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := port.NewDreamTaskPayload([]int{42}, "benetnasch-public", "v1", "dream-key", time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestDreamWorkerCompletesReviewOnlyJob(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID: "dream-job-1", Kind: port.AIJobKindDreamCandidate, Payload: dreamWorkerPayload(t), Attempts: 1, MaxAttempts: 3,
	}}
	generator := &dreamWorkerGeneratorFake{outcome: port.DreamCandidateResult{Created: true, ReviewID: "review-1", RunID: "run-1"}}
	worker, err := NewDreamWorker(DreamWorkerDeps{Jobs: jobs, Scheduler: dreamWorkerSchedulerFake{}, Generator: generator, WorkerID: "dream-test-worker"})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed || len(generator.payloads) != 1 || len(jobs.completed) != 1 {
		t.Fatalf("processed=%v err=%v payloads=%+v completed=%+v", processed, err, generator.payloads, jobs.completed)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(jobs.completed[0].Payload, &result); err != nil || result.Status != "review_created" {
		t.Fatalf("result=%s err=%v", jobs.completed[0].Payload, err)
	}
}

func TestDreamWorkerRetriesProviderFailure(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID: "dream-job-2", Kind: port.AIJobKindDreamCandidate, Payload: dreamWorkerPayload(t), Attempts: 1, MaxAttempts: 3,
	}}
	worker, err := NewDreamWorker(DreamWorkerDeps{
		Jobs: jobs, Scheduler: dreamWorkerSchedulerFake{}, Generator: &dreamWorkerGeneratorFake{err: errors.New("provider unavailable")}, WorkerID: "dream-test-worker", RetryBase: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed || len(jobs.retried) != 1 || len(jobs.completed) != 0 {
		t.Fatalf("processed=%v err=%v retried=%+v completed=%+v", processed, err, jobs.retried, jobs.completed)
	}
}

type dreamImageRepositoryFake struct {
	entry     port.DreamEntry
	listed    bool
	completed bool
	status    port.DreamImageStatus
	url       string
	err       string
}

func (f *dreamImageRepositoryFake) Create(context.Context, port.DreamEntry) error { return nil }
func (f *dreamImageRepositoryFake) GetByReview(context.Context, string) (port.DreamEntry, error) {
	return f.entry, nil
}
func (f *dreamImageRepositoryFake) Approve(context.Context, string, string, time.Time) error {
	return nil
}
func (f *dreamImageRepositoryFake) SyncReviewStatus(context.Context, string, port.DreamStatus, time.Time) error {
	return nil
}
func (f *dreamImageRepositoryFake) ListPublic(context.Context, int, int) ([]port.DreamEntry, int, error) {
	return nil, 0, nil
}
func (f *dreamImageRepositoryFake) ListPendingImages(context.Context, int, time.Time) ([]port.DreamEntry, error) {
	if f.listed {
		return nil, nil
	}
	return []port.DreamEntry{f.entry}, nil
}
func (f *dreamImageRepositoryFake) ClaimImage(context.Context, string, string, time.Time, time.Duration) (port.DreamEntry, bool, error) {
	f.listed = true
	return f.entry, true, nil
}
func (f *dreamImageRepositoryFake) CompleteImage(_ context.Context, _ string, _ string, status port.DreamImageStatus, imageURL, imageError string, _ time.Time) error {
	f.completed = true
	f.status = status
	f.url = imageURL
	f.err = imageError
	return nil
}

func TestDreamImageWorkerFallsBackToLocalPlaceholder(t *testing.T) {
	repository := &dreamImageRepositoryFake{entry: port.DreamEntry{ID: "dream-1", ReviewID: "review-1", Status: port.DreamApproved, ImageStatus: port.DreamImagePending, ImagePrompt: "night"}}
	worker, err := NewDreamImageWorker(DreamImageWorkerDeps{Dreams: repository, PlaceholderURL: "/dream-placeholder.svg", WorkerID: "image-test-worker"})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed || !repository.completed || repository.status != port.DreamImagePlaceholder || repository.url != "/dream-placeholder.svg" || repository.err == "" {
		t.Fatalf("processed=%v err=%v repository=%+v", processed, err, repository)
	}
}
