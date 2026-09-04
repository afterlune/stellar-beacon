package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/infra/task"
)

type fakeArticleIndexBackfillCommand struct {
	state port.ArticleIndexBackfillState
	err   error
}

func (f fakeArticleIndexBackfillCommand) Run(context.Context) (port.ArticleIndexBackfillState, error) {
	return f.state, f.err
}

func TestWriteArticleIndexBackfillStateUsesStableJSON(t *testing.T) {
	created := time.Date(2026, 8, 28, 12, 0, 0, 123456789, time.FixedZone("CST", 8*60*60))
	state := port.ArticleIndexBackfillState{
		ID:                 "backfill-1",
		IndexUID:           "article_chunks_v1",
		IndexVersion:       "v1",
		Provider:           "openai",
		Model:              "embedding-model",
		ModelVersion:       "2026-08",
		Dimension:          3,
		EmbeddingBatchSize: 2,
		PageSize:           10,
		Status:             port.ArticleIndexBackfillPaused,
		Cursor:             42,
		ProcessedArticles:  7,
		IndexedChunks:      19,
		PauseRequested:     false,
		LeaseOwner:         "worker-1",
		LeaseUntil:         created.Add(time.Minute),
		LastError:          "",
		StartedAt:          created,
		CreatedAt:          created,
		UpdatedAt:          created.Add(time.Minute),
	}

	var output bytes.Buffer
	if err := writeArticleIndexBackfillState(&output, state); err != nil {
		t.Fatalf("writeArticleIndexBackfillState() error = %v", err)
	}
	var got articleIndexBackfillStateOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode command output: %v", err)
	}
	if got.ID != state.ID || got.Status != state.Status || got.Cursor != state.Cursor || got.IndexedChunks != state.IndexedChunks {
		t.Fatalf("decoded state = %+v, want persisted fields", got)
	}
	if got.LeaseUntil != state.LeaseUntil.UTC().Format("2006-01-02T15:04:05.999999999Z07:00") {
		t.Fatalf("leaseUntil = %q, want UTC RFC3339Nano", got.LeaseUntil)
	}
	if got.CompletedAt != "" || strings.Contains(output.String(), "lastError") {
		t.Fatalf("zero optional fields should be omitted: %s", output.String())
	}
}

func TestExecuteArticleIndexBackfillWritesStateAndReturnsRunError(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	backfill := fakeArticleIndexBackfillCommand{
		state: port.ArticleIndexBackfillState{
			ID:     "backfill-1",
			Status: port.ArticleIndexBackfillFailed,
		},
		err: wantErr,
	}
	var output bytes.Buffer
	if err := executeArticleIndexBackfill(context.Background(), &output, backfill); !errors.Is(err, wantErr) {
		t.Fatalf("executeArticleIndexBackfill() error = %v, want wrapped run error", err)
	}
	if !strings.Contains(output.String(), `"status": "failed"`) {
		t.Fatalf("command output = %s, want failed state", output.String())
	}
}

func TestExecuteArticleIndexBackfillWithLimitRequiresBoundedRunner(t *testing.T) {
	backfill := fakeArticleIndexBackfillCommand{}
	var output bytes.Buffer
	if err := executeArticleIndexBackfillWithLimit(context.Background(), &output, backfill, 1); err == nil {
		t.Fatal("expected bounded runner requirement")
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q, want no status before bounded runner validation", output.String())
	}
}

func TestWriteArticleIndexBackfillStateRejectsNilOutput(t *testing.T) {
	if err := writeArticleIndexBackfillState(nil, port.ArticleIndexBackfillState{}); err == nil {
		t.Fatal("expected nil output error")
	}
}

func TestWriteArticleIndexPlanEmitsOnlyBoundedSummary(t *testing.T) {
	var output bytes.Buffer
	result := task.ArticleIndexPlanResult{
		Index:             "article_chunks_v1",
		PageSize:          25,
		AfterArticleID:    10,
		LastArticleID:     42,
		ScannedArticles:   3,
		EstimatedChunks:   8,
		InvalidArticles:   1,
		InvalidArticleIDs: []int{42},
		Complete:          true,
	}
	if err := writeArticleIndexPlan(&output, result); err != nil {
		t.Fatalf("writeArticleIndexPlan() error = %v", err)
	}
	var got task.ArticleIndexPlanResult
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("decode article index plan: %v", err)
	}
	if got.Index != result.Index || got.ScannedArticles != result.ScannedArticles || got.InvalidArticleIDs[0] != 42 {
		t.Fatalf("decoded plan = %+v, want %+v", got, result)
	}
	if strings.Contains(output.String(), "article body") {
		t.Fatal("article content must not appear in plan output")
	}
}

func TestWriteArticleIndexPlanRejectsNilOutput(t *testing.T) {
	if err := writeArticleIndexPlan(nil, task.ArticleIndexPlanResult{}); err == nil {
		t.Fatal("expected nil output error")
	}
}

func TestRequireArticleIndexBackfillWriteAuthorization(t *testing.T) {
	if err := requireArticleIndexBackfillWriteAuthorization(false); err == nil {
		t.Fatal("expected backfill write authorization error")
	}
	if err := requireArticleIndexBackfillWriteAuthorization(true); err != nil {
		t.Fatalf("authorized backfill write returned error: %v", err)
	}
}
