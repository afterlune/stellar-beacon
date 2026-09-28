package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type contentAnalyticsArticleRepo struct {
	fakeArticleRepository
	article     entity.TArticle
	seriesCards []*port.ArticleCard
}

func (f *contentAnalyticsArticleRepo) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.article, nil
}

func (f *contentAnalyticsArticleRepo) ListArticleCardsBySeries(context.Context, int) ([]*port.ArticleCard, error) {
	return f.seriesCards, nil
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
	sessions            int
	continuationEvents  []port.ContinuationEventType
	continuationTargets []*port.ContinuationTarget
	continuationErr     error
	dailyRows           []port.ContentDailyMetric
	articleRows         []port.ContentArticleMetric
	targetRows          []port.ContinuationTargetMetric
}

func (r *contentAnalyticsRecordingRepo) RecordReadSession(context.Context, int, time.Time, int, int, int64) error {
	r.sessions++
	return nil
}

func (r *contentAnalyticsRecordingRepo) RecordContinuationEvent(_ context.Context, _ int, _ time.Time, eventType port.ContinuationEventType, target *port.ContinuationTarget) error {
	if r.continuationErr != nil {
		return r.continuationErr
	}
	r.continuationEvents = append(r.continuationEvents, eventType)
	r.continuationTargets = append(r.continuationTargets, target)
	return nil
}

func (r *contentAnalyticsRecordingRepo) ListContinuationTargetMetrics(context.Context, string, string, int) ([]port.ContinuationTargetMetric, error) {
	return r.targetRows, nil
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

func newContinuationEventTestContext(articleID, payload string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/articles/"+articleID+"/continuation-events", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "articleId", Value: articleID}}
	return c
}

func TestTrackContinuationEventRecordsAllowedEvents(t *testing.T) {
	tests := []struct {
		name        string
		article     entity.TArticle
		related     []*port.ArticleCard
		seriesCards []*port.ArticleCard
		payload     string
		wantTarget  *port.ContinuationTarget
	}{
		{
			name: "series impression", article: entity.TArticle{Id: 7, Status: 1},
			payload: `{"eventType":"series_impression"}`,
		},
		{
			name: "related impression", article: entity.TArticle{Id: 7, Status: 1},
			payload: `{"eventType":"related_impression"}`,
		},
		{
			name: "related click", article: entity.TArticle{Id: 7, CategoryId: 1, Status: 1},
			related:    []*port.ArticleCard{{Id: 20, Status: 1}},
			payload:    `{"eventType":"related_click","targetType":"article","targetId":20,"placement":"related","position":1}`,
			wantTarget: &port.ContinuationTarget{Type: port.ContinuationTargetArticle, Id: 20, Placement: port.ContinuationPlacementRelated, Position: 1},
		},
		{
			name: "series next click", article: entity.TArticle{Id: 8, SeriesId: 3, Status: 1},
			seriesCards: []*port.ArticleCard{{Id: 7, Status: 1}, {Id: 8, Status: 1}, {Id: 9, Status: 1}},
			payload:     `{"eventType":"series_click","targetType":"article","targetId":9,"placement":"series_next","position":0}`,
			wantTarget:  &port.ContinuationTarget{Type: port.ContinuationTargetArticle, Id: 9, Placement: port.ContinuationPlacementSeriesNext, Position: 0},
		},
		{
			name: "series index click", article: entity.TArticle{Id: 8, SeriesId: 3, Status: 1},
			payload:    `{"eventType":"series_click","targetType":"series","targetId":3,"placement":"series_index","position":0}`,
			wantTarget: &port.ContinuationTarget{Type: port.ContinuationTargetSeries, Id: 3, Placement: port.ContinuationPlacementSeriesIndex, Position: 0},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			articles := &contentAnalyticsArticleRepo{article: test.article, seriesCards: test.seriesCards}
			articles.related = test.related
			repo := &contentAnalyticsRecordingRepo{}
			svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
				Repo: repo, Articles: articles, Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
			})
			if err != nil {
				t.Fatal(err)
			}
			result := svc.TrackContinuationEvent(newContinuationEventTestContext("7", test.payload))
			if !result.Flag || len(repo.continuationEvents) != 1 {
				t.Fatalf("event was not recorded: result=%+v events=%v", result, repo.continuationEvents)
			}
			if len(repo.continuationTargets) != 1 {
				t.Fatalf("unexpected target count: %#v", repo.continuationTargets)
			}
			gotTarget := repo.continuationTargets[0]
			if test.wantTarget == nil {
				if gotTarget != nil {
					t.Fatalf("impression target = %#v, want nil", gotTarget)
				}
				return
			}
			if gotTarget == nil || *gotTarget != *test.wantTarget {
				t.Fatalf("target = %#v, want %#v", gotTarget, test.wantTarget)
			}
		})
	}
}
func TestTrackContinuationEventValidatesInputsAndArticle(t *testing.T) {
	tests := []struct {
		name        string
		article     entity.TArticle
		related     []*port.ArticleCard
		seriesCards []*port.ArticleCard
		payload     string
	}{
		{name: "invalid event", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"eventType":"other"}`},
		{name: "malformed payload", article: entity.TArticle{Id: 7, Status: 1}, payload: `{`},
		{name: "unpublished article", article: entity.TArticle{Id: 7, Status: 3}, payload: `{"eventType":"related_click"}`},
		{name: "click without target", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"eventType":"related_click"}`},
		{name: "impression with target", article: entity.TArticle{Id: 7, Status: 1}, payload: `{"eventType":"related_impression","targetType":"article","targetId":20,"placement":"related","position":1}`},
		{
			name: "related target no longer recommended", article: entity.TArticle{Id: 7, CategoryId: 1, Status: 1},
			related: []*port.ArticleCard{{Id: 20, Status: 1}},
			payload: `{"eventType":"related_click","targetType":"article","targetId":21,"placement":"related","position":1}`,
		},
		{
			name: "related target position mismatch", article: entity.TArticle{Id: 7, CategoryId: 1, Status: 1},
			related: []*port.ArticleCard{{Id: 20, Status: 1}, {Id: 21, Status: 1}},
			payload: `{"eventType":"related_click","targetType":"article","targetId":20,"placement":"related","position":2}`,
		},
		{
			name: "series neighbor mismatch", article: entity.TArticle{Id: 8, SeriesId: 3, Status: 1},
			seriesCards: []*port.ArticleCard{{Id: 7, Status: 1}, {Id: 8, Status: 1}, {Id: 9, Status: 1}},
			payload:     `{"eventType":"series_click","targetType":"article","targetId":7,"placement":"series_next","position":0}`,
		},
		{
			name: "series index mismatch", article: entity.TArticle{Id: 8, SeriesId: 3, Status: 1},
			payload: `{"eventType":"series_click","targetType":"series","targetId":4,"placement":"series_index","position":0}`,
		},
		{
			name: "source has no series", article: entity.TArticle{Id: 8, Status: 1},
			payload: `{"eventType":"series_click","targetType":"series","targetId":3,"placement":"series_index","position":0}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			articles := &contentAnalyticsArticleRepo{article: test.article, seriesCards: test.seriesCards}
			articles.related = test.related
			repo := &contentAnalyticsRecordingRepo{}
			svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
				Repo: repo, Articles: articles, Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
			})
			if err != nil {
				t.Fatal(err)
			}
			result := svc.TrackContinuationEvent(newContinuationEventTestContext("7", test.payload))
			if result.Flag || len(repo.continuationEvents) != 0 {
				t.Fatalf("expected rejected event: result=%+v events=%v", result, repo.continuationEvents)
			}
		})
	}
}
func TestTrackContinuationEventSkipsBots(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{article: entity.TArticle{Id: 7, Status: 1}},
		Cache: &contentAnalyticsCache{}, Visitor: contentAnalyticsBotVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.TrackContinuationEvent(newContinuationEventTestContext("7", `{"eventType":"series_impression"}`))
	if !result.Flag || len(repo.continuationEvents) != 0 {
		t.Fatalf("bot event should be a no-op: result=%+v events=%v", result, repo.continuationEvents)
	}
}

func TestTrackContinuationEventPropagatesRepositoryFailure(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{continuationErr: context.DeadlineExceeded}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{article: entity.TArticle{Id: 7, Status: 1}},
		Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.TrackContinuationEvent(newContinuationEventTestContext("7", `{"eventType":"related_impression"}`))
	if result.Flag {
		t.Fatalf("expected repository failure, got %+v", result)
	}
}
func TestContentContinuationMetricsRates(t *testing.T) {
	metrics := contentContinuationMetrics(port.ContentDailyMetric{
		SeriesImpressions: 4, SeriesClicks: 1,
		RelatedImpressions: 8, RelatedClicks: 2,
	})
	if metrics.SeriesClickRate != 25 || metrics.RelatedClickRate != 25 || metrics.ContinuationRate != 25 {
		t.Fatalf("unexpected continuation metrics: %+v", metrics)
	}
	if zero := contentContinuationMetrics(port.ContentDailyMetric{}); zero.ContinuationRate != 0 {
		t.Fatalf("zero denominator should produce zero rate: %+v", zero)
	}
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

func TestListContentAnalyticsArticlesSortsByContinuationRate(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{articleRows: []port.ContentArticleMetric{
		{ArticleId: 1, ArticleTitle: "low continuation", Views: 100, SeriesImpressions: 100, SeriesClicks: 5, RelatedImpressions: 100, RelatedClicks: 5},
		{ArticleId: 2, ArticleTitle: "high continuation", Views: 10, SeriesImpressions: 100, SeriesClicks: 20, RelatedImpressions: 100, RelatedClicks: 20},
	}}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{}, Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.ListContentAnalyticsArticles(context.Background(), "7d", "continuationRate", 1, 1)
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(model.PageResultDTO)
	if !ok {
		t.Fatalf("unexpected page type: %T", result.Data)
	}
	records, ok := page.Records.([]model.ContentArticlePerformanceDTO)
	if !ok || len(records) != 1 || records[0].ArticleID != 2 || records[0].Continuation.ContinuationRate != 20 {
		t.Fatalf("unexpected continuation sort: %#v", page.Records)
	}
}
func TestListContinuationTargetsSortsAndPaginates(t *testing.T) {
	repo := &contentAnalyticsRecordingRepo{targetRows: []port.ContinuationTargetMetric{
		{SourceArticleId: 1, SourceArticleTitle: "source one", TargetType: port.ContinuationTargetArticle, TargetId: 10, TargetTitle: "target ten", Placement: port.ContinuationPlacementRelated, Position: 1, Clicks: 5, ModuleImpressions: 10},
		{SourceArticleId: 2, SourceArticleTitle: "source two", TargetType: port.ContinuationTargetArticle, TargetId: 20, TargetTitle: "target twenty", Placement: port.ContinuationPlacementRelated, Position: 2, Clicks: 8, ModuleImpressions: 8},
	}}
	svc, err := NewContentAnalyticsService(ContentAnalyticsServiceDeps{
		Repo: repo, Articles: &contentAnalyticsArticleRepo{}, Cache: &contentAnalyticsCache{}, Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result := svc.ListContinuationTargets(context.Background(), "7d", "clickRate", 0, 1, 1)
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(model.PageResultDTO)
	if !ok || page.Count != 2 {
		t.Fatalf("unexpected page: %#v", result.Data)
	}
	records, ok := page.Records.([]model.ContentContinuationTargetDTO)
	if !ok || len(records) != 1 || records[0].TargetID != 20 || records[0].ClickRate != 100 {
		t.Fatalf("unexpected target ordering: %#v", page.Records)
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
