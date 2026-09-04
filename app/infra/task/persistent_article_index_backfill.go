package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
)

const persistentBackfillSettleTimeout = 5 * time.Second

// PersistentArticleIndexBackfill coordinates the one-shot projector with a
// durable state row. The state row is the source of truth for pause/resume;
// this type owns no goroutine and can safely be run by a short-lived CLI.
type PersistentArticleIndexBackfill struct {
	backfill      *ArticleIndexBackfill
	states        port.ArticleIndexBackfillRepository
	template      port.ArticleIndexBackfillState
	workerID      string
	leaseDuration time.Duration
	now           func() time.Time
}

type PersistentArticleIndexBackfillDeps struct {
	Backfill      *ArticleIndexBackfill
	States        port.ArticleIndexBackfillRepository
	Template      port.ArticleIndexBackfillState
	WorkerID      string
	LeaseDuration time.Duration
}

func NewPersistentArticleIndexBackfill(deps PersistentArticleIndexBackfillDeps) (*PersistentArticleIndexBackfill, error) {
	if deps.Backfill == nil {
		return nil, apperrors.Invalid("task.persistent_article_index_backfill.dependencies", "article index backfill is required")
	}
	if deps.Backfill.projector == nil {
		return nil, apperrors.Invalid("task.persistent_article_index_backfill.dependencies", "article index projector is required")
	}
	if deps.States == nil {
		return nil, apperrors.Invalid("task.persistent_article_index_backfill.dependencies", "backfill state repository is required")
	}
	if deps.Template.Status == "" {
		deps.Template.Status = port.ArticleIndexBackfillPending
	}
	if err := deps.Template.Validate(); err != nil {
		return nil, apperrors.Invalid("task.persistent_article_index_backfill.template", err.Error())
	}
	spec := deps.Backfill.projector.spec
	if deps.Template.IndexUID != spec.UID ||
		deps.Template.IndexVersion != spec.IndexVersion ||
		deps.Template.Provider != spec.Provider ||
		deps.Template.Model != spec.Model ||
		deps.Template.ModelVersion != spec.ModelVersion ||
		deps.Template.Dimension != spec.Dimension ||
		deps.Template.EmbeddingBatchSize != spec.EmbeddingBatchSize ||
		deps.Template.PageSize != deps.Backfill.pageSize {
		return nil, apperrors.Conflict("task.persistent_article_index_backfill.template", "backfill state does not match the target index specification")
	}
	if strings.TrimSpace(deps.WorkerID) == "" {
		deps.WorkerID = "article-index-backfill-" + uuid.NewString()
	}
	if deps.LeaseDuration <= 0 {
		deps.LeaseDuration = defaultArticleIndexBackfillLease
	}
	return &PersistentArticleIndexBackfill{
		backfill:      deps.Backfill,
		states:        deps.States,
		template:      deps.Template,
		workerID:      strings.TrimSpace(deps.WorkerID),
		leaseDuration: deps.LeaseDuration,
		now:           func() time.Time { return time.Now().UTC() },
	}, nil
}

func (p *PersistentArticleIndexBackfill) Create(ctx context.Context) (port.ArticleIndexBackfillState, error) {
	if p == nil {
		return port.ArticleIndexBackfillState{}, errors.New("persistent article index backfill is nil")
	}
	return p.states.CreateOrGet(ctx, p.template)
}

func (p *PersistentArticleIndexBackfill) Status(ctx context.Context) (port.ArticleIndexBackfillState, error) {
	if p == nil {
		return port.ArticleIndexBackfillState{}, errors.New("persistent article index backfill is nil")
	}
	return p.states.Get(ctx, p.template.ID)
}

func (p *PersistentArticleIndexBackfill) RequestPause(ctx context.Context) error {
	if p == nil {
		return errors.New("persistent article index backfill is nil")
	}
	return p.states.RequestPause(ctx, p.template.ID)
}

func (p *PersistentArticleIndexBackfill) Resume(ctx context.Context) error {
	if p == nil {
		return errors.New("persistent article index backfill is nil")
	}
	return p.states.Resume(ctx, p.template.ID)
}

// Run claims the persisted state and projects until completion or a pause.
// Context cancellation is treated as a cooperative pause after the current
// article checkpoint, while provider/database failures are persisted as
// failed and keep the last successful cursor.
func (p *PersistentArticleIndexBackfill) Run(ctx context.Context) (port.ArticleIndexBackfillState, error) {
	return p.run(ctx, 0)
}

// RunWithLimit performs one bounded invocation. The limit applies only to
// this invocation; once the checkpointed state is paused, a later Run call can
// continue the full backfill without silently carrying over the pilot cap.
func (p *PersistentArticleIndexBackfill) RunWithLimit(ctx context.Context, maxArticles int) (port.ArticleIndexBackfillState, error) {
	return p.run(ctx, maxArticles)
}

func (p *PersistentArticleIndexBackfill) run(ctx context.Context, maxArticles int) (port.ArticleIndexBackfillState, error) {
	if p == nil {
		return port.ArticleIndexBackfillState{}, errors.New("persistent article index backfill is nil")
	}
	if maxArticles < 0 {
		return port.ArticleIndexBackfillState{}, apperrors.Invalid("task.persistent_article_index_backfill.max_articles", "max articles cannot be negative")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	persisted, err := p.states.Get(ctx, p.template.ID)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	if !samePersistentBackfillDefinition(persisted, p.template) {
		return persisted, apperrors.Conflict("task.persistent_article_index_backfill.run", "persisted backfill contract does not match the current configuration")
	}
	claimed, ok, err := p.states.Claim(ctx, p.template.ID, p.workerID, p.currentTime(), p.leaseDuration)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	if !ok {
		state, stateErr := p.Status(ctx)
		if stateErr != nil {
			return port.ArticleIndexBackfillState{}, stateErr
		}
		switch state.Status {
		case port.ArticleIndexBackfillCompleted:
			return state, apperrors.Conflict("task.persistent_article_index_backfill.run", "backfill is already completed")
		case port.ArticleIndexBackfillRunning:
			return state, apperrors.Conflict("task.persistent_article_index_backfill.run", "backfill is already running")
		default:
			return state, apperrors.Conflict("task.persistent_article_index_backfill.run", "backfill is paused or not ready to run")
		}
	}
	if !samePersistentBackfillDefinition(claimed, p.template) {
		return claimed, apperrors.Conflict("task.persistent_article_index_backfill.run", "claimed backfill contract does not match the current configuration")
	}

	baseProcessed := claimed.ProcessedArticles
	baseChunks := claimed.IndexedChunks
	_, runErr := p.backfill.RunWithHooks(ctx, claimed.Cursor, ArticleIndexBackfillHooks{
		MaxArticles: maxArticles,
		AfterArticle: func(checkpointCtx context.Context, progress ArticleIndexBackfillProgress) error {
			return p.states.Checkpoint(
				checkpointCtx,
				p.template.ID,
				p.workerID,
				progress.LastArticleID,
				baseProcessed+int64(progress.ProcessedArticles),
				baseChunks+int64(progress.IndexedChunks),
				p.currentTime(),
				p.leaseDuration,
			)
		},
		ShouldPause: func(pauseCtx context.Context) (bool, error) {
			state, err := p.states.Get(pauseCtx, p.template.ID)
			if err != nil {
				return false, err
			}
			return state.PauseRequested, nil
		},
	})
	if runErr == nil {
		if err := ctx.Err(); err != nil {
			return p.pauseAfterCancellation(err)
		}
		if completeErr := p.states.Complete(ctx, p.template.ID, p.workerID, p.currentTime()); completeErr != nil {
			// A pause request may race with the final empty-page read. Complete
			// rejects pause_requested atomically; settle that owned lease as
			// paused instead of leaving it running until expiry.
			state, statusErr := p.states.Get(ctx, p.template.ID)
			if statusErr == nil &&
				state.Status == port.ArticleIndexBackfillRunning &&
				state.LeaseOwner == p.workerID &&
				state.PauseRequested {
				if pauseErr := p.states.Pause(ctx, p.template.ID, p.workerID, p.currentTime()); pauseErr == nil {
					return p.Status(ctx)
				}
			}
			return p.bestEffortStatus(ctx, runErr, completeErr)
		}
		state, err := p.Status(ctx)
		if err != nil {
			return port.ArticleIndexBackfillState{}, err
		}
		return state, nil
	}

	if errors.Is(runErr, ErrArticleIndexBackfillPaused) {
		if err := p.states.Pause(ctx, p.template.ID, p.workerID, p.currentTime()); err != nil {
			return p.bestEffortStatus(ctx, runErr, err)
		}
		state, err := p.Status(ctx)
		if err != nil {
			return port.ArticleIndexBackfillState{}, err
		}
		return state, nil
	}

	// A provider may return a child timeout while the parent CLI context is
	// still alive. Only the parent context is a cooperative shutdown signal;
	// provider failures must remain visible as failed runs.
	if ctx.Err() != nil {
		return p.pauseAfterCancellation(runErr)
	}

	if err := p.states.Fail(ctx, p.template.ID, p.workerID, safeWorkerError(runErr), p.currentTime()); err != nil {
		return p.bestEffortStatus(ctx, runErr, err)
	}
	state, err := p.Status(ctx)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	return state, runErr
}

func (p *PersistentArticleIndexBackfill) pauseAfterCancellation(runErr error) (port.ArticleIndexBackfillState, error) {
	settleCtx, cancel := context.WithTimeout(context.Background(), persistentBackfillSettleTimeout)
	defer cancel()
	if err := p.states.Pause(settleCtx, p.template.ID, p.workerID, p.currentTime()); err != nil {
		return p.bestEffortStatus(settleCtx, runErr, err)
	}
	state, err := p.states.Get(settleCtx, p.template.ID)
	if err != nil {
		return port.ArticleIndexBackfillState{}, err
	}
	return state, nil
}

func (p *PersistentArticleIndexBackfill) bestEffortStatus(ctx context.Context, runErr, settleErr error) (port.ArticleIndexBackfillState, error) {
	state, statusErr := p.states.Get(ctx, p.template.ID)
	switch {
	case runErr == nil && statusErr == nil:
		return state, fmt.Errorf("settle backfill state: %w", settleErr)
	case runErr == nil && statusErr != nil:
		return port.ArticleIndexBackfillState{}, fmt.Errorf("settle backfill state: %w; read backfill state: %v", settleErr, statusErr)
	case statusErr == nil:
		return state, fmt.Errorf("backfill run failed: %w; settle backfill state: %v", runErr, settleErr)
	default:
		return port.ArticleIndexBackfillState{}, fmt.Errorf("backfill run failed: %w; settle backfill state: %v; read backfill state: %v", runErr, settleErr, statusErr)
	}
}

func samePersistentBackfillDefinition(left, right port.ArticleIndexBackfillState) bool {
	return left.ID == right.ID &&
		left.IndexUID == right.IndexUID &&
		left.IndexVersion == right.IndexVersion &&
		left.Provider == right.Provider &&
		left.Model == right.Model &&
		left.ModelVersion == right.ModelVersion &&
		left.Dimension == right.Dimension &&
		left.EmbeddingBatchSize == right.EmbeddingBatchSize &&
		left.PageSize == right.PageSize
}

func (p *PersistentArticleIndexBackfill) currentTime() time.Time {
	if p == nil || p.now == nil {
		return time.Now().UTC()
	}
	return p.now().UTC()
}
