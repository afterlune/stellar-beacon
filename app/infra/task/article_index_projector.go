package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/search"
)

const (
	// These budgets are deliberately conservative defaults for a one-article,
	// one-batch-at-a-time projector. They protect the process from turning a
	// malformed or unexpectedly large article into an unbounded allocation.
	DefaultArticleIndexMaxArticleContentBytes = 2 << 20
	DefaultArticleIndexMaxChunksPerArticle    = 4096
	DefaultArticleIndexMaxEmbeddingBatchSize  = 32
	DefaultArticleIndexMaxEmbeddingBatchBytes = 256 << 10
)

// ArticleIndexResourceLimits are process-local safety budgets. They do not
// change the index contract; they only reject work before a provider request
// or Meilisearch write when the request would exceed the bounded projector
// envelope.
type ArticleIndexResourceLimits struct {
	MaxArticleContentBytes int
	MaxChunksPerArticle    int
	MaxEmbeddingBatchSize  int
	MaxEmbeddingBatchBytes int
}

func defaultArticleIndexResourceLimits() ArticleIndexResourceLimits {
	return ArticleIndexResourceLimits{
		MaxArticleContentBytes: DefaultArticleIndexMaxArticleContentBytes,
		MaxChunksPerArticle:    DefaultArticleIndexMaxChunksPerArticle,
		MaxEmbeddingBatchSize:  DefaultArticleIndexMaxEmbeddingBatchSize,
		MaxEmbeddingBatchBytes: DefaultArticleIndexMaxEmbeddingBatchBytes,
	}
}

func normalizeArticleIndexResourceLimits(limits ArticleIndexResourceLimits) (ArticleIndexResourceLimits, error) {
	defaults := defaultArticleIndexResourceLimits()
	if limits == (ArticleIndexResourceLimits{}) {
		return defaults, nil
	}
	if limits.MaxArticleContentBytes < 0 ||
		limits.MaxChunksPerArticle < 0 ||
		limits.MaxEmbeddingBatchSize < 0 ||
		limits.MaxEmbeddingBatchBytes < 0 {
		return ArticleIndexResourceLimits{}, apperrors.Invalid("task.article_index_projector.resource_limits", "resource limits cannot be negative")
	}
	if limits.MaxArticleContentBytes == 0 {
		limits.MaxArticleContentBytes = defaults.MaxArticleContentBytes
	}
	if limits.MaxChunksPerArticle == 0 {
		limits.MaxChunksPerArticle = defaults.MaxChunksPerArticle
	}
	if limits.MaxEmbeddingBatchSize == 0 {
		limits.MaxEmbeddingBatchSize = defaults.MaxEmbeddingBatchSize
	}
	if limits.MaxEmbeddingBatchBytes == 0 {
		limits.MaxEmbeddingBatchBytes = defaults.MaxEmbeddingBatchBytes
	}
	return limits, nil
}

// ArticleIndexProjector contains the provider-neutral projection algorithm
// shared by lifecycle jobs and full backfills. Its target index is fixed at
// construction time, so a backfill cannot accidentally write to the old
// embedding contract.
type ArticleIndexProjector struct {
	router      port.ModelRouter
	index       port.ArticleChunkIndex
	spec        search.ArticleChunksIndexSpec
	chunker     search.MarkdownChunker
	projections port.ContentProjectionRepository
	limits      ArticleIndexResourceLimits
	now         func() time.Time
}

type ArticleIndexProjectorDeps struct {
	Router      port.ModelRouter
	Index       port.ArticleChunkIndex
	Spec        search.ArticleChunksIndexSpec
	Chunker     search.MarkdownChunker
	Projections port.ContentProjectionRepository
	Limits      ArticleIndexResourceLimits
	Now         func() time.Time
}

func NewArticleIndexProjector(deps ArticleIndexProjectorDeps) (*ArticleIndexProjector, error) {
	if deps.Router == nil {
		return nil, apperrors.Invalid("task.article_index_projector.dependencies", "model router is required")
	}
	if deps.Index == nil {
		return nil, apperrors.Invalid("task.article_index_projector.dependencies", "article chunk index is required")
	}
	if err := deps.Spec.Validate(); err != nil {
		return nil, err
	}
	limits, err := normalizeArticleIndexResourceLimits(deps.Limits)
	if err != nil {
		return nil, err
	}
	if deps.Spec.EmbeddingBatchSize > limits.MaxEmbeddingBatchSize {
		return nil, apperrors.Invalid("task.article_index_projector.resource_limits", "embedding batch size exceeds the projector memory budget")
	}
	return &ArticleIndexProjector{
		router:      deps.Router,
		index:       deps.Index,
		spec:        deps.Spec,
		chunker:     deps.Chunker,
		projections: deps.Projections,
		limits:      limits,
		now:         deps.Now,
	}, nil
}

// UpsertSource replaces the complete article projection and returns the
// number of chunks submitted to Meilisearch. Non-public sources are removed,
// which makes the method safe when a source changes state between pagination
// and processing.
func (p *ArticleIndexProjector) UpsertSource(ctx context.Context, source port.ArticleIndexSource) (int, error) {
	if p == nil {
		return 0, errors.New("article index projector is nil")
	}
	if source.Article.Id <= 0 {
		return 0, apperrors.Invalid("task.article_index_projector.upsert", "article id must be positive")
	}
	if !port.IsPublicArticle(source.Article.Status, source.Article.IsDelete) {
		return 0, p.DeleteArticle(ctx, source.Article.Id)
	}
	if len(source.Article.ArticleContent) > p.limits.MaxArticleContentBytes {
		return 0, apperrors.Invalid("task.article_index_projector.resource_limits", "article content exceeds the projector memory budget")
	}
	documents, err := search.BuildArticleChunkDocuments(articleForIndex(source.Article, source.CategoryName), source.Tags, p.chunker)
	if err != nil {
		return 0, fmt.Errorf("build article %d chunks: %w", source.Article.Id, err)
	}
	if len(documents) > p.limits.MaxChunksPerArticle {
		return 0, apperrors.Invalid("task.article_index_projector.resource_limits", "article chunk count exceeds the projector memory budget")
	}
	gateway, route, err := p.router.ResolveEmbedding(ctx, port.AIUseCaseEmbedding)
	if err != nil {
		return 0, fmt.Errorf("resolve embedding gateway: %w", err)
	}
	if route.Provider != p.spec.Provider || route.Model != p.spec.Model {
		return 0, apperrors.Invalid("task.article_index_projector.embedding_contract", "resolved embedding route does not match the index specification")
	}
	projectionReplaced := false
	indexedCount := 0
	aggregatedEmbedding := make([]float32, p.spec.Dimension)
	embeddingCount := 0
	for start := 0; start < len(documents); start += p.spec.EmbeddingBatchSize {
		end := start + p.spec.EmbeddingBatchSize
		if end > len(documents) {
			end = len(documents)
		}
		batch := documents[start:end]
		inputs := make([]string, len(batch))
		inputBytes := 0
		for index := range batch {
			inputs[index] = batch[index].Text
			if len(inputs[index]) > p.limits.MaxEmbeddingBatchBytes-inputBytes {
				return indexedCount, apperrors.Invalid("task.article_index_projector.resource_limits", "embedding batch input exceeds the projector memory budget")
			}
			inputBytes += len(inputs[index])
		}
		embeddings, err := gateway.Embed(ctx, port.EmbeddingRequest{
			Inputs:  inputs,
			Model:   route.Model,
			Version: p.spec.ModelVersion,
		})
		if err != nil {
			return indexedCount, fmt.Errorf("embed article %d batch %d: %w", source.Article.Id, start/p.spec.EmbeddingBatchSize, err)
		}
		if err := validateEmbeddingResult(embeddings, len(batch), p.spec); err != nil {
			return indexedCount, fmt.Errorf("validate article %d batch %d embedding: %w", source.Article.Id, start/p.spec.EmbeddingBatchSize, err)
		}
		indexed := make([]port.IndexedArticleChunk, len(batch))
		for index := range batch {
			for dimension, value := range embeddings.Vectors[index] {
				aggregatedEmbedding[dimension] += value
			}
			embeddingCount++
			indexed[index] = port.IndexedArticleChunk{
				Document:           batch[index],
				Embedding:          embeddings.Vectors[index],
				EmbeddingModel:     embeddings.Model,
				EmbeddingVersion:   embeddings.Version,
				EmbeddingDimension: embeddings.Dimension,
			}
		}
		if !projectionReplaced {
			// Stable chunk IDs make retries idempotent. Clearing first also
			// removes chunks left behind when an article becomes shorter.
			if err := p.index.DeleteArticle(ctx, source.Article.Id); err != nil {
				return indexedCount, fmt.Errorf("clear article %d old chunks: %w", source.Article.Id, err)
			}
			projectionReplaced = true
		}
		if err := p.index.UpsertChunks(ctx, indexed); err != nil {
			return indexedCount, fmt.Errorf("write article %d batch %d index: %w", source.Article.Id, start/p.spec.EmbeddingBatchSize, err)
		}
		indexedCount += len(indexed)
	}
	if p.projections != nil && embeddingCount > 0 {
		if err := p.saveContentProjection(ctx, source.Article, route.Model, p.spec.ModelVersion, aggregatedEmbedding, embeddingCount); err != nil {
			return indexedCount, err
		}
	}
	return indexedCount, nil
}

func (p *ArticleIndexProjector) DeleteArticle(ctx context.Context, articleID int) error {
	if p == nil {
		return errors.New("article index projector is nil")
	}
	if err := p.index.DeleteArticle(ctx, articleID); err != nil {
		return err
	}
	if p.projections != nil {
		now := p.currentTime()
		if err := p.projections.MarkDeleted(ctx, articleID, now); err != nil {
			return err
		}
	}
	return nil
}

func (p *ArticleIndexProjector) saveContentProjection(ctx context.Context, article port.TArticle, model, version string, embedding []float32, count int) error {
	if article.Id <= 0 || count <= 0 || len(embedding) == 0 {
		return apperrors.Invalid("task.article_index_projector.content_projection", "embedding projection source is invalid")
	}
	for index := range embedding {
		embedding[index] /= float32(count)
	}
	now := p.currentTime()
	lifeStage, err := port.CalculateContentLifeStage(port.ContentLifecycleSnapshot{
		PublishedAt: article.CreateTime,
		UpdatedAt:   article.UpdateTime,
		Now:         now,
	})
	if err != nil {
		return apperrors.Invalid("task.article_index_projector.content_projection", err.Error())
	}
	err = p.projections.Upsert(ctx, port.ContentProjection{
		ArticleID:          article.Id,
		LifeStage:          lifeStage,
		EmbeddingModel:     model,
		EmbeddingVersion:   version,
		EmbeddingDimension: len(embedding),
		PCAInput:           embedding,
		PCAInputVersion:    port.DefaultPCAInputVersion,
		Status:             port.ContentProjectionPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return fmt.Errorf("save content projection for article %d: %w", article.Id, err)
	}
	return nil
}

func (p *ArticleIndexProjector) currentTime() time.Time {
	if p != nil && p.now != nil {
		if now := p.now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}

func validateEmbeddingResult(result port.EmbeddingResult, expectedCount int, spec search.ArticleChunksIndexSpec) error {
	if len(result.Vectors) != expectedCount {
		return fmt.Errorf("provider returned %d vectors, want %d", len(result.Vectors), expectedCount)
	}
	if strings.TrimSpace(result.Model) != spec.Model {
		return errors.New("provider returned an unexpected embedding model")
	}
	if strings.TrimSpace(result.Version) != spec.ModelVersion {
		return errors.New("provider returned an unexpected embedding model version")
	}
	if result.Dimension != spec.Dimension {
		return fmt.Errorf("provider returned dimension %d, want %d", result.Dimension, spec.Dimension)
	}
	for index, vector := range result.Vectors {
		if err := spec.ValidateVector(vector); err != nil {
			return fmt.Errorf("vector %d: %w", index, err)
		}
	}
	return nil
}
