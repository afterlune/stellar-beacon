package service

import (
	"context"
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
