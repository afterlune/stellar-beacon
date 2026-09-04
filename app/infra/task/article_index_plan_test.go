package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/search"
)

type articleIndexPlanSourceRepository struct {
	sources []port.ArticleIndexSource
	pages   []int
}

func (f *articleIndexPlanSourceRepository) ListPublicArticleIndexSources(_ context.Context, afterArticleID, limit int) ([]port.ArticleIndexSource, error) {
	start := 0
	for start < len(f.sources) && f.sources[start].Article.Id <= afterArticleID {
		start++
	}
	end := start + limit
	if end > len(f.sources) {
		end = len(f.sources)
	}
	f.pages = append(f.pages, end-start)
	return append([]port.ArticleIndexSource(nil), f.sources[start:end]...), nil
}

func planArticle(id int, content string) port.ArticleIndexSource {
	return port.ArticleIndexSource{
		Article: entity.TArticle{
			Id:             id,
			ArticleTitle:   "article",
			ArticleContent: content,
			Status:         port.PublicArticleStatus,
			IsDelete:       0,
			CreateTime:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func newArticleIndexPlanForTest(t *testing.T, source port.ArticleIndexSourceRepository, pageSize int) *ArticleIndexPlan {
	t.Helper()
	chunker, err := search.NewMarkdownChunker(search.MarkdownChunkerConfig{ChunkSize: 8, OverlapSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	planner, err := NewArticleIndexPlan(ArticleIndexPlanDeps{
		Sources:  source,
		Chunker:  chunker,
		PageSize: pageSize,
		Index:    "article_chunks_v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return planner
}

func TestArticleIndexPlanCountsChunksWithoutProjectionDependencies(t *testing.T) {
	source := &articleIndexPlanSourceRepository{sources: []port.ArticleIndexSource{
		planArticle(2, "one two three four five six"),
		planArticle(7, "   "),
		planArticle(11, "another article with enough text"),
	}}
	planner := newArticleIndexPlanForTest(t, source, 2)

	result, err := planner.Run(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Complete || result.Limited || result.ReadyForBackfill {
		t.Fatalf("unexpected completion state: %+v", result)
	}
	if result.ScannedArticles != 3 || result.InvalidArticles != 1 || result.LastArticleID != 11 {
		t.Fatalf("unexpected article counts: %+v", result)
	}
	if result.EstimatedChunks < 2 || len(result.InvalidArticleIDs) != 1 || result.InvalidArticleIDs[0] != 7 {
		t.Fatalf("unexpected chunk/invalid report: %+v", result)
	}
	if len(source.pages) < 2 {
		t.Fatalf("expected bounded pages, got %v", source.pages)
	}
}

func TestArticleIndexPlanMarksCompleteReadyWhenAllSourcesChunk(t *testing.T) {
	source := &articleIndexPlanSourceRepository{sources: []port.ArticleIndexSource{
		planArticle(3, "first article"),
		planArticle(8, "second article"),
	}}
	planner := newArticleIndexPlanForTest(t, source, 10)
	result, err := planner.Run(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Complete || !result.ReadyForBackfill || result.InvalidArticles != 0 {
		t.Fatalf("unexpected ready result: %+v", result)
	}
}

func TestArticleIndexPlanBoundedSampleIsNotComplete(t *testing.T) {
	source := &articleIndexPlanSourceRepository{sources: []port.ArticleIndexSource{
		planArticle(1, "first"),
		planArticle(2, "second"),
		planArticle(3, "third"),
	}}
	planner := newArticleIndexPlanForTest(t, source, 2)
	result, err := planner.Run(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ScannedArticles != 2 || !result.Limited || result.Complete || result.ReadyForBackfill || result.LastArticleID != 2 {
		t.Fatalf("unexpected bounded result: %+v", result)
	}
}

func TestNewArticleIndexPlanValidatesPageSize(t *testing.T) {
	source := &articleIndexPlanSourceRepository{}
	for _, pageSize := range []int{-1, 26} {
		if _, err := NewArticleIndexPlan(ArticleIndexPlanDeps{Sources: source, PageSize: pageSize}); !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Fatalf("page size %d error = %v, want validation", pageSize, err)
		}
	}
}

func TestArticleIndexPlanRejectsNonIncreasingSourceIDs(t *testing.T) {
	source := &articleIndexPlanSourceRepository{sources: []port.ArticleIndexSource{
		planArticle(2, "first"),
		planArticle(2, "duplicate"),
	}}
	// The fake repository filters by cursor and therefore cannot produce a
	// duplicate page; use a deliberately malformed repository for this guard.
	malformed := articleIndexPlanSourceRepositoryWithFixedPage{sources: source.sources}
	planner := newArticleIndexPlanForTest(t, malformed, 10)
	_, err := planner.Run(context.Background(), 0, 0)
	if err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("duplicate source IDs error = %v, want validation", err)
	}
}

type articleIndexPlanSourceRepositoryWithFixedPage struct {
	sources []port.ArticleIndexSource
}

func (f articleIndexPlanSourceRepositoryWithFixedPage) ListPublicArticleIndexSources(context.Context, int, int) ([]port.ArticleIndexSource, error) {
	return f.sources, nil
}

func TestArticleIndexPlanHonorsCancellation(t *testing.T) {
	source := &articleIndexPlanSourceRepository{sources: []port.ArticleIndexSource{planArticle(1, "text")}}
	planner := newArticleIndexPlanForTest(t, source, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := planner.Run(ctx, 0, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Run() error = %v, want context.Canceled", err)
	}
}
