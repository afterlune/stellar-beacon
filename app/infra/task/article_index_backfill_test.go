package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/search"
)

type fakeArticleIndexSourceRepository struct {
	sources []port.ArticleIndexSource
	calls   []int
	err     error
}

func (f *fakeArticleIndexSourceRepository) ListPublicArticleIndexSources(_ context.Context, afterArticleID, limit int) ([]port.ArticleIndexSource, error) {
	f.calls = append(f.calls, afterArticleID)
	if f.err != nil {
		return nil, f.err
	}
	result := make([]port.ArticleIndexSource, 0, limit)
	for _, source := range f.sources {
		if source.Article.Id <= afterArticleID {
			continue
		}
		result = append(result, source)
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func newArticleIndexBackfillForTest(t *testing.T, source port.ArticleIndexSourceRepository, index port.ArticleChunkIndex, pageSize int) *ArticleIndexBackfill {
	t.Helper()
	chunker, err := search.NewMarkdownChunker(search.MarkdownChunkerConfig{ChunkSize: 8, OverlapSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router:  fakeEmbeddingRouter{gateway: fakeEmbeddingGateway{}, route: port.ModelRoute{UseCase: port.AIUseCaseEmbedding, Provider: "openai", Protocol: port.ProviderProtocolOpenAIChatCompletions, Model: "text-embedding-3-small"}},
		Index:   index,
		Spec:    articleIndexTestSpec(),
		Chunker: chunker,
	})
	if err != nil {
		t.Fatalf("NewArticleIndexProjector() error = %v", err)
	}
	backfill, err := NewArticleIndexBackfill(ArticleIndexBackfillDeps{
		Sources:   source,
		Projector: projector,
		PageSize:  pageSize,
	})
	if err != nil {
		t.Fatalf("NewArticleIndexBackfill() error = %v", err)
	}
	return backfill
}

func articleIndexSource(id int) port.ArticleIndexSource {
	return port.ArticleIndexSource{
		Article: entity.TArticle{
			Id:             id,
			ArticleTitle:   "Article " + string(rune('0'+id)),
			ArticleContent: "one two three four five six seven eight nine",
			Status:         port.PublicArticleStatus,
			CreateTime:     time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		},
		CategoryName: "engineering",
		Tags:         []string{"go", "agent"},
	}
}

func TestArticleIndexBackfillUsesKeysetCursorAndTargetIndex(t *testing.T) {
	sources := &fakeArticleIndexSourceRepository{sources: []port.ArticleIndexSource{
		articleIndexSource(1),
		articleIndexSource(2),
		articleIndexSource(3),
	}}
	index := &fakeArticleChunkIndex{}
	backfill := newArticleIndexBackfillForTest(t, sources, index, 2)

	result, err := backfill.Run(context.Background(), 0)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Index != articleIndexTestSpec().UID || result.ProcessedArticles != 3 || result.LastArticleID != 3 || result.IndexedChunks < 3 {
		t.Fatalf("unexpected backfill result: %+v", result)
	}
	if len(sources.calls) != 2 || sources.calls[0] != 0 || sources.calls[1] != 2 {
		t.Fatalf("source cursors = %v, want [0 2]", sources.calls)
	}
	if len(index.deleted) != 3 {
		t.Fatalf("deleted articles = %v, want one replacement delete per article", index.deleted)
	}

	sources.calls = nil
	result, err = backfill.Run(context.Background(), 2)
	if err != nil {
		t.Fatalf("resumed Run() error = %v", err)
	}
	if result.ProcessedArticles != 1 || result.LastArticleID != 3 || len(sources.calls) != 1 || sources.calls[0] != 2 {
		t.Fatalf("unexpected resumed backfill: result=%+v calls=%v", result, sources.calls)
	}
}

func TestArticleIndexBackfillStopsAfterPerRunLimitAtCheckpoint(t *testing.T) {
	sources := &fakeArticleIndexSourceRepository{sources: []port.ArticleIndexSource{
		articleIndexSource(1),
		articleIndexSource(2),
		articleIndexSource(3),
	}}
	index := &fakeArticleChunkIndex{}
	backfill := newArticleIndexBackfillForTest(t, sources, index, 2)

	result, err := backfill.RunWithHooks(context.Background(), 0, ArticleIndexBackfillHooks{
		MaxArticles: 1,
	})
	if !errors.Is(err, ErrArticleIndexBackfillPaused) {
		t.Fatalf("RunWithHooks() error = %v, want cooperative pause", err)
	}
	if result.ProcessedArticles != 1 || result.LastArticleID != 1 {
		t.Fatalf("bounded result = %+v, want one checkpointed article", result)
	}
	if len(index.deleted) != 1 {
		t.Fatalf("deleted articles = %v, want only the checkpointed article", index.deleted)
	}
	if len(sources.calls) != 1 || sources.calls[0] != 0 {
		t.Fatalf("source cursors = %v, want one page read from cursor 0", sources.calls)
	}
}

func TestArticleIndexBackfillRejectsNegativePerRunLimit(t *testing.T) {
	backfill := newArticleIndexBackfillForTest(t, &fakeArticleIndexSourceRepository{}, &fakeArticleChunkIndex{}, 1)

	_, err := backfill.RunWithHooks(context.Background(), 0, ArticleIndexBackfillHooks{MaxArticles: -1})
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("RunWithHooks() error kind = %v, want validation", apperrors.KindOf(err))
	}
}

func TestArticleIndexBackfillDoesNotAdvanceCursorAfterProjectionFailure(t *testing.T) {
	sources := &fakeArticleIndexSourceRepository{sources: []port.ArticleIndexSource{articleIndexSource(1)}}
	index := &fakeArticleChunkIndex{upsertErr: errors.New("index unavailable")}
	backfill := newArticleIndexBackfillForTest(t, sources, index, 10)

	result, err := backfill.Run(context.Background(), 0)
	if err == nil || result.ProcessedArticles != 0 || result.LastArticleID != 0 {
		t.Fatalf("Run() = result=%+v error=%v, want failure before cursor advance", result, err)
	}
}

func TestArticleIndexBackfillStopsBeforeReadingWhenCanceled(t *testing.T) {
	sources := &fakeArticleIndexSourceRepository{sources: []port.ArticleIndexSource{articleIndexSource(1)}}
	index := &fakeArticleChunkIndex{}
	backfill := newArticleIndexBackfillForTest(t, sources, index, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := backfill.Run(ctx, 0)
	if !errors.Is(err, context.Canceled) || result.ProcessedArticles != 0 || len(sources.calls) != 0 {
		t.Fatalf("Run() = result=%+v error=%v calls=%v, want immediate cancellation", result, err, sources.calls)
	}
}

func TestArticleIndexProjectorDeletesNonPublicSourceWithoutEmbedding(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		deleted int
	}{
		{name: "private", status: 2},
		{name: "deleted", status: port.PublicArticleStatus, deleted: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			index := &fakeArticleChunkIndex{}
			backfill := newArticleIndexBackfillForTest(t, &fakeArticleIndexSourceRepository{}, index, 1)
			source := articleIndexSource(42)
			source.Article.Status = test.status
			source.Article.IsDelete = test.deleted

			indexed, err := backfill.projector.UpsertSource(context.Background(), source)
			if err != nil || indexed != 0 {
				t.Fatalf("UpsertSource() = indexed %d error %v, want delete-only", indexed, err)
			}
			if len(index.deleted) != 1 || index.deleted[0] != 42 || len(index.upserted) != 0 {
				t.Fatalf("non-public projection: deleted=%v upserted=%d", index.deleted, len(index.upserted))
			}
		})
	}
}

func TestNewArticleIndexBackfillValidatesDependenciesAndPageSize(t *testing.T) {
	if _, err := NewArticleIndexBackfill(ArticleIndexBackfillDeps{}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("missing dependencies error kind = %v, want validation", apperrors.KindOf(err))
	}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router: fakeEmbeddingRouter{gateway: fakeEmbeddingGateway{}, route: port.ModelRoute{Provider: "openai", Model: "text-embedding-3-small"}},
		Index:  &fakeArticleChunkIndex{},
		Spec:   articleIndexTestSpec(),
	})
	if err != nil {
		t.Fatal(err)
	}
	source := &fakeArticleIndexSourceRepository{}
	for _, pageSize := range []int{-1, maxArticleIndexBackfillPageSize + 1} {
		if _, err := NewArticleIndexBackfill(ArticleIndexBackfillDeps{Sources: source, Projector: projector, PageSize: pageSize}); !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Fatalf("page size %d error kind = %v, want validation", pageSize, apperrors.KindOf(err))
		}
	}
	if _, err := NewArticleIndexBackfill(ArticleIndexBackfillDeps{Sources: source, Projector: projector, PageSize: 1}); err != nil {
		t.Fatalf("valid page size error = %v", err)
	}
}

type countingArticleIndexEmbeddingGateway struct {
	calls *int
}

func (g countingArticleIndexEmbeddingGateway) Embed(ctx context.Context, request port.EmbeddingRequest) (port.EmbeddingResult, error) {
	(*g.calls)++
	return fakeEmbeddingGateway{}.Embed(ctx, request)
}

func TestArticleIndexProjectorRejectsOversizedArticleBeforeProviderOrIndex(t *testing.T) {
	calls := 0
	index := &fakeArticleChunkIndex{}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router: fakeEmbeddingRouter{
			gateway: countingArticleIndexEmbeddingGateway{calls: &calls},
			route:   port.ModelRoute{Provider: "openai", Model: "text-embedding-3-small"},
		},
		Index: index,
		Spec:  articleIndexTestSpec(),
	})
	if err != nil {
		t.Fatal(err)
	}
	source := articleIndexSource(1)
	source.Article.ArticleContent = strings.Repeat("x", DefaultArticleIndexMaxArticleContentBytes+1)

	if _, err := projector.UpsertSource(context.Background(), source); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("UpsertSource() error kind = %v, want validation", apperrors.KindOf(err))
	}
	if calls != 0 || len(index.events) != 0 {
		t.Fatalf("oversized article caused side effects: provider calls=%d index events=%v", calls, index.events)
	}
}

func TestArticleIndexProjectorRejectsChunkCountBeforeProviderOrIndex(t *testing.T) {
	chunker, err := search.NewMarkdownChunker(search.MarkdownChunkerConfig{ChunkSize: 4, OverlapSize: 0})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	index := &fakeArticleChunkIndex{}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router: fakeEmbeddingRouter{
			gateway: countingArticleIndexEmbeddingGateway{calls: &calls},
			route:   port.ModelRoute{Provider: "openai", Model: "text-embedding-3-small"},
		},
		Index:   index,
		Spec:    articleIndexTestSpec(),
		Chunker: chunker,
		Limits:  ArticleIndexResourceLimits{MaxChunksPerArticle: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	source := articleIndexSource(1)
	if _, err := projector.UpsertSource(context.Background(), source); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("UpsertSource() error kind = %v, want validation", apperrors.KindOf(err))
	}
	if calls != 0 || len(index.events) != 0 {
		t.Fatalf("oversized chunk set caused side effects: provider calls=%d index events=%v", calls, index.events)
	}
}

func TestNewArticleIndexProjectorRejectsBatchAboveMemoryBudget(t *testing.T) {
	spec := articleIndexTestSpec()
	spec.EmbeddingBatchSize = 3
	_, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router: fakeEmbeddingRouter{gateway: fakeEmbeddingGateway{}, route: port.ModelRoute{Provider: "openai", Model: "text-embedding-3-small"}},
		Index:  &fakeArticleChunkIndex{},
		Spec:   spec,
		Limits: ArticleIndexResourceLimits{MaxEmbeddingBatchSize: 2},
	})
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("NewArticleIndexProjector() error kind = %v, want validation", apperrors.KindOf(err))
	}
}

func TestArticleIndexProjectorRejectsEmbeddingInputBatchAboveMemoryBudget(t *testing.T) {
	calls := 0
	index := &fakeArticleChunkIndex{}
	projector, err := NewArticleIndexProjector(ArticleIndexProjectorDeps{
		Router: fakeEmbeddingRouter{
			gateway: countingArticleIndexEmbeddingGateway{calls: &calls},
			route:   port.ModelRoute{Provider: "openai", Model: "text-embedding-3-small"},
		},
		Index:  index,
		Spec:   articleIndexTestSpec(),
		Limits: ArticleIndexResourceLimits{MaxEmbeddingBatchBytes: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projector.UpsertSource(context.Background(), articleIndexSource(1)); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("UpsertSource() error kind = %v, want validation", apperrors.KindOf(err))
	}
	if calls != 0 || len(index.events) != 0 {
		t.Fatalf("oversized embedding input caused side effects: provider calls=%d index events=%v", calls, index.events)
	}
}
