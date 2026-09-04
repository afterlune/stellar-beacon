package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	// DefaultArticleIndexBackfillPageSize intentionally reads one complete
	// article at a time. The source query includes the full article body, so a
	// large page can retain many megabytes before the projector has a chance to
	// enforce its per-article budget.
	DefaultArticleIndexBackfillPageSize = 1
	maxArticleIndexBackfillPageSize     = 25
	defaultArticleIndexBackfillLease    = 10 * time.Minute
)

var ErrArticleIndexBackfillPaused = errors.New("article index backfill paused")

// ArticleIndexBackfillProgress is emitted after one article has been
// projected successfully. A persistence coordinator can checkpoint it and
// then honor a pause request without losing the completed article.
type ArticleIndexBackfillProgress struct {
	ArticleID         int
	ProcessedArticles int
	IndexedChunks     int
	LastArticleID     int
}

type ArticleIndexBackfillHooks struct {
	AfterArticle func(context.Context, ArticleIndexBackfillProgress) error
	ShouldPause  func(context.Context) (bool, error)
	// MaxArticles is a per-run safety limit. When it is reached, the current
	// article has already been checkpointed and the run returns the same
	// cooperative pause signal used by an operator pause request.
	MaxArticles int
}

// ArticleIndexBackfillDeps is the explicit composition contract for a
// one-shot, keyset-paginated rebuild. The projector owns a fixed target index
// and embedding contract; the backfill never discovers or writes an older
// index implicitly.
type ArticleIndexBackfillDeps struct {
	Sources   port.ArticleIndexSourceRepository
	Projector *ArticleIndexProjector
	PageSize  int
}

// ArticleIndexBackfill reads complete public article sources and rebuilds the
// projector's target index. It has no internal goroutine or durable state:
// LastArticleID is the resume cursor that a later M2-09 command can persist.
type ArticleIndexBackfill struct {
	sources   port.ArticleIndexSourceRepository
	projector *ArticleIndexProjector
	pageSize  int
}

type ArticleIndexBackfillResult struct {
	Index             string
	ProcessedArticles int
	IndexedChunks     int
	LastArticleID     int
}

func NewArticleIndexBackfill(deps ArticleIndexBackfillDeps) (*ArticleIndexBackfill, error) {
	if deps.Sources == nil {
		return nil, apperrors.Invalid("task.article_index_backfill.dependencies", "article index source repository is required")
	}
	if deps.Projector == nil {
		return nil, apperrors.Invalid("task.article_index_backfill.dependencies", "article index projector is required")
	}
	if deps.PageSize == 0 {
		deps.PageSize = DefaultArticleIndexBackfillPageSize
	}
	if deps.PageSize < 1 || deps.PageSize > maxArticleIndexBackfillPageSize {
		return nil, apperrors.Invalid("task.article_index_backfill.page_size", "page size must be between 1 and 25")
	}
	return &ArticleIndexBackfill{
		sources:   deps.Sources,
		projector: deps.Projector,
		pageSize:  deps.PageSize,
	}, nil
}

func (b *ArticleIndexBackfill) PageSize() int {
	if b == nil {
		return 0
	}
	return b.pageSize
}

// Run rebuilds from an exclusive article ID cursor. The cursor advances only
// after an article has been projected successfully, so a caller can resume
// after a transient provider or Meilisearch failure without skipping it.
func (b *ArticleIndexBackfill) Run(ctx context.Context, afterArticleID int) (ArticleIndexBackfillResult, error) {
	return b.RunWithHooks(ctx, afterArticleID, ArticleIndexBackfillHooks{})
}

// RunWithHooks is the checkpoint-aware form used by the persistent command.
// Hooks run in the caller's goroutine and are never detached, so a pause or a
// checkpoint failure has a deterministic cursor boundary.
func (b *ArticleIndexBackfill) RunWithHooks(ctx context.Context, afterArticleID int, hooks ArticleIndexBackfillHooks) (ArticleIndexBackfillResult, error) {
	result := ArticleIndexBackfillResult{}
	if b == nil {
		return result, errors.New("article index backfill is nil")
	}
	result.Index = b.projector.spec.UID
	if afterArticleID < 0 {
		return result, apperrors.Invalid("task.article_index_backfill.cursor", "after article id cannot be negative")
	}
	if hooks.MaxArticles < 0 {
		return result, apperrors.Invalid("task.article_index_backfill.max_articles", "max articles cannot be negative")
	}
	result.LastArticleID = afterArticleID
	if ctx == nil {
		ctx = context.Background()
	}

	cursor := afterArticleID
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if paused, err := backfillShouldPause(ctx, hooks); err != nil {
			return result, fmt.Errorf("check article index backfill pause state: %w", err)
		} else if paused {
			return result, ErrArticleIndexBackfillPaused
		}
		sources, err := b.sources.ListPublicArticleIndexSources(ctx, cursor, b.pageSize)
		if err != nil {
			return result, fmt.Errorf("list article index sources after %d: %w", cursor, err)
		}
		if len(sources) == 0 {
			return result, nil
		}
		if len(sources) > b.pageSize {
			return result, apperrors.Invalid("task.article_index_backfill.page", "source repository returned more records than requested")
		}

		for _, source := range sources {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			articleID := source.Article.Id
			if articleID <= cursor {
				return result, apperrors.Invalid("task.article_index_backfill.cursor", "source repository must return strictly increasing article IDs")
			}
			indexedChunks, err := b.projector.UpsertSource(ctx, source)
			if err != nil {
				return result, fmt.Errorf("project article %d: %w", articleID, err)
			}
			result.ProcessedArticles++
			result.IndexedChunks += indexedChunks
			cursor = articleID
			result.LastArticleID = cursor
			if hooks.AfterArticle != nil {
				if err := hooks.AfterArticle(ctx, ArticleIndexBackfillProgress{
					ArticleID:         articleID,
					ProcessedArticles: result.ProcessedArticles,
					IndexedChunks:     result.IndexedChunks,
					LastArticleID:     result.LastArticleID,
				}); err != nil {
					return result, err
				}
			}
			if hooks.MaxArticles > 0 && result.ProcessedArticles >= hooks.MaxArticles {
				return result, ErrArticleIndexBackfillPaused
			}
			if paused, err := backfillShouldPause(ctx, hooks); err != nil {
				return result, fmt.Errorf("check article index backfill pause state: %w", err)
			} else if paused {
				return result, ErrArticleIndexBackfillPaused
			}
		}

		if len(sources) < b.pageSize {
			return result, nil
		}
	}
}

func backfillShouldPause(ctx context.Context, hooks ArticleIndexBackfillHooks) (bool, error) {
	if hooks.ShouldPause == nil {
		return false, nil
	}
	return hooks.ShouldPause(ctx)
}
