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

type fakeArticleIndexJobs struct {
	job         port.AIJob
	claimed     bool
	completed   []port.AIJobResult
	retried     []retryCall
	deadLetters []string
}

type retryCall struct {
	id        string
	runAfter  time.Time
	lastError string
}

func (f *fakeArticleIndexJobs) Enqueue(context.Context, port.AIJob) error { return nil }

func (f *fakeArticleIndexJobs) Claim(context.Context, string, time.Time, time.Duration) (port.AIJob, bool, error) {
	if f.claimed {
		return port.AIJob{}, false, nil
	}
	f.claimed = true
	job := f.job
	job.Status = port.AIJobRunning
	job.Attempts++
	return job, true, nil
}

func (f *fakeArticleIndexJobs) Complete(_ context.Context, _ string, _ string, result port.AIJobResult) error {
	f.completed = append(f.completed, result)
	return nil
}

func (f *fakeArticleIndexJobs) Retry(_ context.Context, id, _ string, runAfter time.Time, lastError string) error {
	f.retried = append(f.retried, retryCall{id: id, runAfter: runAfter, lastError: lastError})
	return nil
}

func (f *fakeArticleIndexJobs) DeadLetter(_ context.Context, id, _, lastError string) error {
	f.deadLetters = append(f.deadLetters, id+":"+lastError)
	return nil
}

type fakeEmbeddingGateway struct{}

func (fakeEmbeddingGateway) Embed(_ context.Context, request port.EmbeddingRequest) (port.EmbeddingResult, error) {
	vectors := make([][]float32, len(request.Inputs))
	for index := range request.Inputs {
		vectors[index] = []float32{float32(index + 1), 0.2, 0.3}
	}
	return port.EmbeddingResult{
		Vectors:   vectors,
		Model:     request.Model,
		Dimension: 3,
		Version:   request.Version,
	}, nil
}

type fakeEmbeddingRouter struct {
	gateway port.EmbeddingGateway
	route   port.ModelRoute
}

func (f fakeEmbeddingRouter) ResolveChat(context.Context, port.AIUseCase) (port.ChatGateway, port.ModelRoute, error) {
	return nil, port.ModelRoute{}, errors.New("chat is not used by article index worker")
}

func (f fakeEmbeddingRouter) ResolveEmbedding(context.Context, port.AIUseCase) (port.EmbeddingGateway, port.ModelRoute, error) {
	return f.gateway, f.route, nil
}

type fakeArticleChunkIndex struct {
	events    []string
	deleted   []int
	upserted  [][]port.IndexedArticleChunk
	deleteErr error
	upsertErr error
}

func (f *fakeArticleChunkIndex) UpsertChunks(_ context.Context, chunks []port.IndexedArticleChunk) error {
	f.events = append(f.events, "upsert")
	if f.upsertErr != nil {
		return f.upsertErr
	}
	copyOfChunks := make([]port.IndexedArticleChunk, len(chunks))
	copy(copyOfChunks, chunks)
	f.upserted = append(f.upserted, copyOfChunks)
	return nil
}

func (f *fakeArticleChunkIndex) DeleteArticle(_ context.Context, articleID int) error {
	f.events = append(f.events, "delete")
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, articleID)
	return nil
}

type fakeArticleRepositoryForIndex struct {
	article  entity.TArticle
	category string
	tags     []string
	err      error
}

func (f *fakeArticleRepositoryForIndex) ListTopAndFeaturedArticles(context.Context) ([]*port.ArticleCard, error) {
	return nil, nil
}
func (f *fakeArticleRepositoryForIndex) ListArticles(context.Context, int, int) ([]*port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepositoryForIndex) GetArticlesByCategoryID(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepositoryForIndex) GetArticleByID(context.Context, int) (port.Article, error) {
	return port.Article{}, nil
}
func (f *fakeArticleRepositoryForIndex) GetPreArticleByID(context.Context, int) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepositoryForIndex) GetNextArticleByID(context.Context, int) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepositoryForIndex) GetFirstArticle(context.Context) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepositoryForIndex) GetLastArticle(context.Context) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepositoryForIndex) ListArticlesByTagID(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepositoryForIndex) ListArchives(context.Context, int, int) ([]port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepositoryForIndex) CountArticleAdmins(context.Context, port.ArticleFilter) (int, error) {
	return 0, nil
}
func (f *fakeArticleRepositoryForIndex) ListArticlesAdmin(context.Context, port.ArticleFilter) ([]*port.ArticleAdmin, error) {
	return nil, nil
}
func (f *fakeArticleRepositoryForIndex) ListArticleStatistics(context.Context) ([]port.ArticleStatistics, error) {
	return nil, nil
}
func (f *fakeArticleRepositoryForIndex) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepositoryForIndex) SaveOrUpdate(context.Context, entity.TArticle, string, []string) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepositoryForIndex) UpdateTopAndFeatured(context.Context, int, int, int) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepositoryForIndex) UpdateDelete(context.Context, []int, int) error { return nil }
func (f *fakeArticleRepositoryForIndex) Delete(context.Context, []int) error            { return nil }
func (f *fakeArticleRepositoryForIndex) GetAdminArticle(context.Context, int) (entity.TArticle, string, []string, error) {
	if f.err != nil {
		return entity.TArticle{}, "", nil, f.err
	}
	return f.article, f.category, append([]string(nil), f.tags...), nil
}
func (f *fakeArticleRepositoryForIndex) Export(context.Context, []int) ([]entity.TArticle, error) {
	return nil, nil
}

func articleIndexTestSpec() search.ArticleChunksIndexSpec {
	return search.ArticleChunksIndexSpec{
		UID:                "article_chunks_v1",
		PrimaryKey:         "id",
		Provider:           "openai",
		Model:              "text-embedding-3-small",
		ModelVersion:       "2026-08-28",
		IndexVersion:       "v1",
		Dimension:          3,
		EmbeddingBatchSize: 2,
	}
}

func newArticleIndexWorkerForTest(t *testing.T, jobs *fakeArticleIndexJobs, index *fakeArticleChunkIndex) *ArticleIndexWorker {
	t.Helper()
	chunker, err := search.NewMarkdownChunker(search.MarkdownChunkerConfig{ChunkSize: 8, OverlapSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewArticleIndexWorker(ArticleIndexWorkerDeps{
		Jobs: jobs,
		Articles: &fakeArticleRepositoryForIndex{
			article: entity.TArticle{
				Id:             42,
				ArticleTitle:   "Agent notes",
				ArticleContent: "one two three four five six seven eight nine ten",
				Status:         port.PublicArticleStatus,
				CreateTime:     time.Date(2026, 8, 28, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
			},
			category: "engineering",
			tags:     []string{"go", "agent", "go"},
		},
		Router:    fakeEmbeddingRouter{gateway: fakeEmbeddingGateway{}, route: port.ModelRoute{UseCase: port.AIUseCaseEmbedding, Provider: "openai", Protocol: port.ProviderProtocolOpenAIChatCompletions, Model: "text-embedding-3-small"}},
		Index:     index,
		Spec:      articleIndexTestSpec(),
		Chunker:   chunker,
		WorkerID:  "test-worker",
		RetryBase: time.Second,
	})
	if err != nil {
		t.Fatalf("NewArticleIndexWorker() error = %v", err)
	}
	return worker
}

func articleIndexJobPayload(t *testing.T, action port.ArticleIndexAction) []byte {
	t.Helper()
	payload, err := port.NewArticleIndexJobPayload(42, action, port.ArticleUpdated, port.PublicArticleStatus, 0, time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestArticleIndexWorkerBatchesEmbeddingAndReplacesStableChunks(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-1", Kind: port.AIJobKindArticleIndex, Payload: articleIndexJobPayload(t, port.ArticleIndexUpsert), MaxAttempts: 3}}
	index := &fakeArticleChunkIndex{}
	worker := newArticleIndexWorkerForTest(t, jobs, index)

	processed, err := worker.processOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("processOne() = processed %v error %v", processed, err)
	}
	if len(jobs.completed) != 1 || len(jobs.retried) != 0 || len(jobs.deadLetters) != 0 {
		t.Fatalf("job transitions: completed=%d retried=%d dead=%d", len(jobs.completed), len(jobs.retried), len(jobs.deadLetters))
	}
	if len(index.deleted) != 1 || index.deleted[0] != 42 || len(index.upserted) == 0 {
		t.Fatalf("index replacement: deleted=%v upserted=%d", index.deleted, len(index.upserted))
	}
	if index.events[0] != "delete" {
		t.Fatalf("index events = %v, want delete before upsert", index.events)
	}
	seenIDs := make(map[string]struct{})
	for _, batch := range index.upserted {
		if len(batch) > 2 {
			t.Fatalf("embedding batch produced %d indexed chunks, want <= 2", len(batch))
		}
		for _, indexed := range batch {
			if indexed.Document.ArticleID != 42 || indexed.EmbeddingModel != articleIndexTestSpec().Model || indexed.EmbeddingVersion != articleIndexTestSpec().ModelVersion {
				t.Fatalf("unexpected indexed chunk: %+v", indexed)
			}
			if _, exists := seenIDs[indexed.Document.ID]; exists {
				t.Fatalf("duplicate stable chunk id: %s", indexed.Document.ID)
			}
			seenIDs[indexed.Document.ID] = struct{}{}
		}
	}
	if len(seenIDs) < 2 {
		t.Fatalf("indexed chunk count = %d, want multiple chunks", len(seenIDs))
	}
}

func TestArticleIndexWorkerDeletesPrivateArticleWithoutEmbedding(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-2", Kind: port.AIJobKindArticleIndex, Payload: articleIndexJobPayload(t, port.ArticleIndexDelete), MaxAttempts: 3}}
	index := &fakeArticleChunkIndex{}
	worker := newArticleIndexWorkerForTest(t, jobs, index)

	if _, err := worker.processOne(context.Background()); err != nil {
		t.Fatalf("processOne() error = %v", err)
	}
	if len(index.deleted) != 1 || len(index.upserted) != 0 || len(jobs.completed) != 1 {
		t.Fatalf("delete job: deleted=%v upserted=%d completed=%d", index.deleted, len(index.upserted), len(jobs.completed))
	}
}

func TestArticleIndexWorkerDoesNotProcessTheSameClaimTwice(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID:          "job-once",
		Kind:        port.AIJobKindArticleIndex,
		Payload:     articleIndexJobPayload(t, port.ArticleIndexDelete),
		MaxAttempts: 3,
	}}
	index := &fakeArticleChunkIndex{}
	worker := newArticleIndexWorkerForTest(t, jobs, index)

	processed, err := worker.RunOnce(context.Background())
	if err != nil || !processed {
		t.Fatalf("first RunOnce() = processed %v error %v", processed, err)
	}
	processed, err = worker.RunOnce(context.Background())
	if err != nil || processed {
		t.Fatalf("second RunOnce() = processed %v error %v, want no second claim", processed, err)
	}
	if len(jobs.completed) != 1 || len(index.deleted) != 1 {
		t.Fatalf("same job was processed more than once: completed=%d deleted=%d", len(jobs.completed), len(index.deleted))
	}
}

func TestArticleIndexWorkerRechecksVisibilityForStaleUpsertJobs(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		deleted int
		err     error
	}{
		{name: "private", status: 2},
		{name: "soft deleted", status: port.PublicArticleStatus, deleted: 1},
		{name: "hard deleted", err: apperrors.NotFound("article.get_admin")},
	} {
		t.Run(test.name, func(t *testing.T) {
			jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-stale", Kind: port.AIJobKindArticleIndex, Payload: articleIndexJobPayload(t, port.ArticleIndexUpsert), MaxAttempts: 3}}
			index := &fakeArticleChunkIndex{}
			worker := newArticleIndexWorkerForTest(t, jobs, index)
			repository := worker.articles.(*fakeArticleRepositoryForIndex)
			repository.article.Status = test.status
			repository.article.IsDelete = test.deleted
			repository.err = test.err

			processed, err := worker.processOne(context.Background())
			if err != nil || !processed {
				t.Fatalf("processOne() = processed %v error %v", processed, err)
			}
			if len(index.deleted) != 1 || index.deleted[0] != 42 || len(index.upserted) != 0 || len(jobs.completed) != 1 {
				t.Fatalf("stale upsert handling: deleted=%v upserted=%d completed=%d", index.deleted, len(index.upserted), len(jobs.completed))
			}
		})
	}
}

func TestArticleIndexWorkerRetriesTransientIndexFailureAndDeadLettersAfterMaximum(t *testing.T) {
	for _, test := range []struct {
		name      string
		attempts  int
		wantRetry bool
		wantDead  bool
	}{
		{name: "retry", attempts: 1, wantRetry: true},
		{name: "dead letter", attempts: 3, wantDead: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-failure", Kind: port.AIJobKindArticleIndex, Payload: articleIndexJobPayload(t, port.ArticleIndexDelete), Attempts: test.attempts - 1, MaxAttempts: 3}}
			index := &fakeArticleChunkIndex{deleteErr: errors.New("meilisearch unavailable")}
			worker := newArticleIndexWorkerForTest(t, jobs, index)

			processed, err := worker.processOne(context.Background())
			if err != nil || !processed {
				t.Fatalf("processOne() = processed %v error %v", processed, err)
			}
			if (len(jobs.retried) > 0) != test.wantRetry {
				t.Fatalf("retry transitions = %v, want retry=%v", jobs.retried, test.wantRetry)
			}
			if (len(jobs.deadLetters) > 0) != test.wantDead {
				t.Fatalf("dead-letter transitions = %v, want dead=%v", jobs.deadLetters, test.wantDead)
			}
			if len(jobs.completed) != 0 {
				t.Fatal("failed job must not be completed")
			}
			if len(jobs.retried) > 0 && jobs.retried[0].lastError != "internal_error" {
				t.Fatalf("retry error = %q, provider detail should be sanitized", jobs.retried[0].lastError)
			}
		})
	}
}

func TestArticleIndexWorkerConstructorValidatesAllDependencies(t *testing.T) {
	_, err := NewArticleIndexWorker(ArticleIndexWorkerDeps{})
	if !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("constructor error kind = %v, want validation", apperrors.KindOf(err))
	}
}
