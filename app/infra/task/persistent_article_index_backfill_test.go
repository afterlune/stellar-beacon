package task

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type fakeArticleIndexBackfillStates struct {
	state                port.ArticleIndexBackfillState
	checkpointCount      int
	pauseAfterCheckpoint int
	claimCount           int
	lastError            string
	pauseBeforeComplete  bool
}

func (f *fakeArticleIndexBackfillStates) CreateOrGet(_ context.Context, state port.ArticleIndexBackfillState) (port.ArticleIndexBackfillState, error) {
	if f.state.ID == "" {
		f.state = state
		if f.state.Status == "" {
			f.state.Status = port.ArticleIndexBackfillPending
		}
	}
	if f.state.ID != state.ID || f.state.IndexUID != state.IndexUID || f.state.ModelVersion != state.ModelVersion {
		return port.ArticleIndexBackfillState{}, apperrors.Conflict("test.backfill.create", "definition mismatch")
	}
	return f.state, nil
}

func (f *fakeArticleIndexBackfillStates) Get(context.Context, string) (port.ArticleIndexBackfillState, error) {
	if f.state.ID == "" {
		return port.ArticleIndexBackfillState{}, apperrors.NotFound("test.backfill.get")
	}
	return f.state, nil
}

func (f *fakeArticleIndexBackfillStates) Claim(_ context.Context, id, owner string, now time.Time, lease time.Duration) (port.ArticleIndexBackfillState, bool, error) {
	if f.state.ID != id {
		return port.ArticleIndexBackfillState{}, false, apperrors.NotFound("test.backfill.claim")
	}
	if f.state.PauseRequested || (f.state.Status == port.ArticleIndexBackfillRunning && f.state.LeaseUntil.After(now)) || f.state.Status == port.ArticleIndexBackfillCompleted {
		return port.ArticleIndexBackfillState{}, false, nil
	}
	f.claimCount++
	f.state.Status = port.ArticleIndexBackfillRunning
	f.state.LeaseOwner = owner
	f.state.LeaseUntil = now.Add(lease)
	return f.state, true, nil
}

func (f *fakeArticleIndexBackfillStates) RequestPause(_ context.Context, id string) error {
	if f.state.ID != id {
		return apperrors.NotFound("test.backfill.pause")
	}
	f.state.PauseRequested = true
	return nil
}

func (f *fakeArticleIndexBackfillStates) Resume(_ context.Context, id string) error {
	if f.state.ID != id {
		return apperrors.NotFound("test.backfill.resume")
	}
	f.state.Status = port.ArticleIndexBackfillPending
	f.state.PauseRequested = false
	f.state.LeaseOwner = ""
	f.state.LeaseUntil = time.Time{}
	f.lastError = ""
	f.state.LastError = ""
	return nil
}

func (f *fakeArticleIndexBackfillStates) Checkpoint(_ context.Context, id, owner string, cursor int, processedArticles, indexedChunks int64, now time.Time, lease time.Duration) error {
	if f.state.ID != id || f.state.Status != port.ArticleIndexBackfillRunning || f.state.LeaseOwner != owner {
		return apperrors.Conflict("test.backfill.checkpoint", "lease mismatch")
	}
	if cursor < f.state.Cursor || processedArticles < f.state.ProcessedArticles || indexedChunks < f.state.IndexedChunks {
		return apperrors.Conflict("test.backfill.checkpoint", "progress regressed")
	}
	f.state.Cursor = cursor
	f.state.ProcessedArticles = processedArticles
	f.state.IndexedChunks = indexedChunks
	f.state.LeaseUntil = now.Add(lease)
	f.checkpointCount++
	if f.pauseAfterCheckpoint > 0 && f.checkpointCount >= f.pauseAfterCheckpoint {
		f.state.PauseRequested = true
	}
	return nil
}

func (f *fakeArticleIndexBackfillStates) Pause(_ context.Context, id, owner string, _ time.Time) error {
	if f.state.ID != id || f.state.LeaseOwner != owner {
		return apperrors.Conflict("test.backfill.pause", "lease mismatch")
	}
	f.state.Status = port.ArticleIndexBackfillPaused
	f.state.PauseRequested = false
	f.state.LeaseOwner = ""
	f.state.LeaseUntil = time.Time{}
	return nil
}

func (f *fakeArticleIndexBackfillStates) Complete(_ context.Context, id, owner string, _ time.Time) error {
	if f.pauseBeforeComplete {
		f.state.PauseRequested = true
	}
	if f.state.ID != id || f.state.LeaseOwner != owner || f.state.PauseRequested {
		return apperrors.Conflict("test.backfill.complete", "lease or pause mismatch")
	}
	f.state.Status = port.ArticleIndexBackfillCompleted
	f.state.LeaseOwner = ""
	f.state.LeaseUntil = time.Time{}
	return nil
}

func (f *fakeArticleIndexBackfillStates) Fail(_ context.Context, id, owner, lastError string, _ time.Time) error {
	if f.state.ID != id || f.state.LeaseOwner != owner {
		return apperrors.Conflict("test.backfill.fail", "lease mismatch")
	}
	f.state.Status = port.ArticleIndexBackfillFailed
	f.state.LeaseOwner = ""
	f.state.LeaseUntil = time.Time{}
	f.state.LastError = lastError
	f.lastError = lastError
	return nil
}

func persistentBackfillTemplate() port.ArticleIndexBackfillState {
	spec := articleIndexTestSpec()
	return port.ArticleIndexBackfillState{
		ID:                 "backfill-test",
		IndexUID:           spec.UID,
		IndexVersion:       spec.IndexVersion,
		Provider:           spec.Provider,
		Model:              spec.Model,
		ModelVersion:       spec.ModelVersion,
		Dimension:          spec.Dimension,
		EmbeddingBatchSize: spec.EmbeddingBatchSize,
		PageSize:           2,
		Status:             port.ArticleIndexBackfillPending,
	}
}

func newPersistentArticleIndexBackfillForTest(t *testing.T, states *fakeArticleIndexBackfillStates, index *fakeArticleChunkIndex) *PersistentArticleIndexBackfill {
	t.Helper()
	sources := &fakeArticleIndexSourceRepository{sources: []port.ArticleIndexSource{
		articleIndexSource(1),
		articleIndexSource(2),
		articleIndexSource(3),
	}}
	backfill := newArticleIndexBackfillForTest(t, sources, index, 2)
	persistent, err := NewPersistentArticleIndexBackfill(PersistentArticleIndexBackfillDeps{
		Backfill:      backfill,
		States:        states,
		Template:      persistentBackfillTemplate(),
		WorkerID:      "persistent-test-worker",
		LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("NewPersistentArticleIndexBackfill() error = %v", err)
	}
	return persistent
}

func TestPersistentArticleIndexBackfillPausesAfterCheckpointAndResumes(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{pauseAfterCheckpoint: 1}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	state, err := persistent.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if state.Status != port.ArticleIndexBackfillPaused || state.Cursor != 1 || state.ProcessedArticles != 1 {
		t.Fatalf("paused state = %+v, want cursor 1 and paused", state)
	}

	if err := persistent.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	states.pauseAfterCheckpoint = 0
	state, err = persistent.Run(context.Background())
	if err != nil {
		t.Fatalf("resumed Run() error = %v", err)
	}
	if state.Status != port.ArticleIndexBackfillCompleted || state.Cursor != 3 || state.ProcessedArticles != 3 {
		t.Fatalf("completed state = %+v, want cursor 3 and completed", state)
	}
	if states.claimCount != 2 {
		t.Fatalf("claim count = %d, want one claim per run", states.claimCount)
	}
}

func TestPersistentArticleIndexBackfillPerRunLimitPausesAtCheckpoint(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	state, err := persistent.RunWithLimit(context.Background(), 1)
	if err != nil {
		t.Fatalf("RunWithLimit() error = %v", err)
	}
	if state.Status != port.ArticleIndexBackfillPaused || state.Cursor != 1 || state.ProcessedArticles != 1 {
		t.Fatalf("bounded state = %+v, want one checkpointed article and paused status", state)
	}

	if err := persistent.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	state, err = persistent.Run(context.Background())
	if err != nil {
		t.Fatalf("unbounded continuation error = %v", err)
	}
	if state.Status != port.ArticleIndexBackfillCompleted || state.Cursor != 3 || state.ProcessedArticles != 3 {
		t.Fatalf("continued state = %+v, want completion without carrying over the pilot limit", state)
	}
}

func TestPersistentArticleIndexBackfillRejectsNegativePerRunLimit(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := persistent.RunWithLimit(context.Background(), -1)
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("RunWithLimit() error kind = %v, want validation", apperrors.KindOf(err))
	}
	if states.claimCount != 0 {
		t.Fatalf("claim count = %d, want no claim for invalid limit", states.claimCount)
	}
}

func TestPersistentArticleIndexBackfillPersistsFailureAndCanResume(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	index := &fakeArticleChunkIndex{upsertErr: errors.New("index unavailable")}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, index)
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatal(err)
	}

	state, err := persistent.Run(context.Background())
	if err == nil || state.Status != port.ArticleIndexBackfillFailed || state.Cursor != 0 || state.LastError != "internal_error" {
		t.Fatalf("failed state = %+v error=%v, want failed at cursor 0 with sanitized error", state, err)
	}
	if err := persistent.Resume(context.Background()); err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	index.upsertErr = nil
	state, err = persistent.Run(context.Background())
	if err != nil || state.Status != port.ArticleIndexBackfillCompleted || state.Cursor != 3 {
		t.Fatalf("resumed state = %+v error=%v, want completed", state, err)
	}
}

func TestPersistentArticleIndexBackfillRejectsSecondActiveRunner(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := states.Claim(context.Background(), persistent.template.ID, "other-worker", time.Now().UTC(), time.Minute)
	if err != nil || !ok || claimed.Status != port.ArticleIndexBackfillRunning {
		t.Fatalf("pre-claim = state=%+v ok=%v err=%v", claimed, ok, err)
	}
	state, err := persistent.Run(context.Background())
	if !apperrors.IsKind(err, apperrors.KindConflict) || state.Status != port.ArticleIndexBackfillRunning {
		t.Fatalf("second Run() = state=%+v error=%v, want conflict while running", state, err)
	}
}

func TestNewPersistentArticleIndexBackfillValidatesTargetContract(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	backfill := newArticleIndexBackfillForTest(t, &fakeArticleIndexSourceRepository{}, &fakeArticleChunkIndex{}, 2)
	template := persistentBackfillTemplate()
	template.ModelVersion = "different"
	if _, err := NewPersistentArticleIndexBackfill(PersistentArticleIndexBackfillDeps{
		Backfill: backfill,
		States:   states,
		Template: template,
	}); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("contract mismatch error kind = %v, want conflict", apperrors.KindOf(err))
	}
}

func TestPersistentArticleIndexBackfillRejectsConfigurationDriftOnResume(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{}
	first := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := first.Create(context.Background()); err != nil {
		t.Fatal(err)
	}

	driftedBackfill := newArticleIndexBackfillForTest(t, &fakeArticleIndexSourceRepository{}, &fakeArticleChunkIndex{}, 3)
	driftedTemplate := persistentBackfillTemplate()
	driftedTemplate.PageSize = 3
	drifted, err := NewPersistentArticleIndexBackfill(PersistentArticleIndexBackfillDeps{
		Backfill: driftedBackfill,
		States:   states,
		Template: driftedTemplate,
		WorkerID: "drifted-worker",
	})
	if err != nil {
		t.Fatalf("NewPersistentArticleIndexBackfill() error = %v", err)
	}

	state, err := drifted.Run(context.Background())
	if !apperrors.IsKind(err, apperrors.KindConflict) || state.PageSize != 2 {
		t.Fatalf("drifted Run() = state=%+v error=%v, want persisted contract conflict", state, err)
	}
	if states.claimCount != 0 {
		t.Fatalf("claim count = %d, want no claim after contract mismatch", states.claimCount)
	}
}

func TestPersistentArticleIndexBackfillSettlesFinalPauseRace(t *testing.T) {
	states := &fakeArticleIndexBackfillStates{pauseBeforeComplete: true}
	persistent := newPersistentArticleIndexBackfillForTest(t, states, &fakeArticleChunkIndex{})
	if _, err := persistent.Create(context.Background()); err != nil {
		t.Fatal(err)
	}

	state, err := persistent.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if state.Status != port.ArticleIndexBackfillPaused || state.Cursor != 3 || state.ProcessedArticles != 3 {
		t.Fatalf("final pause race state = %+v, want paused after final checkpoint", state)
	}
}
