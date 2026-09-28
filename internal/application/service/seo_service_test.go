package service

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

type seoTestPlatformRepository struct {
	port.PlatformRepository
	author             port.AuthorCard
	authorErr          error
	authorHandle       string
	hotArticles        []*port.ArticleCard
	hotErr             error
	latestArticles     []*port.ArticleCard
	latestErr          error
	authors            []*port.AuthorCard
	authorsTotal       int
	authorsErr         error
	authorsSort        string
	hotArticleCalls    int
	latestArticleCalls int
}

func (f *seoTestPlatformRepository) GetAuthorByHandle(_ context.Context, handle string, _ int) (port.AuthorCard, error) {
	f.authorHandle = handle
	return f.author, f.authorErr
}

func (f *seoTestPlatformRepository) ListAuthorArticlesHot(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	f.hotArticleCalls++
	return f.hotArticles, len(f.hotArticles), f.hotErr
}

func (f *seoTestPlatformRepository) ListAuthorArticles(context.Context, int, int, int) ([]*port.ArticleCard, int, error) {
	f.latestArticleCalls++
	return f.latestArticles, len(f.latestArticles), f.latestErr
}

func (f *seoTestPlatformRepository) ListAuthors(_ context.Context, _, _ int, _ int, sort string) ([]*port.AuthorCard, int, error) {
	f.authorsSort = sort
	return f.authors, f.authorsTotal, f.authorsErr
}

type seoTestArticleRepository struct {
	port.ArticleRepository
	archives    []port.ArticleCard
	archivesErr error
}

func (f *seoTestArticleRepository) ListArchives(context.Context, int, int) ([]port.ArticleCard, int, error) {
	return f.archives, len(f.archives), f.archivesErr
}

func newSeoTestService(t *testing.T, platform port.PlatformRepository, articles port.ArticleRepository) *MySeoService {
	t.Helper()
	previousBaseURL := config.PublicSiteURL
	config.PublicSiteURL = "https://beacon.example"
	t.Cleanup(func() { config.PublicSiteURL = previousBaseURL })
	service, err := NewSeoService(articles, platform)
	if err != nil {
		t.Fatalf("NewSeoService() error = %v", err)
	}
	return service
}

func seoTestContext(method, target string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, nil)
	context.Params = params
	return context, recorder
}

func TestRenderAuthorHTMLIncludesPublicProfileSignals(t *testing.T) {
	created := time.Date(2026, 9, 20, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	platform := &seoTestPlatformRepository{
		author: port.AuthorCard{
			PublicAuthor: port.PublicAuthor{
				Id: 7, Handle: "test-author", Nickname: `<测试 & 作者>`, Avatar: `https://cdn.example.test/avatar.png?x=1&y=2`,
				Intro: `<script>alert("x")</script> 公开简介`, Website: "https://example.com/about?a=1&b=2",
			},
			ArticleCount: 3, TalkCount: 2, SeriesCount: 1, CollectionCount: 4, FollowerCount: 5,
		},
		hotArticles: []*port.ArticleCard{{Id: 101, ArticleTitle: `<公开文章 & 第一篇>`, CreateTime: created}},
	}
	service := newSeoTestService(t, platform, &seoTestArticleRepository{})
	context, recorder := seoTestContext(http.MethodGet, "/u/test-author", gin.Params{{Key: "handle", Value: "test-author"}})

	service.RenderAuthorHTML(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		`<title>&lt;测试 &amp; 作者&gt; (@test-author) · Stellar Beacon</title>`,
		`<link rel="canonical" href="https://beacon.example/u/test-author">`,
		`<meta property="og:type" content="profile">`,
		`<meta property="og:image" content="https://cdn.example.test/avatar.png?x=1&amp;y=2">`,
		`"@type":"ProfilePage"`,
		`"@type":"Person"`,
		`<dd>公开文章</dd>`,
		`<a href="https://beacon.example/articles/101">&lt;公开文章 &amp; 第一篇&gt;</a>`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("body does not contain %q", expected)
		}
	}
	if strings.Contains(body, `<script>alert("x")</script>`) {
		t.Fatal("author intro was rendered without HTML escaping")
	}
	if platform.authorHandle != "test-author" {
		t.Fatalf("author handle = %q, want test-author", platform.authorHandle)
	}
	if platform.hotArticleCalls != 1 || platform.latestArticleCalls != 0 {
		t.Fatalf("article calls = hot:%d latest:%d, want hot only", platform.hotArticleCalls, platform.latestArticleCalls)
	}
}

func TestRenderAuthorHTMLFallsBackToLatestArticles(t *testing.T) {
	platform := &seoTestPlatformRepository{
		author:         port.AuthorCard{PublicAuthor: port.PublicAuthor{Id: 7, Handle: "test-author", Nickname: "测试作者"}},
		latestArticles: []*port.ArticleCard{{Id: 102, ArticleTitle: "最新文章", CreateTime: time.Now()}},
	}
	service := newSeoTestService(t, platform, &seoTestArticleRepository{})
	context, recorder := seoTestContext(http.MethodGet, "/u/test-author", gin.Params{{Key: "handle", Value: "test-author"}})

	service.RenderAuthorHTML(context)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "最新文章") {
		t.Fatalf("fallback response = %d %q", recorder.Code, recorder.Body.String())
	}
	if platform.hotArticleCalls != 1 || platform.latestArticleCalls != 1 {
		t.Fatalf("article calls = hot:%d latest:%d, want both", platform.hotArticleCalls, platform.latestArticleCalls)
	}
}

func TestRenderAuthorHTMLStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		authorErr  error
		wantStatus int
	}{
		{name: "not found", authorErr: apperrors.NotFound("seo.author"), wantStatus: http.StatusNotFound},
		{name: "repository unavailable", authorErr: stderrors.New("database unavailable"), wantStatus: http.StatusServiceUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newSeoTestService(t, &seoTestPlatformRepository{authorErr: test.authorErr}, &seoTestArticleRepository{})
			context, recorder := seoTestContext(http.MethodGet, "/u/missing", gin.Params{{Key: "handle", Value: "missing"}})

			service.RenderAuthorHTML(context)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestRenderSitemapIncludesPublicAuthors(t *testing.T) {
	published := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	platform := &seoTestPlatformRepository{
		authors: []*port.AuthorCard{
			{PublicAuthor: port.PublicAuthor{Id: 1, Handle: "first-author"}, LastPublishedAt: &published},
			{PublicAuthor: port.PublicAuthor{Id: 2, Handle: "second-author"}},
		},
		authorsTotal: 2,
	}
	articles := &seoTestArticleRepository{archives: []port.ArticleCard{{Id: 7, CreateTime: published}}}
	service := newSeoTestService(t, platform, articles)
	context, recorder := seoTestContext(http.MethodGet, "/sitemap.xml", nil)

	service.RenderSitemap(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		"https://beacon.example/articles/7",
		"https://beacon.example/authors",
		"https://beacon.example/u/first-author",
		"https://beacon.example/u/second-author",
		"<lastmod>2026-09-21</lastmod>",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("sitemap does not contain %q", expected)
		}
	}
	if platform.authorsSort != port.AuthorSortActive {
		t.Fatalf("author sort = %q, want %q", platform.authorsSort, port.AuthorSortActive)
	}
}

func TestRenderSitemapReturnsUnavailableWhenAuthorsFail(t *testing.T) {
	service := newSeoTestService(t, &seoTestPlatformRepository{authorsErr: stderrors.New("database unavailable")}, &seoTestArticleRepository{})
	context, recorder := seoTestContext(http.MethodGet, "/sitemap.xml", nil)

	service.RenderSitemap(context)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
