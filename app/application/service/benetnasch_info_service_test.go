package service

import (
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type aboutSiteRepository struct {
	about        string
	updatedAbout string
}

func (f *aboutSiteRepository) CountArticles(context.Context) (int64, error)   { return 0, nil }
func (f *aboutSiteRepository) CountCategories(context.Context) (int64, error) { return 0, nil }
func (f *aboutSiteRepository) CountTags(context.Context) (int64, error)       { return 0, nil }
func (f *aboutSiteRepository) CountTalks(context.Context) (int64, error)      { return 0, nil }
func (f *aboutSiteRepository) CountRecentContent(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (f *aboutSiteRepository) CountComments(context.Context, int) (int64, error) { return 0, nil }
func (f *aboutSiteRepository) CountUsers(context.Context) (int64, error)         { return 0, nil }
func (f *aboutSiteRepository) ListUniqueViews(context.Context, string, string) ([]port.UniqueView, error) {
	return nil, nil
}
func (f *aboutSiteRepository) ListArticleRank(context.Context, []int) ([]port.ArticleRank, error) {
	return nil, nil
}
func (f *aboutSiteRepository) GetWebsiteConfig(context.Context) (string, error)  { return "{}", nil }
func (f *aboutSiteRepository) UpdateWebsiteConfig(context.Context, string) error { return nil }
func (f *aboutSiteRepository) GetAbout(context.Context, int) (string, error)     { return f.about, nil }
func (f *aboutSiteRepository) UpdateAbout(_ context.Context, _ int, content string) error {
	f.updatedAbout = content
	f.about = content
	return nil
}

type aboutCaptureCache struct {
	fakeServiceCache
	cached string
	value  any
}

func (f *aboutCaptureCache) Get(_ context.Context, _ string) (string, error) {
	if f.cached == "" {
		return "", port.ErrCacheMiss
	}
	return f.cached, nil
}

func (f *aboutCaptureCache) Set(_ context.Context, _ string, value any, _ time.Duration) error {
	f.value = value
	return nil
}

func TestBenetnaschInfoServiceSerializesAboutForRepositoryAndCache(t *testing.T) {
	repository := &aboutSiteRepository{}
	cache := &aboutCaptureCache{}
	service := &MyBenetnaschInfoService{site: repository, cache: cache}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/about", strings.NewReader(`{"content":"# 关于我"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	result := service.UpdateAbout(serviceTestRequest{ginContextForServiceTest: c})
	if !result.Flag {
		t.Fatalf("unexpected update result: %+v", result)
	}

	var stored model.AboutDTO
	if err := json.Unmarshal([]byte(repository.updatedAbout), &stored); err != nil {
		t.Fatalf("repository value is not about JSON: %v", err)
	}
	if stored.Content != "# 关于我" {
		t.Fatalf("stored about content = %q", stored.Content)
	}
	cacheValue, ok := cache.value.(string)
	if !ok || cacheValue != repository.updatedAbout {
		t.Fatalf("cache value = %#v, repository value = %q", cache.value, repository.updatedAbout)
	}

	readResult := service.GetAbout(context.Background())
	about, ok := readResult.Data.(model.AboutDTO)
	if !readResult.Flag || !ok || about.Content != "# 关于我" {
		t.Fatalf("unexpected read result: %+v", readResult)
	}
}

func TestBenetnaschInfoServiceReadsLegacyPlainTextAbout(t *testing.T) {
	legacyContent := "[旧版关于我](https://example.com/about)"
	cache := &aboutCaptureCache{cached: legacyContent}
	service := &MyBenetnaschInfoService{
		site:  &aboutSiteRepository{about: legacyContent},
		cache: cache,
	}

	result := service.GetAbout(context.Background())
	about, ok := result.Data.(model.AboutDTO)
	if !result.Flag || !ok || about.Content != legacyContent {
		t.Fatalf("unexpected legacy read result: %+v", result)
	}
	normalized, ok := cache.value.(string)
	if !ok || normalized != `{"content":"[旧版关于我](https://example.com/about)"}` {
		t.Fatalf("legacy cache was not normalized: %#v", cache.value)
	}
}

func TestBenetnaschInfoServiceDoesNotCacheMalformedAbout(t *testing.T) {
	cache := &aboutCaptureCache{}
	service := &MyBenetnaschInfoService{
		site:  &aboutSiteRepository{about: `{"content":`},
		cache: cache,
	}

	result := service.GetAbout(context.Background())
	if result.Flag {
		t.Fatalf("malformed about should fail: %+v", result)
	}
	if cache.value != nil {
		t.Fatalf("malformed about was cached: %#v", cache.value)
	}
}
