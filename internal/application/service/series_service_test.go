package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"

	"github.com/gin-gonic/gin"
)

type fakeSeriesRepository struct {
	items   []*port.Series
	record  entity.TSeries
	saved   entity.TSeries
	deleted []int
}

func (f *fakeSeriesRepository) ListPublic(context.Context) ([]*port.Series, error) {
	return f.items, nil
}
func (f *fakeSeriesRepository) ListAdmin(context.Context, int, int, string) ([]*port.Series, int64, error) {
	return f.items, int64(len(f.items)), nil
}
func (f *fakeSeriesRepository) ListOptions(context.Context) ([]*port.Series, error) {
	return f.items, nil
}
func (f *fakeSeriesRepository) Get(context.Context, int) (entity.TSeries, error) {
	if f.record.Id == 0 {
		return entity.TSeries{}, apperrors.NotFound("series.get")
	}
	return f.record, nil
}
func (f *fakeSeriesRepository) SaveOrUpdate(_ context.Context, series entity.TSeries) (entity.TSeries, error) {
	f.saved = series
	if series.Id == 0 {
		series.Id = 12
	}
	return series, nil
}
func (f *fakeSeriesRepository) Delete(_ context.Context, seriesID int) error {
	f.deleted = append(f.deleted, seriesID)
	return nil
}

type seriesArticles struct {
	fakeArticleRepository
	cards []*port.ArticleCard
}

func (f *seriesArticles) ListArticleCardsBySeries(context.Context, int) ([]*port.ArticleCard, error) {
	return f.cards, nil
}

func mustSeriesService(t *testing.T, repo port.SeriesRepository, articles port.ArticleRepository) *MySeriesService {
	t.Helper()
	service, err := NewSeriesService(SeriesServiceDeps{Repo: repo, Articles: articles})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func seriesContext(t *testing.T, method, target, body string, params gin.Params) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if body == "" {
		c.Request = httptest.NewRequest(method, target, nil)
	} else {
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Params = params
	return c
}

func TestSeriesPublicDetailReturnsOrderedArticles(t *testing.T) {
	repo := &fakeSeriesRepository{record: entity.TSeries{Id: 3, SeriesName: "渲染管线"}}
	articles := &seriesArticles{cards: []*port.ArticleCard{{Id: 1, ArticleTitle: "第一篇"}, {Id: 2, ArticleTitle: "第二篇"}}}
	service := mustSeriesService(t, repo, articles)

	result := service.GetPublicSeries(seriesContext(t, http.MethodGet, "/v1/public/series/3", "", gin.Params{{Key: "seriesId", Value: "3"}}))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	detail, ok := result.Data.(SeriesDetailDTO)
	if !ok {
		t.Fatalf("unexpected payload: %+v", result.Data)
	}
	if detail.Series.SeriesName != "渲染管线" || detail.Series.ArticleCount != 2 {
		t.Fatalf("unexpected series payload: %+v", detail.Series)
	}
	if len(detail.Articles) != 2 || detail.Articles[0].Id != 1 {
		t.Fatalf("articles must keep the authored order: %+v", detail.Articles)
	}
}

func TestSeriesPublicDetailReportsMissingSeries(t *testing.T) {
	service := mustSeriesService(t, &fakeSeriesRepository{}, &seriesArticles{})

	result := service.GetPublicSeries(seriesContext(t, http.MethodGet, "/v1/public/series/9", "", gin.Params{{Key: "seriesId", Value: "9"}}))
	if result.Flag || result.Message != "数据不存在" {
		t.Fatalf("missing series must be reported: %+v", result)
	}
}

func TestSeriesSaveValidatesName(t *testing.T) {
	repo := &fakeSeriesRepository{}
	service := mustSeriesService(t, repo, &seriesArticles{})

	empty := service.SaveOrUpdateSeries(seriesContext(t, http.MethodPost, "/v1/admin/series", `{"seriesName":"  "}`, nil))
	if empty.Flag {
		t.Fatal("an empty series name must fail")
	}

	created := service.SaveOrUpdateSeries(seriesContext(t, http.MethodPost, "/v1/admin/series", `{"seriesName":" 渲染管线 ","seriesDesc":"desc"}`, nil))
	if !created.Flag {
		t.Fatalf("unexpected result: %+v", created)
	}
	if repo.saved.SeriesName != "渲染管线" {
		t.Fatalf("series name must be trimmed: %q", repo.saved.SeriesName)
	}
}

func TestSeriesDeleteReportsUnknownSeries(t *testing.T) {
	repo := &fakeSeriesRepository{}
	service := mustSeriesService(t, repo, &seriesArticles{})

	missing := service.DeleteSeries(seriesContext(t, http.MethodDelete, "/v1/admin/series", `[]`, nil))
	if missing.Flag {
		t.Fatal("an empty id list must fail")
	}
	ok := service.DeleteSeries(seriesContext(t, http.MethodDelete, "/v1/admin/series", `[4]`, nil))
	if !ok.Flag || len(repo.deleted) != 1 || repo.deleted[0] != 4 {
		t.Fatalf("unexpected delete result: %+v %v", ok, repo.deleted)
	}
}

func TestSeriesServiceRejectsMissingDependencies(t *testing.T) {
	if _, err := NewSeriesService(SeriesServiceDeps{}); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
