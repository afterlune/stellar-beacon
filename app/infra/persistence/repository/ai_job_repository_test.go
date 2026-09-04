package repository

import (
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestNormalizeAIJobFillsSafeDefaultsAndCopiesPayload(t *testing.T) {
	payload := []byte(`{"articleId":42}`)
	job, err := normalizeAIJob(port.AIJob{
		Kind:           port.AIJobKindArticleIndex,
		IdempotencyKey: "article:42:index",
		Payload:        payload,
	})
	if err != nil {
		t.Fatalf("normalizeAIJob() error = %v", err)
	}
	if job.ID == "" || job.Status != port.AIJobPending || job.MaxAttempts != port.DefaultAIJobMaxAttempts || job.RunAfter.IsZero() || job.CreatedAt.IsZero() || job.UpdatedAt.IsZero() {
		t.Fatalf("defaults were not applied: %+v", job)
	}
	payload[0] = 'X'
	if job.Payload[0] == 'X' {
		t.Fatal("normalizeAIJob() did not copy payload")
	}
}

func TestNormalizeAIJobRejectsInvalidEnqueueState(t *testing.T) {
	base := port.AIJob{Kind: "test", IdempotencyKey: "key", Payload: []byte("{}")}
	tests := []port.AIJob{
		{IdempotencyKey: base.IdempotencyKey, Payload: base.Payload},
		{Kind: base.Kind, Payload: base.Payload},
		{Kind: base.Kind, IdempotencyKey: base.IdempotencyKey},
		{Kind: base.Kind, IdempotencyKey: base.IdempotencyKey, Payload: base.Payload, Status: port.AIJobRunning},
		{Kind: base.Kind, IdempotencyKey: base.IdempotencyKey, Payload: base.Payload, Attempts: -1},
		{Kind: base.Kind, IdempotencyKey: base.IdempotencyKey, Payload: base.Payload, Attempts: 4, MaxAttempts: 3},
	}
	for index, job := range tests {
		if _, err := normalizeAIJob(job); err == nil {
			t.Fatalf("case %d: normalizeAIJob() error = nil", index)
		}
	}
}

func TestAIJobRowMapsToProviderNeutralJob(t *testing.T) {
	when := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)
	job := (aiJobRow{
		ID:             "job-1",
		Kind:           port.AIJobKindArticleIndex,
		IdempotencyKey: "key-1",
		Payload:        "{}",
		Status:         string(port.AIJobRunning),
		Attempts:       1,
		MaxAttempts:    3,
		RunAfter:       when,
		CreatedAt:      when,
		UpdatedAt:      when,
	}).aiJob()
	if job.ID != "job-1" || job.Status != port.AIJobRunning || string(job.Payload) != "{}" || job.Attempts != 1 || !job.RunAfter.Equal(when) {
		t.Fatalf("unexpected neutral job: %+v", job)
	}
}

func TestAIJobReturningColumnsQualifyUpdateTarget(t *testing.T) {
	for _, column := range []string{
		"job.id",
		"job.kind",
		"job.idempotency_key",
		"job.payload",
		"job.status",
		"job.attempts",
		"job.max_attempts",
		"job.run_after",
		"job.last_error",
		"job.created_at",
		"job.updated_at",
	} {
		if !strings.Contains(aiJobReturningColumns, column) {
			t.Fatalf("returning columns %q does not qualify %q", aiJobReturningColumns, column)
		}
	}
}
