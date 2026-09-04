package task

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/search"
)

const (
	// DefaultArticleIndexPlanPageSize reads one complete article at a time.
	// Planning also loads full bodies, so it must not create a large retained
	// page merely because it does not call a Provider.
	DefaultArticleIndexPlanPageSize = 1
	maxArticleIndexPlanPageSize     = 25
	maxArticleIndexPlanIDSamples    = 20
)

// ArticleIndexPlan is the data-only preflight for a full article-index
// rebuild. It deliberately has no embedding gateway, Meilisearch index, or
// durable backfill state dependency, so running it cannot make an external
// provider call or mutate either the database or search service.
type ArticleIndexPlan struct {
	sources  port.ArticleIndexSourceRepository
	chunker  search.MarkdownChunker
	pageSize int
	index    string
}

type ArticleIndexPlanDeps struct {
	Sources  port.ArticleIndexSourceRepository
	Chunker  search.MarkdownChunker
	PageSize int
	Index    string
}

// ArticleIndexPlanResult contains only aggregate data and bounded article ID
// samples. Article titles, bodies, tags, and provider configuration are never
// copied into the result or emitted by the command.
type ArticleIndexPlanResult struct {
	Index             string `json:"index"`
	PageSize          int    `json:"pageSize"`
	AfterArticleID    int    `json:"afterArticleId"`
	LastArticleID     int    `json:"lastArticleId"`
	ScannedArticles   int64  `json:"scannedArticles"`
	EstimatedChunks   int64  `json:"estimatedChunks"`
	InvalidArticles   int64  `json:"invalidArticles"`
	InvalidArticleIDs []int  `json:"invalidArticleIds,omitempty"`
	Limited           bool   `json:"limited"`
	Complete          bool   `json:"complete"`
	ReadyForBackfill  bool   `json:"readyForBackfill"`
}

func NewArticleIndexPlan(deps ArticleIndexPlanDeps) (*ArticleIndexPlan, error) {
	if deps.Sources == nil {
		return nil, apperrors.Invalid("task.article_index_plan.dependencies", "article index source repository is required")
	}
	if deps.PageSize == 0 {
		deps.PageSize = DefaultArticleIndexPlanPageSize
	}
	if deps.PageSize < 1 || deps.PageSize > maxArticleIndexPlanPageSize {
		return nil, apperrors.Invalid("task.article_index_plan.page_size", "page size must be between 1 and 25")
	}
	return &ArticleIndexPlan{
		sources:  deps.Sources,
		chunker:  deps.Chunker,
		pageSize: deps.PageSize,
		index:    strings.TrimSpace(deps.Index),
	}, nil
}

func (p *ArticleIndexPlan) PageSize() int {
	if p == nil {
		return 0
	}
	return p.pageSize
}

// Run scans a keyset-paginated public article snapshot and counts the chunks
// produced by the exact same local chunking/normalization path used by the
// projector. maxArticles is zero for an uncapped scan; a positive value is a
// deliberate bounded sample and is reported as Limited rather than Complete.
func (p *ArticleIndexPlan) Run(ctx context.Context, afterArticleID, maxArticles int) (ArticleIndexPlanResult, error) {
	result := ArticleIndexPlanResult{}
	if p == nil {
		return result, errors.New("article index plan is nil")
	}
	if afterArticleID < 0 {
		return result, apperrors.Invalid("task.article_index_plan.cursor", "after article id cannot be negative")
	}
	if maxArticles < 0 {
		return result, apperrors.Invalid("task.article_index_plan.max_articles", "max articles cannot be negative")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result.Index = p.index
	result.PageSize = p.pageSize
	result.AfterArticleID = afterArticleID
	result.LastArticleID = afterArticleID

	cursor := afterArticleID
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		pageSize := p.pageSize
		if maxArticles > 0 {
			remaining := maxArticles - int(result.ScannedArticles)
			if remaining <= 0 {
				result.Limited = true
				return finalizeArticleIndexPlanResult(result), nil
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}
		sources, err := p.sources.ListPublicArticleIndexSources(ctx, cursor, pageSize)
		if err != nil {
			return result, fmt.Errorf("list article index plan sources after %d: %w", cursor, err)
		}
		if len(sources) == 0 {
			result.Complete = true
			return finalizeArticleIndexPlanResult(result), nil
		}
		if len(sources) > pageSize {
			return result, apperrors.Invalid("task.article_index_plan.page", "source repository returned more records than requested")
		}

		for _, source := range sources {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			articleID := source.Article.Id
			if articleID <= cursor {
				return result, apperrors.Invalid("task.article_index_plan.cursor", "source repository must return strictly increasing article IDs")
			}
			result.ScannedArticles++
			result.LastArticleID = articleID
			cursor = articleID
			documents, err := search.BuildArticleChunkDocuments(articleForIndex(source.Article, source.CategoryName), source.Tags, p.chunker)
			if err != nil {
				result.InvalidArticles++
				if len(result.InvalidArticleIDs) < maxArticleIndexPlanIDSamples {
					result.InvalidArticleIDs = append(result.InvalidArticleIDs, articleID)
				}
			} else {
				result.EstimatedChunks += int64(len(documents))
			}
			if maxArticles > 0 && int(result.ScannedArticles) >= maxArticles {
				result.Limited = true
				return finalizeArticleIndexPlanResult(result), nil
			}
		}

		if len(sources) < pageSize {
			result.Complete = true
			return finalizeArticleIndexPlanResult(result), nil
		}
	}
}

func finalizeArticleIndexPlanResult(result ArticleIndexPlanResult) ArticleIndexPlanResult {
	result.ReadyForBackfill = result.Complete && !result.Limited && result.InvalidArticles == 0
	return result
}
