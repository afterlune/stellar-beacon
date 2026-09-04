package search

import (
	"context"
	"errors"

	apperrors "benetnasch/app/domain/errors"

	"github.com/meilisearch/meilisearch-go"
)

// ArticleChunksIndexSwapPlan describes one explicit, reversible swap. Current
// and Candidate are physical Meilisearch UIDs; the candidate must already be
// provisioned and fully backfilled by the caller before Swap is invoked.
//
// Meilisearch performs the swap as one asynchronous atomic task. No index is
// deleted, and reversing the two fields produces the rollback operation.
type ArticleChunksIndexSwapPlan struct {
	Current   ArticleChunksIndexSpec
	Candidate ArticleChunksIndexSpec
}

func NewArticleChunksIndexSwapPlan(current, candidate ArticleChunksIndexSpec) (ArticleChunksIndexSwapPlan, error) {
	plan := ArticleChunksIndexSwapPlan{Current: current, Candidate: candidate}
	if err := plan.Validate(); err != nil {
		return ArticleChunksIndexSwapPlan{}, err
	}
	return plan, nil
}

func (p ArticleChunksIndexSwapPlan) Validate() error {
	if err := ValidateArticleChunksIndexMigration(p.Current, p.Candidate); err != nil {
		return err
	}
	if p.Current.UID == p.Candidate.UID {
		return apperrors.Invalid("search.article_chunks.swap", "current and candidate index UIDs must differ")
	}
	return nil
}

// Reverse returns the same swap in rollback order. It does not contact
// Meilisearch, so callers can decide when the rollback task is safe to submit.
func (p ArticleChunksIndexSwapPlan) Reverse() ArticleChunksIndexSwapPlan {
	return ArticleChunksIndexSwapPlan{Current: p.Candidate, Candidate: p.Current}
}

// SwapArticleChunksIndex verifies both indexes and submits a non-rename swap.
// With rename=false Meilisearch exchanges documents, settings, and task
// history atomically. The returned task must be waited on by the operator
// before declaring the cutover complete.
func SwapArticleChunksIndex(ctx context.Context, client meilisearch.ServiceManager, plan ArticleChunksIndexSwapPlan) (*meilisearch.TaskInfo, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, apperrors.Unavailable("search.article_chunks.swap", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for _, item := range []struct {
		name string
		spec ArticleChunksIndexSpec
	}{
		{name: "current", spec: plan.Current},
		{name: "candidate", spec: plan.Candidate},
	} {
		index, err := client.GetIndexWithContext(ctx, item.spec.UID)
		if err != nil {
			if isMeiliNotFound(err) {
				return nil, apperrors.NotFound("search.article_chunks.swap." + item.name)
			}
			return nil, apperrors.Unavailable("search.article_chunks.swap.inspect", err)
		}
		if index == nil {
			return nil, apperrors.Unavailable("search.article_chunks.swap.inspect", errors.New("Meilisearch returned an empty index response"))
		}
		if index.PrimaryKey != item.spec.PrimaryKey {
			return nil, apperrors.Conflict("search.article_chunks.swap", "index primary key does not match the plan")
		}
		if err := validateArticleChunksIndexSettings(ctx, client, item.spec.UID); err != nil {
			return nil, err
		}
	}

	task, err := client.SwapIndexesWithContext(ctx, []*meilisearch.SwapIndexesParams{
		{
			Indexes: []string{plan.Current.UID, plan.Candidate.UID},
			Rename:  false,
		},
	})
	if err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.swap", err)
	}
	if task == nil {
		return nil, apperrors.Unavailable("search.article_chunks.swap", errors.New("Meilisearch returned an empty swap task"))
	}
	return task, nil
}
