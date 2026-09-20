package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeArticleRepository struct {
	listErr      error
	archives     []port.ArticleCard
	related      []*port.ArticleCard
	record       entity.TArticle
	recordErr    error
	saveErr      error
	savedArticle entity.TArticle
	saveCalls    int
	updateResult entity.TArticle
	updateErr    error
	updateCalls  int
	trashCalls   int
	deleteCalls  int
}

type fakeArticleSearcher struct {
	hits      []port.ArticleSearchHit
	total     int64
	err       error
	gotOffset int
	gotLimit  int
}

type recordingArticleCache struct {
	fakeServiceCache
	value      string
	viewWrites int
}

func (c *recordingArticleCache) Get(context.Context, string) (string, error) {
	if c.value == "" {
		return "", port.ErrCacheMiss
	}
	return c.value, nil
}

func (c *recordingArticleCache) ZIncrBy(context.Context, string, float64, string) (float64, error) {
	c.viewWrites++
	return float64(c.viewWrites), nil
}

type recordingContentAnalyticsRepo struct {
	fakeContentAnalyticsRepository
	viewWrites int
}

func (r *recordingContentAnalyticsRepo) RecordView(context.Context, int, time.Time) error {
	r.viewWrites++
	return nil
}

func (f *fakeArticleSearcher) Search(_ context.Context, _ string, offset, limit int) (port.ArticleSearchPage, error) {
	f.gotOffset = offset
	f.gotLimit = limit
	total := f.total
	if total == 0 {
		total = int64(len(f.hits))
	}
	return port.ArticleSearchPage{Hits: f.hits, Total: total}, f.err
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
func (f *fakeArticleRepository) ListArticleCardsByIDs(context.Context, []int) ([]*port.ArticleCard, error) {
	return nil, nil
}
func (f *fakeArticleRepository) ListArticleCardsBySeries(context.Context, int) ([]*port.ArticleCard, error) {
	return nil, nil
}
func (f *fakeArticleRepository) ListRelatedArticles(context.Context, int, int, int, int) ([]*port.ArticleCard, error) {
	return f.related, nil
}
func (f *fakeArticleRepository) PublishDueArticles(context.Context) ([]int, error) {
	return nil, nil
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
	return f.record, f.recordErr
}
func (f *fakeArticleRepository) SaveOrUpdate(context.Context, entity.TArticle, string, []string) (entity.TArticle, error) {
	f.saveCalls++
	return f.savedArticle, f.saveErr
}
func (f *fakeArticleRepository) UpdateTopAndFeatured(context.Context, int, int, int) (entity.TArticle, error) {
	f.updateCalls++
	return f.updateResult, f.updateErr
}
func (f *fakeArticleRepository) UpdateDelete(context.Context, []int, int) error {
	f.trashCalls++
	return nil
}
func (f *fakeArticleRepository) Delete(context.Context, []int) error {
	f.deleteCalls++
	return nil
}

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
	if result.Data == nil {
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
	searcher := &fakeArticleSearcher{hits: []port.ArticleSearchHit{{
		ArticleSearch:      port.ArticleSearch{Id: 7, ArticleTitle: "raw title", ArticleContent: "raw content"},
		HighlightedTitle:   "<mark>title</mark>",
		HighlightedContent: "<mark>content</mark>",
	}}, total: 7}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=title&current=2&size=3"))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(model.PageResultDTO)
	if !ok || page.Count != 7 || page.Page != 2 || page.PageSize != 3 {
		t.Fatalf("unexpected search page: %#v", result.Data)
	}
	hits, ok := page.Records.([]model.ArticleSearchDTO)
	if !ok || len(hits) != 1 {
		t.Fatalf("unexpected search result: %#v", page.Records)
	}
	if hits[0].ArticleTitle != "<mark>title</mark>" || hits[0].ArticleContent != "<mark>content</mark>" {
		t.Fatalf("highlighted fields were not applied: %#v", hits[0])
	}
	if searcher.gotOffset != 3 || searcher.gotLimit != 3 {
		t.Fatalf("pagination was not forwarded: offset=%d limit=%d", searcher.gotOffset, searcher.gotLimit)
	}
}

func TestArticleServiceSearchNormalizesPagingAndEmptyKeywords(t *testing.T) {
	searcher := &fakeArticleSearcher{}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=title&current=-4&size=500"))
	page, ok := result.Data.(model.PageResultDTO)
	if !ok || page.Page != 1 || page.PageSize != 50 {
		t.Fatalf("paging was not clamped: %#v", result.Data)
	}
	if searcher.gotOffset != 0 || searcher.gotLimit != 50 {
		t.Fatalf("clamped paging was not forwarded: offset=%d limit=%d", searcher.gotOffset, searcher.gotLimit)
	}

	searcher.gotOffset = -1
	searcher.gotLimit = -1
	empty := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=&current=2&size=9"))
	emptyPage, ok := empty.Data.(model.PageResultDTO)
	if !ok || emptyPage.Count != 0 || emptyPage.Page != 2 || emptyPage.PageSize != 9 {
		t.Fatalf("empty search page = %#v", empty.Data)
	}
	if searcher.gotOffset != -1 || searcher.gotLimit != -1 {
		t.Fatalf("empty keyword search should not call the searcher: offset=%d limit=%d", searcher.gotOffset, searcher.gotLimit)
	}
}

func TestArticleServiceCountsCachedPublicViews(t *testing.T) {
	cache := &recordingArticleCache{value: `{"id":7,"status":1,"isDelete":0,"articleTitle":"cached"}`}
	content := &recordingContentAnalyticsRepo{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: &fakeArticleRepository{}, Reactions: &fakeArticleReactionRepository{},
		ContentAnalytics: content, Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := articleRequestContext("/v1/public/articles/7")
	ctx.Params = gin.Params{{Key: "articleId", Value: "7"}}
	result := service.GetArticleById(ctx)
	if !result.Flag || cache.viewWrites != 1 || content.viewWrites != 1 {
		t.Fatalf("cached public article should count one view: result=%+v cache=%d analytics=%d", result, cache.viewWrites, content.viewWrites)
	}
}

func TestArticleServiceDoesNotCountNonPublicCachedViews(t *testing.T) {
	cache := &recordingArticleCache{value: `{"id":7,"status":3,"isDelete":0,"articleTitle":"draft"}`}
	content := &recordingContentAnalyticsRepo{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: &fakeArticleRepository{}, Reactions: &fakeArticleReactionRepository{},
		ContentAnalytics: content, Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := articleRequestContext("/v1/public/articles/7")
	ctx.Params = gin.Params{{Key: "articleId", Value: "7"}}
	_ = service.GetArticleById(ctx)
	if cache.viewWrites != 0 || content.viewWrites != 0 {
		t.Fatalf("non-public cached article must not count views: cache=%d analytics=%d", cache.viewWrites, content.viewWrites)
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

func articleJSONContext(method, target, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestArticleServiceRejectsNonOwnerMutations(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 9, UserId: 2, Status: 1}}
	service := mustArticleService(t, repo, nil)

	saveCtx := articleJSONContext(http.MethodPost, "/v1/admin/articles", `{"id":9,"articleTitle":"forbidden","articleContent":"body","status":1}`)
	saveCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := service.SaveOrUpdateArticle(saveCtx); result.Flag || repo.saveCalls != 0 {
		t.Fatalf("non-owner save must be forbidden before persistence: result=%+v calls=%d", result, repo.saveCalls)
	}

	trashCtx := articleJSONContext(http.MethodPut, "/v1/admin/articles/trash", `{"ids":[9],"isDelete":1}`)
	trashCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := service.UpdateArticleDelete(trashCtx); result.Flag || repo.trashCalls != 0 {
		t.Fatalf("non-owner trash must be forbidden before persistence: result=%+v calls=%d", result, repo.trashCalls)
	}

	deleteCtx := articleJSONContext(http.MethodDelete, "/v1/admin/articles/batch-delete", `[9]`)
	deleteCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := service.DeleteArticles(deleteCtx); result.Flag || repo.deleteCalls != 0 {
		t.Fatalf("non-owner delete must be forbidden before persistence: result=%+v calls=%d", result, repo.deleteCalls)
	}
}

func TestArticleServiceRejectsRecommendationForHiddenArticle(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 9, UserId: 2, Status: 1, ModerationStatus: "hidden"}}
	service := mustArticleService(t, repo, nil)
	ctx := articleJSONContext(http.MethodPut, "/v1/admin/articles/featured", `{"id":9,"isTop":0,"isFeatured":1}`)
	result := service.UpdateArticleTopAndFeatured(ctx)
	if result.Flag || repo.updateCalls != 0 {
		t.Fatalf("hidden article must not be recommended: result=%+v calls=%d", result, repo.updateCalls)
	}
}
