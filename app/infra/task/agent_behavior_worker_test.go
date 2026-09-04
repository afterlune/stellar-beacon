package task

import (
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeBehaviorScheduler struct {
	next    int
	enqueue int
	err     error
}

func (f *fakeBehaviorScheduler) Scan(context.Context, int) (int, int, error) {
	return f.next, f.enqueue, f.err
}

type fakeBehaviorGenerator struct {
	payloads []port.AgentReadingTaskPayload
	outcome  port.AgentBehaviorCandidateResult
	err      error
}

func (f *fakeBehaviorGenerator) Generate(_ context.Context, payload port.AgentReadingTaskPayload) (port.AgentBehaviorCandidateResult, error) {
	f.payloads = append(f.payloads, payload)
	return f.outcome, f.err
}

func behaviorTaskPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := port.NewAgentReadingTaskPayload(
		42,
		time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC),
		port.AgentBehaviorActionComment,
		"benetnasch-public",
		"v1",
		time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestAgentBehaviorWorkerCompletesReviewOnlyJob(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID:          "behavior-job-1",
		Kind:        port.AIJobKindAgentArticleReading,
		Payload:     behaviorTaskPayload(t),
		Attempts:    1,
		MaxAttempts: 3,
	}}
	generator := &fakeBehaviorGenerator{outcome: port.AgentBehaviorCandidateResult{Created: true, ReviewID: "review-1", RunID: "run-1"}}
	worker, err := NewAgentBehaviorWorker(AgentBehaviorWorkerDeps{
		Jobs:      jobs,
		Scheduler: &fakeBehaviorScheduler{},
		Generator: generator,
		WorkerID:  "behavior-test-worker",
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("processOne() = processed:%v err:%v", processed, err)
	}
	if len(generator.payloads) != 1 || len(jobs.completed) != 1 || jobs.completed[0].RunID != "run-1" {
		t.Fatalf("generator payloads = %+v, completed = %+v", generator.payloads, jobs.completed)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(jobs.completed[0].Payload, &result); err != nil || result.Status != "review_created" {
		t.Fatalf("completed payload = %s, error = %v", jobs.completed[0].Payload, err)
	}
}

func TestAgentBehaviorWorkerRetriesProviderFailureAndDoesNotPublish(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID:          "behavior-job-2",
		Kind:        port.AIJobKindAgentArticleReading,
		Payload:     behaviorTaskPayload(t),
		Attempts:    1,
		MaxAttempts: 3,
	}}
	generator := &fakeBehaviorGenerator{err: errors.New("provider unavailable")}
	worker, err := NewAgentBehaviorWorker(AgentBehaviorWorkerDeps{
		Jobs: jobs, Scheduler: &fakeBehaviorScheduler{}, Generator: generator, WorkerID: "behavior-test-worker", RetryBase: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("processOne() = processed:%v err:%v", processed, err)
	}
	if len(jobs.retried) != 1 || len(jobs.completed) != 0 || len(jobs.deadLetters) != 0 {
		t.Fatalf("job settlement = retried:%+v completed:%+v dead:%+v", jobs.retried, jobs.completed, jobs.deadLetters)
	}
}

func TestAgentBehaviorWorkerConstructorRequiresSafeDependencies(t *testing.T) {
	if _, err := NewAgentBehaviorWorker(AgentBehaviorWorkerDeps{}); err == nil {
		t.Fatal("expected missing dependency error")
	}
}
