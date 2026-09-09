package service

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeArticleRepository struct {
	listErr  error
	archives []port.ArticleCard
}

type fakeArticleSearcher struct {
	hits []port.ArticleSearchHit
	err  error
}

func (f *fakeArticleSearcher) Search(context.Context, string) ([]port.ArticleSearchHit, error) {
	return f.hits, f.err
}

func (f *fakeArticleRepository) ListTopAndFeaturedArticles(context.Context) ([]*port.ArticleCard, error) {
	return nil, nil
}
func (f *fakeArticleRepository) ListArticles(context.Context, int, int) ([]*port.ArticleCard, int, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return []*port.ArticleCard{{Id: 1, ArticleTitle: "test"}}, 1, nil
}
func (f *fakeArticleRepository) GetArticlesByCategoryID(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepository) GetArticleByID(context.Context, int) (port.Article, error) {
	return port.Article{}, nil
}
func (f *fakeArticleRepository) GetPreArticleByID(context.Context, int) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepository) GetNextArticleByID(context.Context, int) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepository) GetFirstArticle(context.Context) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepository) GetLastArticle(context.Context) (port.ArticleCard, error) {
	return port.ArticleCard{}, nil
}
func (f *fakeArticleRepository) ListArticlesByTagID(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	return nil, 0, nil
}
func (f *fakeArticleRepository) ListArchives(context.Context, int, int) ([]port.ArticleCard, int, error) {
	return f.archives, len(f.archives), nil
}
func (f *fakeArticleRepository) CountArticleAdmins(context.Context, port.ArticleFilter) (int, error) {
	return 0, nil
}
func (f *fakeArticleRepository) ListArticlesAdmin(context.Context, port.ArticleFilter) ([]*port.ArticleAdmin, error) {
	return nil, nil
}
func (f *fakeArticleRepository) ListArticleStatistics(context.Context) ([]port.ArticleStatistics, error) {
	return nil, nil
}
func (f *fakeArticleRepository) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepository) SaveOrUpdate(context.Context, entity.TArticle, string, []string) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepository) UpdateTopAndFeatured(context.Context, int, int, int) (entity.TArticle, error) {
	return entity.TArticle{}, nil
}
func (f *fakeArticleRepository) UpdateDelete(context.Context, []int, int) error { return nil }
func (f *fakeArticleRepository) Delete(context.Context, []int) error            { return nil }
func (f *fakeArticleRepository) GetAdminArticle(context.Context, int) (entity.TArticle, string, []string, error) {
	return entity.TArticle{}, "", nil, nil
}
func (f *fakeArticleRepository) Export(context.Context, []int) ([]entity.TArticle, error) {
	return nil, nil
}

func articleTestContext() *gin.Context {
	return articleRequestContext("/articles?current=1&size=10")
}

func articleRequestContext(target string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return c
}

func TestArticleServiceListPropagatesRepositoryData(t *testing.T) {
	result := mustArticleService(t, &fakeArticleRepository{}, nil).ListArticles(articleTestContext())
	if !result.Flag || result.Code != 20000 {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(interface{})
	if !ok || page == nil {
		t.Fatal("expected page data")
	}
}

func TestArticleServiceMapsRepositoryFailureWithoutLeakingDetail(t *testing.T) {
	result := mustArticleService(t, &fakeArticleRepository{
		listErr: apperrors.Unavailable("article.list", testServiceError("connection refused: password=secret")),
	}, nil).ListArticles(articleTestContext())
	if result.Flag {
		t.Fatal("expected failed result")
	}
	if result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestArticleServiceRejectsInvalidArticleIDs(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, nil)
	for name, result := range map[string]model.ResultVO{
		"category": service.ListArticlesByCategoryId(articleRequestContext("/articles?current=1&size=10&categoryId=bad")),
		"tag":      service.ListArticlesByTagId(articleRequestContext("/articles?current=1&size=10&tagId=bad")),
	} {
		if result.Flag || result.Message != "参数格式不正确" {
			t.Fatalf("%s: unexpected result: %+v", name, result)
		}
	}
}

func TestArticleServiceSortsArchivesNewestFirst(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{archives: []port.ArticleCard{
		{Id: 1, CreateTime: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Id: 2, CreateTime: time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)},
		{Id: 3, CreateTime: time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC)},
	}}, nil)
	result := service.ListArchives(articleRequestContext("/archives/all?current=1&size=10"))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(model.PageResultDTO)
	if !ok {
		t.Fatalf("unexpected page type: %T", result.Data)
	}
	archives, ok := page.Records.([]model.ArchiveDTO)
	if !ok {
		t.Fatalf("unexpected archive type: %T", page.Records)
	}
	if len(archives) != 2 || archives[0].Time != "2025-3-4" || archives[1].Time != "2024-1-2" {
		t.Fatalf("unexpected archive order: %#v", archives)
	}
	if len(archives[1].Articles) != 2 {
		t.Fatalf("same-day articles were not grouped: %#v", archives[1].Articles)
	}
}

func TestArticleServiceUsesTypedSearchPort(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, &fakeArticleSearcher{hits: []port.ArticleSearchHit{{
		ArticleSearch:      port.ArticleSearch{Id: 7, ArticleTitle: "raw title", ArticleContent: "raw content"},
		HighlightedTitle:   "<mark>title</mark>",
		HighlightedContent: "<mark>content</mark>",
	}}})
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=title"))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	hits, ok := result.Data.([]model.ArticleSearchDTO)
	if !ok || len(hits) != 1 {
		t.Fatalf("unexpected search result: %#v", result.Data)
	}
	if hits[0].ArticleTitle != "<mark>title</mark>" || hits[0].ArticleContent != "<mark>content</mark>" {
		t.Fatalf("highlighted fields were not applied: %#v", hits[0])
	}
}

func TestArticleServiceMapsSearchFailure(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, &fakeArticleSearcher{err: apperrors.Unavailable("search.articles", testServiceError("meili unavailable"))})
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=title"))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

type testServiceError string

func (e testServiceError) Error() string { return string(e) }
