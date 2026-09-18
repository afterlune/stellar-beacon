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
}

func (c *contentAnalyticsCache) SetNX(context.Context, string, any, time.Duration) (bool, error) {
	if c.created {
		return false, nil
	}
	c.created = true
	return true, nil
}

func (c *contentAnalyticsCache) PFCount(context.Context, ...string) (int64, error) {
	return c.pfCount, nil
}

type contentAnalyticsRecordingRepo struct {
	fakeContentAnalyticsRepository
	sessions int
}

func (r *contentAnalyticsRecordingRepo) RecordReadSession(context.Context, int, time.Time, int, int, int64) error {
	r.sessions++
	return nil
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
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/public/articles/7/read-sessions", strings.NewReader(payload))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "articleId", Value: "7"}}
		result := svc.TrackReadSession(c)
		if !result.Flag {
			t.Fatalf("unexpected result on report %d: %+v", i+1, result)
		}
	}
	if repo.sessions != 1 {
		t.Fatalf("expected one aggregate write, got %d", repo.sessions)
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
