package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

type fakeContentUnderstandingGateway struct {
	result   port.ContentUnderstanding
	err      error
	requests []port.ContentUnderstandingRequest
}

func (f *fakeContentUnderstandingGateway) Analyze(_ context.Context, request port.ContentUnderstandingRequest) (port.ContentUnderstanding, error) {
	f.requests = append(f.requests, request)
	return f.result, f.err
}

func contentUnderstandingJobPayload(t *testing.T, articleID int) []byte {
	t.Helper()
	payload, err := port.NewContentUnderstandingJobPayload(articleID, time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func newContentUnderstandingWorkerForTest(t *testing.T, jobs *fakeArticleIndexJobs, analyzer port.ContentUnderstandingGateway) *ContentUnderstandingWorker {
	t.Helper()
	worker, err := NewContentUnderstandingWorker(ContentUnderstandingWorkerDeps{
		Jobs: jobs,
		Articles: &fakeArticleRepositoryForIndex{article: entity.TArticle{
			Id:             42,
			ArticleTitle:   "Agent notes",
			ArticleContent: "one two three",
			Status:         port.PublicArticleStatus,
		}},
		Analyzer:  analyzer,
		WorkerID:  "content-test-worker",
		RetryBase: time.Second,
	})
	if err != nil {
		t.Fatalf("NewContentUnderstandingWorker() error = %v", err)
	}
	return worker
}

func TestContentUnderstandingWorkerAnalyzesCurrentPublicArticle(t *testing.T) {
	jobs := &fakeArticleIndexJobs{job: port.AIJob{
		ID:          "job-content-1",
		Kind:        port.AIJobKindContentUnderstanding,
		Payload:     contentUnderstandingJobPayload(t, 42),
		Attempts:    1,
		MaxAttempts: 3,
	}}
	analyzer := &fakeContentUnderstandingGateway{result: port.ContentUnderstanding{
		Summary:  "文章摘要",
		Category: "工程",
		Tags:     []string{"Go", "Agent"},
		RunID:    "run-content-1",
	}}
	worker := newContentUnderstandingWorkerForTest(t, jobs, analyzer)
	processed, err := worker.processOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("processOne() = processed %v error %v", processed, err)
	}
	if len(analyzer.requests) != 1 || analyzer.requests[0].ArticleID != 42 || analyzer.requests[0].Title != "Agent notes" {
		t.Fatalf("analyzer requests = %+v", analyzer.requests)
	}
	if len(jobs.completed) != 1 || jobs.completed[0].RunID != "run-content-1" || len(jobs.retried) != 0 {
		t.Fatalf("job transitions = %+v", jobs)
	}
	var result contentUnderstandingJobResult
	if err := json.Unmarshal(jobs.completed[0].Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ready" || result.ArticleID != 42 || result.Summary != "文章摘要" || result.Category != "工程" || len(result.Tags) != 2 {
		t.Fatalf("result = %+v", result)
	}
}

func TestContentUnderstandingWorkerSkipsNonPublicAndMissingArticles(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		deleted    int
		err        error
		statusWant string
	}{
		{name: "private", status: 2, statusWant: "skipped_not_public"},
		{name: "deleted", status: port.PublicArticleStatus, deleted: 1, statusWant: "skipped_not_public"},
		{name: "missing", err: apperrors.NotFound("article.get_admin"), statusWant: "skipped_article_not_found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-content-skip", Kind: port.AIJobKindContentUnderstanding, Payload: contentUnderstandingJobPayload(t, 42), Attempts: 1, MaxAttempts: 3}}
			analyzer := &fakeContentUnderstandingGateway{result: port.ContentUnderstanding{Summary: "unused", Category: "unused"}}
			worker := newContentUnderstandingWorkerForTest(t, jobs, analyzer)
			repository := worker.articles.(*fakeArticleRepositoryForIndex)
			repository.article.Status = test.status
			repository.article.IsDelete = test.deleted
			repository.err = test.err
			if _, err := worker.processOne(context.Background()); err != nil {
				t.Fatalf("processOne() error = %v", err)
			}
			if len(analyzer.requests) != 0 || len(jobs.completed) != 1 {
				t.Fatalf("analyzer calls=%d completed=%d", len(analyzer.requests), len(jobs.completed))
			}
			var result contentUnderstandingJobResult
			if err := json.Unmarshal(jobs.completed[0].Payload, &result); err != nil {
				t.Fatal(err)
			}
			if result.Status != test.statusWant {
				t.Fatalf("status = %q, want %q", result.Status, test.statusWant)
			}
		})
	}
}

func TestContentUnderstandingWorkerRetriesAndDeadLettersAnalyzerFailure(t *testing.T) {
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
			jobs := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-content-failure", Kind: port.AIJobKindContentUnderstanding, Payload: contentUnderstandingJobPayload(t, 42), Attempts: test.attempts - 1, MaxAttempts: 3}}
			worker := newContentUnderstandingWorkerForTest(t, jobs, &fakeContentUnderstandingGateway{err: errors.New("provider failed")})
			processed, err := worker.processOne(context.Background())
			if err != nil || !processed {
				t.Fatalf("processOne() = processed %v error %v", processed, err)
			}
			if (len(jobs.retried) > 0) != test.wantRetry || (len(jobs.deadLetters) > 0) != test.wantDead {
				t.Fatalf("retry=%v dead=%v", jobs.retried, jobs.deadLetters)
			}
			if len(jobs.retried) > 0 && jobs.retried[0].lastError != "internal_error" {
				t.Fatalf("retry error = %q", jobs.retried[0].lastError)
			}
		})
	}
}

func TestContentUnderstandingWorkerValidatesDependenciesAndKinds(t *testing.T) {
	if _, err := NewContentUnderstandingWorker(ContentUnderstandingWorkerDeps{}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("constructor error kind = %v, want validation", apperrors.KindOf(err))
	}
	j := &fakeArticleIndexJobs{job: port.AIJob{ID: "job-other", Kind: "other", Payload: contentUnderstandingJobPayload(t, 42), Attempts: 1, MaxAttempts: 3}}
	worker := newContentUnderstandingWorkerForTest(t, j, &fakeContentUnderstandingGateway{})
	if _, err := worker.processOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(j.deadLetters) != 1 || len(j.completed) != 0 {
		t.Fatalf("unsupported kind transitions = %+v", j)
	}
}
