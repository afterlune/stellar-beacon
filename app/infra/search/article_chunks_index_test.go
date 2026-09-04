package search

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"

	"github.com/meilisearch/meilisearch-go"
)

func TestArticleChunksIndexUIDIsVersionedAndSafe(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		valid   bool
	}{
		{name: "v1", version: "v1", want: "article_chunks_v1", valid: true},
		{name: "canonicalizes case", version: "V2", want: "article_chunks_v2", valid: true},
		{name: "date version", version: "2026-08-28_a", want: "article_chunks_2026-08-28_a", valid: true},
		{name: "empty", version: "", valid: false},
		{name: "path separator", version: "v1/next", valid: false},
		{name: "leading separator", version: "_v1", valid: false},
		{name: "too long", version: "abcdefghijklmnopqrstuvwxyz123456", valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ArticleChunksIndexUID(tt.version)
			if tt.valid {
				if err != nil || got != tt.want {
					t.Fatalf("ArticleChunksIndexUID(%q) = %q, %v; want %q", tt.version, got, err, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ArticleChunksIndexUID(%q) error = nil, want validation error", tt.version)
			}
		})
	}
}

func TestNewArticleChunksIndexSpecRequiresExplicitEmbedderContract(t *testing.T) {
	route := config.AIModelRoute{Provider: "openai", Model: "text-embedding-3-small"}
	embedding := config.AIEmbeddingConfig{
		IndexVersion: "v1",
		ModelVersion: "2024-01-01",
		Dimension:    1536,
		BatchSize:    32,
	}
	spec, err := NewArticleChunksIndexSpec(route, embedding)
	if err != nil {
		t.Fatalf("NewArticleChunksIndexSpec() error = %v", err)
	}
	if spec.UID != "article_chunks_v1" || spec.PrimaryKey != "id" || spec.Provider != "openai" || spec.Model != "text-embedding-3-small" || spec.Dimension != 1536 {
		t.Fatalf("unexpected index spec: %+v", spec)
	}

	tests := []struct {
		name  string
		setup func(*config.AIModelRoute, *config.AIEmbeddingConfig)
	}{
		{name: "provider", setup: func(route *config.AIModelRoute, _ *config.AIEmbeddingConfig) { route.Provider = "" }},
		{name: "model", setup: func(route *config.AIModelRoute, _ *config.AIEmbeddingConfig) { route.Model = "" }},
		{name: "model version", setup: func(_ *config.AIModelRoute, embedding *config.AIEmbeddingConfig) { embedding.ModelVersion = "" }},
		{name: "dimension", setup: func(_ *config.AIModelRoute, embedding *config.AIEmbeddingConfig) { embedding.Dimension = 0 }},
		{name: "batch size", setup: func(_ *config.AIModelRoute, embedding *config.AIEmbeddingConfig) { embedding.BatchSize = 0 }},
		{name: "index version", setup: func(_ *config.AIModelRoute, embedding *config.AIEmbeddingConfig) {
			embedding.IndexVersion = "bad/version"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidateRoute := route
			candidateEmbedding := embedding
			tt.setup(&candidateRoute, &candidateEmbedding)
			if _, err := NewArticleChunksIndexSpec(candidateRoute, candidateEmbedding); err == nil {
				t.Fatal("NewArticleChunksIndexSpec() error = nil, want validation error")
			}
		})
	}
}

func TestValidateArticleChunksIndexMigrationRequiresNewUIDForContractChanges(t *testing.T) {
	previous := testArticleChunksSpec()
	tests := []struct {
		name      string
		mutate    func(*ArticleChunksIndexSpec)
		wantError bool
		wantKind  apperrors.Kind
	}{
		{name: "same contract may reuse spec", mutate: func(target *ArticleChunksIndexSpec) {}, wantError: false},
		{name: "model", mutate: func(target *ArticleChunksIndexSpec) { target.Model = "text-embedding-3-large" }, wantError: true, wantKind: apperrors.KindConflict},
		{name: "model version", mutate: func(target *ArticleChunksIndexSpec) { target.ModelVersion = "2026-08-28" }, wantError: true, wantKind: apperrors.KindConflict},
		{name: "dimension", mutate: func(target *ArticleChunksIndexSpec) { target.Dimension = 3072 }, wantError: true, wantKind: apperrors.KindConflict},
		{name: "provider", mutate: func(target *ArticleChunksIndexSpec) { target.Provider = "sglang" }, wantError: true, wantKind: apperrors.KindConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := previous
			tt.mutate(&target)
			if tt.wantError {
				err := ValidateArticleChunksIndexMigration(previous, target)
				if !apperrors.IsKind(err, tt.wantKind) {
					t.Fatalf("ValidateArticleChunksIndexMigration() error kind = %v, want %v", apperrors.KindOf(err), tt.wantKind)
				}
				return
			}
			if err := ValidateArticleChunksIndexMigration(previous, target); err != nil {
				t.Fatalf("ValidateArticleChunksIndexMigration() error = %v", err)
			}
		})
	}

	target := previous
	target.IndexVersion = "v2"
	target.UID = "article_chunks_v2"
	target.ModelVersion = "2026-08-28"
	target.Dimension = 3072
	if err := ValidateArticleChunksIndexMigration(previous, target); err != nil {
		t.Fatalf("new versioned target should be accepted: %v", err)
	}
}

func TestArticleChunksIndexSpecValidatesVectorShapeAndValues(t *testing.T) {
	spec := ArticleChunksIndexSpec{Dimension: 2}
	if err := spec.ValidateVector([]float32{0.1, -0.2}); err != nil {
		t.Fatalf("ValidateVector() error = %v", err)
	}
	for _, vector := range [][]float32{
		{0.1},
		{0.1, 0.2, 0.3},
		{float32(math.NaN()), 0.2},
		{float32(math.Inf(1)), 0.2},
	} {
		if err := spec.ValidateVector(vector); err == nil {
			t.Fatalf("ValidateVector(%v) error = nil, want validation error", vector)
		}
	}
}

func testArticleChunksSpec() ArticleChunksIndexSpec {
	return ArticleChunksIndexSpec{
		UID:                "article_chunks_v1",
		PrimaryKey:         "id",
		Provider:           "openai",
		Model:              "text-embedding-3-small",
		ModelVersion:       "2024-01-01",
		IndexVersion:       "v1",
		Dimension:          1536,
		EmbeddingBatchSize: 32,
	}
}

func TestEnsureArticleChunksIndexCreatesOnlyMissingVersionedIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"index not found","code":"index_not_found","type":"invalid_request"}`)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/indexes" {
			var body struct {
				UID        string `json:"uid"`
				PrimaryKey string `json:"primaryKey"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create request: %v", err)
			}
			if body.UID != "article_chunks_v1" || body.PrimaryKey != "id" {
				t.Errorf("create request = %+v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"taskUid":7,"indexUid":"article_chunks_v1","status":"enqueued","type":"indexCreation"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := meilisearch.New(server.URL)
	task, err := EnsureArticleChunksIndex(context.Background(), client, testArticleChunksSpec())
	if err != nil {
		t.Fatalf("EnsureArticleChunksIndex() error = %v", err)
	}
	if task == nil || task.TaskUID != 7 {
		t.Fatalf("task = %+v, want task UID 7", task)
	}
}

func TestEnsureArticleChunksIndexLeavesExistingIndexUntouched(t *testing.T) {
	postCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCalls++
		}
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"article_chunks_v1","primaryKey":"id"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := meilisearch.New(server.URL)
	task, err := EnsureArticleChunksIndex(context.Background(), client, testArticleChunksSpec())
	if err != nil || task != nil {
		t.Fatalf("EnsureArticleChunksIndex() = task %v error %v, want no-op", task, err)
	}
	if postCalls != 0 {
		t.Fatalf("create calls = %d, want 0", postCalls)
	}
}

func TestEnsureArticleChunksIndexRejectsPrimaryKeyMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"article_chunks_v1","primaryKey":"wrong"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := meilisearch.New(server.URL)
	_, err := EnsureArticleChunksIndex(context.Background(), client, testArticleChunksSpec())
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("EnsureArticleChunksIndex() error kind = %v, want %v", apperrors.KindOf(err), apperrors.KindConflict)
	}
}

func TestProvisionArticleChunksIndexConfiguresExplicitSearchAndFilterFields(t *testing.T) {
	var paths []string
	var filterableFields []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"article_chunks_v1","primaryKey":"id"}`)
			return
		}
		if r.Method == http.MethodPut && (r.URL.Path == "/indexes/article_chunks_v1/settings/searchable-attributes" || r.URL.Path == "/indexes/article_chunks_v1/settings/filterable-attributes") {
			if r.URL.Path == "/indexes/article_chunks_v1/settings/filterable-attributes" {
				if err := json.NewDecoder(r.Body).Decode(&filterableFields); err != nil {
					t.Errorf("decode filterable fields: %v", err)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"taskUid":8}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := meilisearch.New(server.URL)
	tasks, err := ProvisionArticleChunksIndex(context.Background(), client, testArticleChunksSpec())
	if err != nil {
		t.Fatalf("ProvisionArticleChunksIndex() error = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2 settings tasks", len(tasks))
	}
	joined := strings.Join(paths, "|")
	for _, path := range []string{
		"GET /indexes/article_chunks_v1",
		"PUT /indexes/article_chunks_v1/settings/searchable-attributes",
		"PUT /indexes/article_chunks_v1/settings/filterable-attributes",
	} {
		if !strings.Contains(joined, path) {
			t.Fatalf("request paths = %q, missing %q", joined, path)
		}
	}
	if !reflect.DeepEqual(filterableFields, []string{
		ArticleChunkArticleIDField,
		ArticleChunkCategoryField,
		ArticleChunkTagsField,
		ArticleChunkPublishedAtField,
		ArticleChunkPublishedAtUnixField,
		ArticleChunkStatusField,
		ArticleChunkIsDeleteField,
	}) {
		t.Fatalf("filterable fields = %v", filterableFields)
	}
}

func TestSwapArticleChunksIndexSubmitsExplicitAtomicSwap(t *testing.T) {
	current := testArticleChunksSpec()
	candidate := current
	candidate.IndexVersion = "v2"
	candidate.UID = "article_chunks_v2"
	candidate.ModelVersion = "2026-08-28"

	var swapPayload []struct {
		Indexes []string `json:"indexes"`
		Rename  bool     `json:"rename"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && (r.URL.Path == "/indexes/article_chunks_v1" || r.URL.Path == "/indexes/article_chunks_v2"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"`+strings.TrimPrefix(r.URL.Path, "/indexes/")+`","primaryKey":"id"}`)
		case r.Method == http.MethodGet && (r.URL.Path == "/indexes/article_chunks_v1/settings" || r.URL.Path == "/indexes/article_chunks_v2/settings"):
			writeArticleChunksSettings(w)
		case r.Method == http.MethodPost && r.URL.Path == "/swap-indexes":
			if err := json.NewDecoder(r.Body).Decode(&swapPayload); err != nil {
				t.Errorf("decode swap payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"taskUid":42,"status":"enqueued","type":"indexSwap"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	plan, err := NewArticleChunksIndexSwapPlan(current, candidate)
	if err != nil {
		t.Fatalf("NewArticleChunksIndexSwapPlan() error = %v", err)
	}
	task, err := SwapArticleChunksIndex(context.Background(), meilisearch.New(server.URL), plan)
	if err != nil {
		t.Fatalf("SwapArticleChunksIndex() error = %v", err)
	}
	if task.TaskUID != 42 {
		t.Fatalf("swap task UID = %d, want 42", task.TaskUID)
	}
	if len(swapPayload) != 1 || strings.Join(swapPayload[0].Indexes, ",") != "article_chunks_v1,article_chunks_v2" || swapPayload[0].Rename {
		t.Fatalf("swap payload = %+v, want non-rename pair", swapPayload)
	}

	rollback := plan.Reverse()
	if rollback.Current.UID != candidate.UID || rollback.Candidate.UID != current.UID {
		t.Fatalf("rollback plan = %+v, want reversed UIDs", rollback)
	}
	if err := rollback.Validate(); err != nil {
		t.Fatalf("rollback plan validation error = %v", err)
	}
}

func TestValidateArticleChunksIndexSettingsIgnoresFilterableAttributeOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1/settings" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"searchableAttributes":["articleTitle","text"],"filterableAttributes":["articleId","category","isDelete","publishedAt","publishedAtUnix","status","tags"]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	if err := validateArticleChunksIndexSettings(context.Background(), meilisearch.New(server.URL), "article_chunks_v1"); err != nil {
		t.Fatalf("validateArticleChunksIndexSettings() error = %v, want success for reordered filterable attributes", err)
	}
}

func TestSwapArticleChunksIndexRequiresBothIndexes(t *testing.T) {
	current := testArticleChunksSpec()
	candidate := current
	candidate.IndexVersion = "v2"
	candidate.UID = "article_chunks_v2"
	candidate.ModelVersion = "2026-08-28"
	postCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"article_chunks_v1","primaryKey":"id"}`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1/settings" {
			writeArticleChunksSettings(w)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/swap-indexes" {
			postCalls++
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"index not found","code":"index_not_found","type":"invalid_request"}`)
	}))
	defer server.Close()

	plan, err := NewArticleChunksIndexSwapPlan(current, candidate)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SwapArticleChunksIndex(context.Background(), meilisearch.New(server.URL), plan)
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("SwapArticleChunksIndex() error kind = %v, want not_found", apperrors.KindOf(err))
	}
	if postCalls != 0 {
		t.Fatalf("swap calls = %d, want no swap when candidate is missing", postCalls)
	}
}

func TestSwapArticleChunksIndexRejectsMismatchedSettings(t *testing.T) {
	current := testArticleChunksSpec()
	candidate := current
	candidate.IndexVersion = "v2"
	candidate.UID = "article_chunks_v2"
	candidate.ModelVersion = "2026-08-28"
	postCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && (r.URL.Path == "/indexes/article_chunks_v1" || r.URL.Path == "/indexes/article_chunks_v2"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"uid":"`+strings.TrimPrefix(r.URL.Path, "/indexes/")+`","primaryKey":"id"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v1/settings":
			writeArticleChunksSettings(w)
		case r.Method == http.MethodGet && r.URL.Path == "/indexes/article_chunks_v2/settings":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"searchableAttributes":["text"],"filterableAttributes":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/swap-indexes":
			postCalls++
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	plan, err := NewArticleChunksIndexSwapPlan(current, candidate)
	if err != nil {
		t.Fatal(err)
	}
	_, err = SwapArticleChunksIndex(context.Background(), meilisearch.New(server.URL), plan)
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("SwapArticleChunksIndex() error kind = %v, want conflict", apperrors.KindOf(err))
	}
	if postCalls != 0 {
		t.Fatalf("swap calls = %d, want no swap when settings mismatch", postCalls)
	}
}

func writeArticleChunksSettings(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"searchableAttributes":["articleTitle","text"],"filterableAttributes":["articleId","category","tags","publishedAt","publishedAtUnix","status","isDelete"]}`)
}

func TestWaitForArticleChunksTasksRequiresSuccessfulTasks(t *testing.T) {
	reader := &fakeArticleChunksTaskReader{
		wait: func(_ context.Context, taskUID int64, _ time.Duration) (*meilisearch.Task, error) {
			return &meilisearch.Task{TaskUID: taskUID, IndexUID: "article_chunks_v1", Status: meilisearch.TaskStatusSucceeded}, nil
		},
	}
	completed, err := WaitForArticleChunksTasks(context.Background(), reader, "article_chunks_v1", []*meilisearch.TaskInfo{{TaskUID: 9, IndexUID: "article_chunks_v1"}}, 0)
	if err != nil {
		t.Fatalf("WaitForArticleChunksTasks() error = %v", err)
	}
	if len(completed) != 1 || completed[0].TaskUID != 9 {
		t.Fatalf("completed tasks = %+v, want task 9", completed)
	}

	reader.wait = func(_ context.Context, taskUID int64, _ time.Duration) (*meilisearch.Task, error) {
		return &meilisearch.Task{TaskUID: taskUID, IndexUID: "article_chunks_v1", Status: meilisearch.TaskStatusFailed}, nil
	}
	if _, err := WaitForArticleChunksTasks(context.Background(), reader, "article_chunks_v1", []*meilisearch.TaskInfo{{TaskUID: 10}}, time.Millisecond); !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("failed task error kind = %v, want unavailable", apperrors.KindOf(err))
	}
}

func TestArticleChunksTaskUIDNormalizesCurrentAndLegacyTaskResponses(t *testing.T) {
	if got := ArticleChunksTaskUID(&meilisearch.Task{UID: 9}); got != 9 {
		t.Fatalf("ArticleChunksTaskUID(current) = %d, want 9", got)
	}
	if got := ArticleChunksTaskUID(&meilisearch.Task{TaskUID: 10}); got != 10 {
		t.Fatalf("ArticleChunksTaskUID(legacy) = %d, want 10", got)
	}
	if got := ArticleChunksTaskUID(nil); got != 0 {
		t.Fatalf("ArticleChunksTaskUID(nil) = %d, want 0", got)
	}
}

func TestWaitForArticleChunksTasksRejectsInvalidTaskOwnershipAndCancellation(t *testing.T) {
	reader := &fakeArticleChunksTaskReader{
		wait: func(ctx context.Context, _ int64, _ time.Duration) (*meilisearch.Task, error) {
			return nil, ctx.Err()
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := WaitForArticleChunksTasks(ctx, reader, "article_chunks_v1", []*meilisearch.TaskInfo{{TaskUID: 11}}, time.Millisecond); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v, want context canceled", err)
	}

	reader.wait = func(_ context.Context, _ int64, _ time.Duration) (*meilisearch.Task, error) {
		return &meilisearch.Task{TaskUID: 12, IndexUID: "article_chunks_v2", Status: meilisearch.TaskStatusSucceeded}, nil
	}
	if _, err := WaitForArticleChunksTasks(context.Background(), reader, "article_chunks_v1", []*meilisearch.TaskInfo{{TaskUID: 12}}, time.Millisecond); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("ownership error kind = %v, want conflict", apperrors.KindOf(err))
	}

	if _, err := WaitForArticleChunksTasks(context.Background(), reader, "article_chunks_v1", []*meilisearch.TaskInfo{{TaskUID: 13, IndexUID: "article_chunks_v2"}}, time.Millisecond); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("task info ownership error kind = %v, want conflict", apperrors.KindOf(err))
	}
}

type fakeArticleChunksTaskReader struct {
	wait func(context.Context, int64, time.Duration) (*meilisearch.Task, error)
}

func (f *fakeArticleChunksTaskReader) GetTask(int64) (*meilisearch.Task, error) {
	return nil, nil
}

func (f *fakeArticleChunksTaskReader) GetTaskWithContext(context.Context, int64) (*meilisearch.Task, error) {
	return nil, nil
}

func (f *fakeArticleChunksTaskReader) GetTasks(*meilisearch.TasksQuery) (*meilisearch.TaskResult, error) {
	return nil, nil
}

func (f *fakeArticleChunksTaskReader) GetTasksWithContext(context.Context, *meilisearch.TasksQuery) (*meilisearch.TaskResult, error) {
	return nil, nil
}

func (f *fakeArticleChunksTaskReader) WaitForTask(taskUID int64, interval time.Duration) (*meilisearch.Task, error) {
	return f.wait(context.Background(), taskUID, interval)
}

func (f *fakeArticleChunksTaskReader) WaitForTaskWithContext(ctx context.Context, taskUID int64, interval time.Duration) (*meilisearch.Task, error) {
	return f.wait(ctx, taskUID, interval)
}

func TestNewArticleChunksIndexSwapPlanRejectsSameUID(t *testing.T) {
	spec := testArticleChunksSpec()
	if _, err := NewArticleChunksIndexSwapPlan(spec, spec); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("same UID error kind = %v, want validation", apperrors.KindOf(err))
	}

	changed := spec
	changed.ModelVersion = "2026-08-28"
	if _, err := NewArticleChunksIndexSwapPlan(spec, changed); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("same UID changed contract error kind = %v, want conflict", apperrors.KindOf(err))
	}
}
