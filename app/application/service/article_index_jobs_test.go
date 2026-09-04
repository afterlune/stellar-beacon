package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"

	"github.com/gin-gonic/gin"
)

type recordingAIJobRepository struct {
	jobs []port.AIJob
	err  error
}

func (r *recordingAIJobRepository) Enqueue(_ context.Context, job port.AIJob) error {
	if r.err != nil {
		return r.err
	}
	r.jobs = append(r.jobs, job)
	return nil
}

func (r *recordingAIJobRepository) Claim(context.Context, string, time.Time, time.Duration) (port.AIJob, bool, error) {
	return port.AIJob{}, false, nil
}

func (r *recordingAIJobRepository) Complete(context.Context, string, string, port.AIJobResult) error {
	return nil
}

func (r *recordingAIJobRepository) Retry(context.Context, string, string, time.Time, string) error {
	return nil
}

func (r *recordingAIJobRepository) DeadLetter(context.Context, string, string, string) error {
	return nil
}

type articleJobRepository struct {
	fakeArticleRepository
	article entity.TArticle
}

func (r *articleJobRepository) SaveOrUpdate(context.Context, entity.TArticle, string, []string) (entity.TArticle, error) {
	return r.article, nil
}

func articleServiceWithJobs(t *testing.T, repo port.ArticleRepository, jobs port.AIJobRepository) *MyArticleService {
	t.Helper()
	service, err := NewArticleService(ArticleServiceDeps{
		Repo:    repo,
		Cache:   fakeServiceCache{},
		Storage: fakeServiceStorage{},
		Search:  &fakeArticleSearcher{},
		AIJobs:  jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func articleServiceWithContentUnderstandingJobs(t *testing.T, repo port.ArticleRepository, jobs port.AIJobRepository) *MyArticleService {
	t.Helper()
	service, err := NewArticleService(ArticleServiceDeps{
		Repo:                     repo,
		Cache:                    fakeServiceCache{},
		Storage:                  fakeServiceStorage{},
		Search:                   &fakeArticleSearcher{},
		ContentUnderstandingJobs: jobs,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func articleJSONRequestContext(target, body string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return serviceTestRequest{ginContextForServiceTest: c}
}

func decodeArticleIndexJob(t *testing.T, job port.AIJob) port.ArticleIndexJobPayload {
	t.Helper()
	var payload port.ArticleIndexJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		t.Fatalf("decode article index job: %v", err)
	}
	return payload
}

func decodeContentUnderstandingJob(t *testing.T, job port.AIJob) port.ContentUnderstandingJobPayload {
	t.Helper()
	var payload port.ContentUnderstandingJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		t.Fatalf("decode content understanding job: %v", err)
	}
	return payload
}

func TestArticleServiceEnqueuesDeleteStateJobs(t *testing.T) {
	jobs := &recordingAIJobRepository{}
	service := articleServiceWithJobs(t, &fakeArticleRepository{}, jobs)
	result := service.UpdateArticleDelete(articleJSONRequestContext("/admin/article/delete", `{"ids":[7,8],"isDelete":1}`))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(jobs.jobs) != 2 {
		t.Fatalf("enqueued jobs = %d, want 2", len(jobs.jobs))
	}
	for index, job := range jobs.jobs {
		payload := decodeArticleIndexJob(t, job)
		if job.Kind != port.AIJobKindArticleIndex || job.Status != port.AIJobPending || payload.ArticleID != []int{7, 8}[index] || payload.Action != port.ArticleIndexDelete || payload.Event != port.ArticleDeleted {
			t.Fatalf("unexpected delete job: %+v, payload=%+v", job, payload)
		}
	}
}

func TestArticleServiceEnqueuesArticleDeletionJobs(t *testing.T) {
	jobs := &recordingAIJobRepository{}
	service := articleServiceWithJobs(t, &fakeArticleRepository{}, jobs)
	result := service.DeleteArticles(articleJSONRequestContext("/admin/article/delete-hard", `[9]`))
	if !result.Flag || len(jobs.jobs) != 1 {
		t.Fatalf("unexpected result/jobs: %+v, %#v", result, jobs.jobs)
	}
	payload := decodeArticleIndexJob(t, jobs.jobs[0])
	if payload.ArticleID != 9 || payload.Action != port.ArticleIndexDelete || payload.Event != port.ArticleDeleted || payload.IsDelete != 1 {
		t.Fatalf("unexpected hard-delete job: %+v", payload)
	}
}

func TestArticleServiceEnqueuesPublishedArticleJob(t *testing.T) {
	when := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)
	jobs := &recordingAIJobRepository{}
	repo := &articleJobRepository{article: entity.TArticle{
		Id:             42,
		ArticleTitle:   "title",
		ArticleContent: "body",
		Status:         port.PublicArticleStatus,
		UpdateTime:     when,
	}}
	service := articleServiceWithJobs(t, repo, jobs)
	c := articleJSONRequestContext("/admin/article/save", `{"articleTitle":"title","articleContent":"body","status":1}`)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 1})
	result := service.SaveOrUpdateArticle(c)
	if !result.Flag || len(jobs.jobs) != 1 {
		t.Fatalf("unexpected result/jobs: %+v, %#v", result, jobs.jobs)
	}
	payload := decodeArticleIndexJob(t, jobs.jobs[0])
	if payload.ArticleID != 42 || payload.Action != port.ArticleIndexUpsert || payload.Event != port.ArticlePublished || payload.Status != port.PublicArticleStatus {
		t.Fatalf("unexpected publish job: %+v", payload)
	}
}

func TestArticleServiceEnqueuesPublishedContentUnderstandingJob(t *testing.T) {
	jobs := &recordingAIJobRepository{}
	repo := &articleJobRepository{article: entity.TArticle{
		Id:             42,
		ArticleTitle:   "title",
		ArticleContent: "body",
		Status:         port.PublicArticleStatus,
		UpdateTime:     time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC),
	}}
	service := articleServiceWithContentUnderstandingJobs(t, repo, jobs)
	c := articleJSONRequestContext("/admin/article/save", `{"articleTitle":"title","articleContent":"body","status":1}`)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 1})
	result := service.SaveOrUpdateArticle(c)
	if !result.Flag || len(jobs.jobs) != 1 {
		t.Fatalf("unexpected result/jobs: %+v, %#v", result, jobs.jobs)
	}
	job := jobs.jobs[0]
	payload := decodeContentUnderstandingJob(t, job)
	if job.Kind != port.AIJobKindContentUnderstanding || job.Status != port.AIJobPending || payload.ArticleID != 42 {
		t.Fatalf("unexpected content understanding job: %+v, payload=%+v", job, payload)
	}
	if !strings.Contains(job.IdempotencyKey, "article:42:content_understanding:") {
		t.Fatalf("unexpected idempotency key: %q", job.IdempotencyKey)
	}
}

func TestArticleServiceEnqueuesContentUnderstandingJobsWhenArticlesAreRestored(t *testing.T) {
	jobs := &recordingAIJobRepository{}
	service := articleServiceWithContentUnderstandingJobs(t, &fakeArticleRepository{}, jobs)
	result := service.UpdateArticleDelete(articleJSONRequestContext("/admin/article/delete", `{"ids":[7,8],"isDelete":0}`))
	if !result.Flag || len(jobs.jobs) != 2 {
		t.Fatalf("unexpected result/jobs: %+v, %#v", result, jobs.jobs)
	}
	for index, job := range jobs.jobs {
		payload := decodeContentUnderstandingJob(t, job)
		if job.Kind != port.AIJobKindContentUnderstanding || payload.ArticleID != []int{7, 8}[index] || job.Status != port.AIJobPending {
			t.Fatalf("unexpected restore job: %+v, payload=%+v", job, payload)
		}
	}
}

func TestArticleServiceDoesNotEnqueueContentUnderstandingForPrivateArticle(t *testing.T) {
	jobs := &recordingAIJobRepository{}
	repo := &articleJobRepository{article: entity.TArticle{
		Id:         42,
		Status:     2,
		UpdateTime: time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC),
	}}
	service := articleServiceWithContentUnderstandingJobs(t, repo, jobs)
	c := articleJSONRequestContext("/admin/article/save", `{"articleTitle":"draft","articleContent":"body","status":2}`)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 1})
	result := service.SaveOrUpdateArticle(c)
	if !result.Flag || len(jobs.jobs) != 0 {
		t.Fatalf("private article unexpectedly enqueued content job: %+v, %#v", result, jobs.jobs)
	}
}

func TestArticleServiceSurfacesArticleJobPersistenceFailure(t *testing.T) {
	jobs := &recordingAIJobRepository{err: apperrors.Unavailable("ai_job.enqueue", errors.New("queue unavailable"))}
	service := articleServiceWithJobs(t, &fakeArticleRepository{}, jobs)
	result := service.UpdateArticleDelete(articleJSONRequestContext("/admin/article/delete", `{"ids":[7],"isDelete":1}`))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected queue failure result: %+v", result)
	}
}
