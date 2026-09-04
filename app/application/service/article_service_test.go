package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeArticleRepository struct {
	listErr       error
	archives      []port.ArticleCard
	articleRecord entity.TArticle
}

type fakeArticleSearcher struct {
	hits []port.ArticleSearchHit
	err  error
}

type articleCacheStub struct {
	fakeServiceCache
	cached    string
	access    bool
	accessErr error
}

func (f *articleCacheStub) Get(context.Context, string) (string, error) {
	return f.cached, nil
}

func (f *articleCacheStub) SIsMember(context.Context, string, any) (bool, error) {
	return f.access, f.accessErr
}

func (f *fakeArticleSearcher) Search(context.Context, string) ([]port.ArticleSearchHit, error) {
	return f.hits, f.err
}

type fakeArticleModeSearcher struct {
	fakeArticleSearcher
	mode     port.SearchMode
	keywords string
	modeErr  error
}

func (f *fakeArticleModeSearcher) SearchWithMode(_ context.Context, keywords string, mode port.SearchMode) ([]port.ArticleSearchHit, error) {
	f.keywords = keywords
	f.mode = mode
	return f.hits, f.modeErr
}

type fakeArticleFilteredModeSearcher struct {
	fakeArticleModeSearcher
	filter port.KnowledgeFilter
}

func (f *fakeArticleFilteredModeSearcher) SearchWithModeAndFilter(_ context.Context, keywords string, mode port.SearchMode, filter port.KnowledgeFilter) ([]port.ArticleSearchHit, error) {
	f.keywords = keywords
	f.mode = mode
	f.filter = filter
	return f.hits, f.modeErr
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
	return f.articleRecord, nil
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

func articleTestContext() serviceTestRequest {
	return articleRequestContext("/articles?current=1&size=10")
}

func articleRequestContext(target string) serviceTestRequest {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return serviceTestRequest{ginContextForServiceTest: c}
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

func TestArticleServiceRejectsInvalidInjectedArticleVO(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, nil)
	c := articleRequestContext("/admin/articles")
	c.Set("articleVO", "unexpected value")

	result := service.SaveOrUpdateArticle(c)
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("unexpected result for invalid injected article value: %+v", result)
	}
}

func TestArticleServiceReportsArticleAccessFailureMessage(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{
		articleRecord: entity.TArticle{Id: 7, Status: 2},
	}, nil)
	c := articleRequestContext("/articles/7")
	c.Params = gin.Params{{Key: "articleId", Value: "7"}}
	c.Set("userInfo", model.UserDetailsDTO{Id: 42})

	result := service.GetArticleById(c)
	if result.Flag || result.Code != 52003 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Message != "文章密码认证未通过" {
		t.Fatalf("unexpected access failure message: %q", result.Message)
	}
}

func TestArticleServiceChecksCurrentVisibilityBeforeReadingCache(t *testing.T) {
	cached, err := json.Marshal(model.ArticleDTO{Id: 7, Status: port.PublicArticleStatus, ArticleContent: "cached private content"})
	if err != nil {
		t.Fatal(err)
	}
	service := mustArticleServiceWithCache(t, &fakeArticleRepository{
		articleRecord: entity.TArticle{Id: 7, Status: 2},
	}, nil, &articleCacheStub{cached: string(cached)})
	c := articleRequestContext("/articles/7")
	c.Params = gin.Params{{Key: "articleId", Value: "7"}}

	result := service.GetArticleById(c)
	if result.Flag || result.Message != "无权访问" {
		t.Fatalf("cached private article bypassed visibility check: %+v", result)
	}
}

func TestArticleServiceAllowsAuthorizedPrivateArticleCache(t *testing.T) {
	cached, err := json.Marshal(model.ArticleDTO{Id: 7, Status: 2, ArticleContent: "authorized cached content"})
	if err != nil {
		t.Fatal(err)
	}
	service := mustArticleServiceWithCache(t, &fakeArticleRepository{
		articleRecord: entity.TArticle{Id: 7, Status: 2},
	}, nil, &articleCacheStub{cached: string(cached), access: true})
	c := articleRequestContext("/articles/7")
	c.Params = gin.Params{{Key: "articleId", Value: "7"}}
	c.Set("userInfo", model.UserDetailsDTO{Id: 42})

	result := service.GetArticleById(c)
	if !result.Flag {
		t.Fatalf("authorized private article cache was rejected: %+v", result)
	}
	article, ok := result.Data.(model.ArticleDTO)
	if !ok || article.ArticleContent != "authorized cached content" {
		t.Fatalf("unexpected cached private article: %#v", result.Data)
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
		Source: &port.ArticleSearchSource{
			ArticleID: 7,
			Title:     "raw title",
			URL:       "/articles/7",
			ChunkID:   "article-7-chunk-0",
		},
		Relevance: &port.ArticleSearchRelevance{Score: 0.88, Mode: port.SearchModeHybrid, Index: "article_chunks_v1"},
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
	if hits[0].HighlightedTitle != "<mark>title</mark>" || hits[0].HighlightedContent != "<mark>content</mark>" || hits[0].Source == nil || hits[0].Source.URL != "/articles/7" || hits[0].Relevance == nil || hits[0].Relevance.Mode != port.SearchModeHybrid {
		t.Fatalf("search metadata was not mapped: %#v", hits[0])
	}
	payload, err := json.Marshal(hits[0])
	if err != nil {
		t.Fatalf("marshal search result: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(payload, &encoded); err != nil {
		t.Fatalf("decode search result: %v", err)
	}
	if encoded["articleTitle"] != "<mark>title</mark>" || encoded["highlightedTitle"] != "<mark>title</mark>" || encoded["source"] == nil || encoded["relevance"] == nil {
		t.Fatalf("unexpected enriched search JSON: %s", payload)
	}
}

func TestArticleServiceMapsSearchFailure(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, &fakeArticleSearcher{err: apperrors.Unavailable("search.articles", testServiceError("meili unavailable"))})
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=title"))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestArticleServiceForwardsExplicitSearchMode(t *testing.T) {
	searcher := &fakeArticleModeSearcher{fakeArticleSearcher: fakeArticleSearcher{hits: []port.ArticleSearchHit{{
		ArticleSearch: port.ArticleSearch{Id: 7, ArticleTitle: "title", ArticleContent: "content"},
	}}}}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=agent&mode=hybrid"))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if searcher.keywords != "agent" || searcher.mode != port.SearchModeHybrid {
		t.Fatalf("forwarded search = keywords %q, mode %q", searcher.keywords, searcher.mode)
	}
}

func TestArticleServiceRejectsInvalidSearchMode(t *testing.T) {
	searcher := &fakeArticleModeSearcher{}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=agent&mode=vector"))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("unexpected invalid mode result: %+v", result)
	}
	if searcher.keywords != "" || searcher.mode != "" {
		t.Fatal("searcher was called for invalid mode")
	}
}

func TestArticleServiceDoesNotFallbackNonKeywordModeToLegacySearcher(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, &fakeArticleSearcher{hits: []port.ArticleSearchHit{{
		ArticleSearch: port.ArticleSearch{Id: 7},
	}}})
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=agent&mode=semantic"))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected semantic capability result: %+v", result)
	}
}

func TestArticleServiceForwardsStructuredSearchFilters(t *testing.T) {
	searcher := &fakeArticleFilteredModeSearcher{fakeArticleModeSearcher: fakeArticleModeSearcher{
		fakeArticleSearcher: fakeArticleSearcher{hits: []port.ArticleSearchHit{{
			ArticleSearch: port.ArticleSearch{Id: 7},
		}}},
	}}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=agent&mode=keyword&category=Go&tags=agent,go&year=2024&from=2024-02-01&to=2024-02-29"))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if searcher.keywords != "agent" || searcher.mode != port.SearchModeKeyword || searcher.filter.Category != "Go" || searcher.filter.Year != 2024 || len(searcher.filter.Tags) != 2 {
		t.Fatalf("forwarded search filter = %#v", searcher.filter)
	}
	if !searcher.filter.From.Equal(time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)) || !searcher.filter.To.Equal(time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("forwarded date range = %v to %v", searcher.filter.From, searcher.filter.To)
	}
}

func TestArticleServiceRejectsInvalidSearchFilter(t *testing.T) {
	searcher := &fakeArticleFilteredModeSearcher{}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	result := service.ListArticlesBySearch(articleRequestContext("/articles/search?keywords=agent&year=not-a-year"))
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("unexpected invalid filter result: %+v", result)
	}
	if searcher.keywords != "" {
		t.Fatal("searcher was called for invalid filter")
	}
}

type testServiceError string

func (e testServiceError) Error() string { return string(e) }
