package api

import (
	"context"
	"encoding/json"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/application/service"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/gin-gonic/gin"
)

type publicArticleReaderStub struct {
	err           error
	listQuery     service.PageQuery
	listCalls     int
	categoryQuery service.CategoryArticleQuery
	tagQuery      service.TagArticleQuery
	searchQuery   service.ArticleSearchQuery
	searchPage    service.ArticlePage[port.ArticleSearchHit]
	accessRequest service.ArticlePasswordAccess
	article       *port.Article
	articleID     int
	articleUserID int
	archivePage   service.ArticlePage[service.ArticleArchiveGroup]
}

func (s *publicArticleReaderStub) ListFeatured(context.Context) (service.FeaturedArticles, error) {
	return service.FeaturedArticles{}, s.err
}
func (s *publicArticleReaderStub) List(_ context.Context, query service.PageQuery) (service.ArticlePage[*port.ArticleCard], error) {
	s.listQuery = query
	s.listCalls++
	return service.ArticlePage[*port.ArticleCard]{Items: []*port.ArticleCard{{Id: 7}}, Total: 1, Page: query.Current, PageSize: query.Size}, s.err
}
func (s *publicArticleReaderStub) ListByCategory(_ context.Context, query service.CategoryArticleQuery) (service.ArticlePage[*port.ArticleCard], error) {
	s.categoryQuery = query
	return service.ArticlePage[*port.ArticleCard]{Items: []*port.ArticleCard{}, Page: query.Page.Current, PageSize: query.Page.Size}, s.err
}
func (s *publicArticleReaderStub) Get(_ context.Context, articleID, userID int) (*port.Article, error) {
	s.articleID, s.articleUserID = articleID, userID
	return s.article, s.err
}
func (s *publicArticleReaderStub) GrantPasswordAccess(_ context.Context, query service.ArticlePasswordAccess) error {
	s.accessRequest = query
	return s.err
}
func (s *publicArticleReaderStub) ListByTag(_ context.Context, query service.TagArticleQuery) (service.ArticlePage[*port.ArticleCard], error) {
	s.tagQuery = query
	return service.ArticlePage[*port.ArticleCard]{Items: []*port.ArticleCard{}, Page: query.Page.Current, PageSize: query.Page.Size}, s.err
}
func (s *publicArticleReaderStub) ListArchives(context.Context, service.PageQuery) (service.ArticlePage[service.ArticleArchiveGroup], error) {
	return s.archivePage, s.err
}
func (s *publicArticleReaderStub) Search(_ context.Context, query service.ArticleSearchQuery) (service.ArticlePage[port.ArticleSearchHit], error) {
	s.searchQuery = query
	if s.searchPage.Items == nil {
		return service.ArticlePage[port.ArticleSearchHit]{Items: []port.ArticleSearchHit{}, Page: query.Page.Current, PageSize: query.Page.Size}, s.err
	}
	return s.searchPage, s.err
}

func usePublicArticleReader(t *testing.T, reader service.PublicArticleReader) {
	t.Helper()
	previous := publicArticleReader
	publicArticleReader = reader
	t.Cleanup(func() { publicArticleReader = previous })
}

func TestListArticlesKeepsResultEnvelopeOnInvalidPagination(t *testing.T) {
	reader := &publicArticleReaderStub{}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/articles/all", ListArticles)
	req := httptest.NewRequest(http.MethodGet, "/articles/all?current=invalid&size=10", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP status: %d", recorder.Code)
	}
	var result model.ResultVO
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Flag || result.Message != "参数格式不正确" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if reader.listCalls != 0 {
		t.Fatal("invalid pagination called the application service")
	}
}

func TestListArticlesUsesUnifiedPaginationAndCanonicalPageEnvelope(t *testing.T) {
	reader := &publicArticleReaderStub{}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/articles", ListArticles)

	for _, test := range []struct {
		target        string
		current, size int
	}{
		{target: "/articles", current: 1, size: 12},
		{target: "/articles?current=3&size=500", current: 3, size: 100},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.target, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("unexpected HTTP status: %d", recorder.Code)
		}
		if reader.listQuery.Current != test.current || reader.listQuery.Size != test.size {
			t.Fatalf("query %q produced pagination %+v", test.target, reader.listQuery)
		}
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		page := body["data"].(map[string]any)
		if page["page"] != float64(test.current) || page["pageSize"] != float64(test.size) || page["total"] != float64(1) {
			t.Fatalf("unexpected page envelope: %#v", page)
		}
		if _, ok := page["items"]; !ok {
			t.Fatalf("canonical items field missing: %#v", page)
		}
		if _, ok := page["records"]; ok {
			t.Fatalf("legacy records field leaked: %#v", page)
		}
	}
}

func TestArticleDiscoveryHandlersNormalizeAliases(t *testing.T) {
	reader := &publicArticleReaderStub{}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/category", GetArticlesByCategoryId)
	router.GET("/tag", ListArticlesByTagId)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/category?name=+engineering+&categoryId=9", nil))
	if reader.categoryQuery.Name != "engineering" || reader.categoryQuery.ID != 0 || reader.categoryQuery.Page != (service.PageQuery{Current: 1, Size: 12}) {
		t.Fatalf("category aliases/default pagination were not normalized: %+v", reader.categoryQuery)
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/tag?tagId=17", nil))
	if reader.tagQuery.ID != 17 || reader.tagQuery.Name != "" || reader.tagQuery.Page.Size != 12 {
		t.Fatalf("tag id/default pagination were not preserved: %+v", reader.tagQuery)
	}
}

func TestAccessArticlePassesSessionAccountIDAndBodyArticleID(t *testing.T) {
	reader := &publicArticleReaderStub{}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userInfo", model.UserDetailsDTO{Id: 41, UserInfoId: 902})
		c.Next()
	})
	router.POST("/articles/:articleId/access", AccessArticle)
	request := httptest.NewRequest(http.MethodPost, "/articles/7/access", strings.NewReader(`{"articleId":7,"articlePassword":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP status: %d", recorder.Code)
	}
	if reader.accessRequest != (service.ArticlePasswordAccess{ArticleID: 7, Password: "secret", UserID: 41}) {
		t.Fatalf("unexpected access request: %+v", reader.accessRequest)
	}
}

func TestGetArticlePassesOptionalSessionAccountID(t *testing.T) {
	reader := &publicArticleReaderStub{article: &port.Article{Id: 7}}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userInfo", model.UserDetailsDTO{Id: 41, UserInfoId: 902})
		c.Next()
	})
	router.GET("/articles/:articleId", GetArticleById)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/articles/7", nil))
	if reader.articleID != 7 || reader.articleUserID != 41 {
		t.Fatalf("unexpected article identity input: id=%d user=%d", reader.articleID, reader.articleUserID)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "OK" || body["data"].(map[string]any)["id"] != float64(7) {
		t.Fatalf("unexpected article response: %#v", body)
	}
}

func TestSearchArticlesUsesSharedPaginationAndMapsHighlights(t *testing.T) {
	reader := &publicArticleReaderStub{searchPage: service.ArticlePage[port.ArticleSearchHit]{
		Items: []port.ArticleSearchHit{{
			ArticleSearch: port.ArticleSearch{Id: 7, ArticleTitle: "raw title"}, HighlightedTitle: "<mark>title</mark>",
		}}, Total: 4, Page: 1, PageSize: 12,
	}}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/search", ListArticlesBySearch)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/search?keywords=+title+", nil))
	if reader.searchQuery.Page != (service.PageQuery{Current: 1, Size: 12}) || reader.searchQuery.Keywords != " title " {
		t.Fatalf("unexpected search query: %+v", reader.searchQuery)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	page := body["data"].(map[string]any)
	items := page["items"].([]any)
	hit := items[0].(map[string]any)
	if page["total"] != float64(4) || hit["highlightedTitle"] != "<mark>title</mark>" || hit["articleTitle"] != "raw title" {
		t.Fatalf("search DTO mapping lost data: %#v", body)
	}
}

func TestListArchivesMapsTypedGroupsToCanonicalPage(t *testing.T) {
	reader := &publicArticleReaderStub{archivePage: service.ArticlePage[service.ArticleArchiveGroup]{
		Items: []service.ArticleArchiveGroup{{Time: "2026-9-27", Articles: []port.ArticleCard{{Id: 7}}}}, Total: 1, Page: 1, PageSize: 12,
	}}
	usePublicArticleReader(t, reader)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/archives", ListArchives)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/archives", nil))
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	page := body["data"].(map[string]any)
	items := page["items"].([]any)
	archive := items[0].(map[string]any)
	if archive["time"] != "2026-9-27" || page["total"] != float64(1) {
		t.Fatalf("typed archive response was not mapped: %#v", body)
	}
}

func TestPublicArticleFailuresKeepLegacyEnvelopeCodes(t *testing.T) {
	for _, test := range []struct {
		failure service.PublicArticleFailure
		code    int
		message string
	}{
		{failure: service.PublicArticleAccessDenied, code: 51000, message: "无权访问"},
		{failure: service.PublicArticlePasswordRequired, code: 52003, message: ""},
		{failure: service.PublicArticleNotFoundForAccess, code: 51000, message: "文章不存在"},
		{failure: service.PublicArticlePasswordInvalid, code: 51000, message: "密码错误"},
	} {
		result := publicArticleFailure(&service.PublicArticleError{Failure: test.failure})
		if result.Flag || result.Code != test.code || result.Message != test.message {
			t.Errorf("failure %q mapped to %+v", test.failure, result)
		}
	}
}
