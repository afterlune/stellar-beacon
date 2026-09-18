package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type contentAnalyticsArticleRepo struct {
	fakeArticleRepository
	article entity.TArticle
}

func (f *contentAnalyticsArticleRepo) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.article, nil
}

type contentAnalyticsCache struct {
	fakeServiceCache
	created bool
	pfCount int64
	pfErr   error
}

func (c *contentAnalyticsCache) SetNX(context.Context, string, any, time.Duration) (bool, error) {
	if c.created {
		return false, nil
	}
	c.created = true
	return true, nil
}

func (c *contentAnalyticsCache) PFCount(context.Context, ...string) (int64, error) {
	return c.pfCount, c.pfErr
}

type contentAnalyticsRecordingRepo struct {
	fakeContentAnalyticsRepository
	sessions    int
	dailyRows   []port.ContentDailyMetric
	articleRows []port.ContentArticleMetric
}

func (r *contentAnalyticsRecordingRepo) RecordReadSession(context.Context, int, time.Time, int, int, int64) error {
	r.sessions++
	return nil
}

func (r *contentAnalyticsRecordingRepo) ListDailyMetrics(context.Context, string, string) ([]port.ContentDailyMetric, error) {
	return r.dailyRows, nil
}

func (r *contentAnalyticsRecordingRepo) ListArticleMetrics(context.Context, string, string) ([]port.ContentArticleMetric, error) {
	return r.articleRows, nil
}

func (r *contentAnalyticsRecordingRepo) GetArticleDailyMetrics(context.Context, int, string, string) ([]port.ContentDailyMetric, error) {
	return r.dailyRows, nil
}

type contentAnalyticsBotVisitor struct{}

func (contentAnalyticsBotVisitor) Resolve(context.Context, *http.Request) (port.VisitorIdentity, error) {
	return port.VisitorIdentity{IsBot: true, Fingerprint: "bot"}, nil
}

func newContentAnalyticsTestContext(articleID, payload string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/articles/"+articleID+"/read-sessions", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "articleId", Value: articleID}}
	return c
}

func TestTrackReadSessionIsIdempotent(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{}
	cache := &contentAnalyticsCache{pfCount: 1}
	articles := &contentAnalyticsArticleRepo{article: entity.TArticle{Id: 7, Status: 1}}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: articles, Cache: cache, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"sessionId":"session-1234","activeMs":5000,"maxScrollPercent":95}`
	for i := 0; i < 2; i++ {
		c := newContentAnalyticsTestContext("7", payload)
		result := svc.TrackReadSession(c)
		if !result.Flag {
			t.Fatalf("unexpected result on report %d: %+v", i+1, result)
		}
	}
	if repo.sessions != 1 {
		t.Fatalf("expected one aggregate write, got %d", repo.sessions)
	}
}

func TestTrackReadSessionValidatesPayloadAndArticle(t *testing.T) {
	tests := []struct {
		name    string
		article entity.TArticle
		payload string
	}{
		{name: "invalid session id", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"sessionId":"short","activeMs":5000,"maxScrollPercent":95}`},
		{name: "too short active time", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"sessionId":"session-1234","activeMs":2999,"maxScrollPercent":95}`},
		{name: "active time over limit", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"sessionId":"session-1234","activeMs":7200001,"maxScrollPercent":95}`},
		{name: "negative scroll", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"sessionId":"session-1234","activeMs":5000,"maxScrollPercent":-1}`},
		{name: "scroll over 100", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"sessionId":"session-1234","activeMs":5000,"maxScrollPercent":101}`},
		{name: "unpublished article", article: entity.TArticle{Id: 7, Status: 3}, payload: `{"sessionId":"session-1234","activeMs":5000,"maxScrollPercent":95}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
				Repo: &contentAnalyticsRecordingRepo{}, Articles: &contentAnalyticsArticleRepo{article: test.article},
				Cache: &contentAnalyticsCache{pfCount: 1}, Visitor: fakeServiceVisitor{},
			})
			if err != nil {
				t.Fatal(err)
			}
			result := svc.TrackReadSession(newContentAnalyticsTestContext("7", test.payload))
			if result.Flag {
				t.Fatalf("expected validation failure, got %+v", result)
			}
		})
	}
}

func TestTrackReadSessionSkipsBots(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{article: entity.TArticle{Id: 7, Status: 1}},
		Cache: &contentAnalyticsCache{pfCount: 1}, Visitor: contentAnalyticsBotVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.TrackReadSession(newContentAnalyticsTestContext("7", `{"sessionId":"session-1234","activeMs":5000,"maxScrollPercent":95}`))
	if !result.Flag || repo.sessions != 0 {
		t.Fatalf("bot report should be a no-op: result=%+v sessions=%d", result, repo.sessions)
	}
}

func TestContentAnalyticsPrefersHyperLogLogUniqueReaders(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{dailyRows: []port.ContentDailyMetric{
		{Date: time.Now().Format("2006-01-02"), Views: 10, UniqueReaders: 5, EffectiveSessions: 4, TotalActiveMs: 20000, CompletedSessions: 3},
	}}
	cache := &contentAnalyticsCache{pfCount: 2}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{}, Cache: cache, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.GetContentAnalytics(context.Background(), "7d")
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, ok := result.Data.(model.ContentAnalyticsDTO)
	if !ok || data.Overview.UniqueReaders != 2 {
		t.Fatalf("expected HLL unique readers, got %#v", result.Data)
	}
}

func TestContentAnalyticsFallsBackToDatabaseWhenHyperLogLogFails(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{dailyRows: []port.ContentDailyMetric{
		{Date: time.Now().Format("2006-01-02"), Views: 10, UniqueReaders: 5, EffectiveSessions: 4, TotalActiveMs: 20000, CompletedSessions: 3},
	}}
	cache := &contentAnalyticsCache{pfErr: context.DeadlineExceeded}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{}, Cache: cache, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.GetContentAnalytics(context.Background(), "7d")
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	data, ok := result.Data.(model.ContentAnalyticsDTO)
	if !ok || data.Overview.UniqueReaders != 5 {
		t.Fatalf("expected database fallback, got %#v", result.Data)
	}
}

func TestListContentAnalyticsArticlesSortsAndPaginates(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{articleRows: []port.ContentArticleMetric{
		{ArticleId: 1, ArticleTitle: "first", Views: 100, EffectiveSessions: 10, TotalActiveMs: 30000, CompletedSessions: 5},
		{ArticleId: 2, ArticleTitle: "second", Views: 200, EffectiveSessions: 5, TotalActiveMs: 30000, CompletedSessions: 4},
	}}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{}, Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.ListContentAnalyticsArticles(context.Background(), "7d", "views", 1, 1)
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(model.PageResultDTO)
	if !ok || page.Count != 2 {
		t.Fatalf("unexpected page result: %#v", result.Data)
	}
	records, ok := page.Records.([]model.ContentArticlePerformanceDTO)
	if !ok || len(records) != 1 || records[0].ArticleID != 2 {
		t.Fatalf("unexpected sorted records: %#v", page.Records)
	}
}

func TestResolveContentAnalyticsRange(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	if window := resolveContentAnalyticsRange("90d", now); len(window.Days) != 90 || window.Unit != "day" {
		t.Fatalf("unexpected 90d range: unit=%s days=%d", window.Unit, len(window.Days))
	}
	window := resolveContentAnalyticsRange("12m", now)
	if window.Unit != "month" || len(window.Buckets) != 12 || window.Start.Format("2006-01") != "2025-10" || window.End.Format("2006-01") != "2026-09" {
		t.Fatalf("unexpected 12m range: %+v", window)
	}
}

var _ port.Cache = (*contentAnalyticsCache)(nil)
