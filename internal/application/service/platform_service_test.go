package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type fakePlatformRepository struct {
	port.PlatformRepository
	moderateCalls       int
	moderateContentType string
	moderateID          int
	moderateAdminID     int
	moderateHidden      bool
	moderateReason      string
	moderateErr         error
}

func (f *fakePlatformRepository) ModerateContent(_ context.Context, contentType string, id, adminID int, hidden bool, reason string) error {
	f.moderateCalls++
	f.moderateContentType = contentType
	f.moderateID = id
	f.moderateAdminID = adminID
	f.moderateHidden = hidden
	f.moderateReason = reason
	return f.moderateErr
}

type fakeArticleDistributor struct {
	ids []int
	err error
}

func (f *fakeArticleDistributor) EnqueueArticle(_ context.Context, articleID int) error {
	f.ids = append(f.ids, articleID)
	return f.err
}

func platformTestContext(method, target, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	return c
}

func mustPlatformService(t *testing.T, repo *fakePlatformRepository, articles *fakeArticleRepository, newsletter articleDistributor) *MyPlatformService {
	t.Helper()
	service, err := NewPlatformService(PlatformServiceDeps{
		Repo: repo, Articles: articles, Newsletter: newsletter, Storage: fakeServiceStorage{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestPlatformModerationRequiresReasonWhenHiding(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.Moderate(platformTestContext(http.MethodPut, "/v1/admin/content/article/12/moderation", `{"contentType":"article","id":12,"hidden":true,"reason":"   "}`))
	if result.Flag || repo.moderateCalls != 0 {
		t.Fatalf("hiding without a reason must be rejected: result=%+v calls=%d", result, repo.moderateCalls)
	}
}

func TestPlatformModerationRejectsOverlongReason(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)
	body := `{"contentType":"article","id":12,"hidden":true,"reason":"` + strings.Repeat("a", 256) + `"}`

	result := service.Moderate(platformTestContext(http.MethodPut, "/v1/admin/content/article/12/moderation", body))
	if result.Flag || repo.moderateCalls != 0 {
		t.Fatalf("overlong moderation reason must be rejected: result=%+v calls=%d", result, repo.moderateCalls)
	}
}

func TestPlatformModerationTrimsReasonAndContentType(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.Moderate(platformTestContext(http.MethodPut, "/v1/admin/content/article/12/moderation", `{"contentType":" article ","id":12,"hidden":true,"reason":"  违规内容  "}`))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repo.moderateCalls != 1 || repo.moderateContentType != "article" || repo.moderateReason != "违规内容" || repo.moderateID != 12 || repo.moderateAdminID != 7 || !repo.moderateHidden {
		t.Fatalf("unexpected moderation call: %+v", repo)
	}
}

func TestPlatformDistributionOnlyAllowsPublicVisibleArticles(t *testing.T) {
	repo := &fakePlatformRepository{}
	articles := &fakeArticleRepository{record: entity.TArticle{Id: 12, Status: 1, ModerationStatus: "hidden"}}
	service := mustPlatformService(t, repo, articles, &fakeArticleDistributor{})
	ctx := platformTestContext(http.MethodPut, "/v1/admin/content/articles/12/distribution", `{"featured":true,"newsletter":false}`)
	ctx.Params = gin.Params{{Key: "articleId", Value: "12"}}

	result := service.DistributeArticle(ctx)
	if result.Flag || articles.updateCalls != 0 {
		t.Fatalf("hidden article must not be distributed: result=%+v calls=%d", result, articles.updateCalls)
	}
}

func TestPlatformDistributionRequiresNewsletterService(t *testing.T) {
	repo := &fakePlatformRepository{}
	articles := &fakeArticleRepository{record: entity.TArticle{Id: 12, Status: 1, ModerationStatus: "visible"}}
	service := mustPlatformService(t, repo, articles, nil)
	ctx := platformTestContext(http.MethodPut, "/v1/admin/content/articles/12/distribution", `{"featured":false,"newsletter":true}`)
	ctx.Params = gin.Params{{Key: "articleId", Value: "12"}}

	result := service.DistributeArticle(ctx)
	if result.Flag || articles.updateCalls != 0 {
		t.Fatalf("newsletter distribution without a service must fail before mutation: result=%+v calls=%d", result, articles.updateCalls)
	}
}

func TestPlatformDistributionUpdatesAndEnqueuesVisibleArticle(t *testing.T) {
	repo := &fakePlatformRepository{}
	articles := &fakeArticleRepository{
		record:       entity.TArticle{Id: 12, Status: 1, ModerationStatus: "visible"},
		updateResult: entity.TArticle{Id: 12},
	}
	newsletter := &fakeArticleDistributor{}
	service := mustPlatformService(t, repo, articles, newsletter)
	ctx := platformTestContext(http.MethodPut, "/v1/admin/content/articles/12/distribution", `{"featured":true,"newsletter":true}`)
	ctx.Params = gin.Params{{Key: "articleId", Value: "12"}}

	result := service.DistributeArticle(ctx)
	if !result.Flag || articles.updateCalls != 1 || len(newsletter.ids) != 1 || newsletter.ids[0] != 12 {
		t.Fatalf("unexpected distribution result: result=%+v updates=%d enqueues=%v", result, articles.updateCalls, newsletter.ids)
	}
}

type recordingObjectStorage struct {
	key string
}

func (s *recordingObjectStorage) Put(_ context.Context, key string, _ io.Reader) (port.ObjectRef, error) {
	s.key = key
	return port.ObjectRef{Key: key, URL: "https://cdn.example.test/" + key}, nil
}

func uploadTestContext(t *testing.T, kind, filename string) *gin.Context {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/studio/uploads?kind="+kind, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

func TestPlatformUploadUsesContentSpecificPrefixes(t *testing.T) {
	cases := []struct {
		kind   string
		prefix string
	}{
		{kind: "article-cover", prefix: "articles/covers/"},
		{kind: "article-inline", prefix: "articles/inline/"},
		{kind: "talk-image", prefix: "talks/"},
		{kind: "series-cover", prefix: "series/covers/"},
		{kind: "avatar", prefix: "avatar/"},
		{kind: "cover", prefix: "articles/covers/"},
		{kind: "talk", prefix: "talks/"},
	}
	for _, testCase := range cases {
		storage := &recordingObjectStorage{}
		service, err := NewPlatformService(PlatformServiceDeps{
			Repo: &fakePlatformRepository{}, Articles: &fakeArticleRepository{}, Storage: storage,
		})
		if err != nil {
			t.Fatal(err)
		}
		result := service.Upload(uploadTestContext(t, testCase.kind, "example.png"))
		if !result.Flag || !strings.HasPrefix(storage.key, testCase.prefix) {
			t.Fatalf("kind %q prefix: result=%+v key=%q", testCase.kind, result, storage.key)
		}
	}
}

func TestPlatformStudioSaveRejectsDatabaseIncompatibleFields(t *testing.T) {
	service := mustPlatformService(t, &fakePlatformRepository{}, &fakeArticleRepository{}, nil)

	article := service.SaveOwnedArticle(platformTestContext(
		http.MethodPost,
		"/v1/studio/articles",
		`{"articleTitle":"`+strings.Repeat("a", 51)+`","articleContent":"body","visibility":"draft","type":1}`,
	))
	if article.Flag {
		t.Fatalf("overlong article title must fail: %+v", article)
	}

	talk := service.SaveOwnedTalk(platformTestContext(
		http.MethodPost,
		"/v1/studio/talks",
		`{"content":"`+strings.Repeat("a", 2001)+`","images":"[]","visibility":"public"}`,
	))
	if talk.Flag {
		t.Fatalf("overlong talk content must fail: %+v", talk)
	}

	series := service.SaveOwnedSeries(platformTestContext(
		http.MethodPost,
		"/v1/studio/series",
		`{"seriesName":"`+strings.Repeat("a", 51)+`","seriesDesc":"","cover":"","visibility":"draft"}`,
	))
	if series.Flag {
		t.Fatalf("overlong series name must fail: %+v", series)
	}
}
