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
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
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
	profileGetResult    port.StudioProfile
	profileGetErr       error
	profileUpdateCalls  int
	profileHandle       string
	profileNickname     string
	profileIntro        string
	profileWebsite      string
	profileAbout        string
	profileLinks        []port.ProfileLink
	profileUpdateErr    error
	activationUpdate    port.StudioActivationUpdate
	activationResult    port.StudioActivation
	activationErr       error
	batchStatusCalls    int
	batchStatusKind     port.StudioContentType
	batchStatusScope    port.StudioBatchScope
	batchStatusActor    port.StudioAuditActor
	batchStatusValue    int
	batchStatusResult   port.StudioBatchMutation
	batchStatusErr      error
	batchDeleteCalls    int
	batchDeleteKind     port.StudioContentType
	batchDeleteScope    port.StudioBatchScope
	batchDeleteActor    port.StudioAuditActor
	batchDeleteResult   port.StudioBatchMutation
	batchDeleteErr      error
	previewResult       port.StudioBatchPreview
	previewErr          error
	retryCalls          int
	retryArticleID      int
	retryResult         port.ScheduledPublish
	retryErr            error
	feedSortCalls       []string
	authorSortCalls     []string
	authorArticleCalls  []string
	topicOverviewSizes  []int
	topicOverviewResult port.TopicOverview
	topicOverviewErr    error
}

func (f *fakePlatformRepository) ListFeedArticles(_ context.Context, _, _ int, featuredOnly bool) ([]*port.ArticleCard, int, error) {
	if featuredOnly {
		f.feedSortCalls = append(f.feedSortCalls, port.FeedSortFeatured)
	} else {
		f.feedSortCalls = append(f.feedSortCalls, port.FeedSortLatest)
	}
	return []*port.ArticleCard{}, 0, nil
}

func (f *fakePlatformRepository) ListFeedArticlesHot(_ context.Context, _, _ int) ([]*port.ArticleCard, int, error) {
	f.feedSortCalls = append(f.feedSortCalls, port.FeedSortHot)
	return []*port.ArticleCard{}, 0, nil
}

func (f *fakePlatformRepository) GetAuthorByHandle(_ context.Context, _ string, _ int) (port.AuthorCard, error) {
	return port.AuthorCard{PublicAuthor: port.PublicAuthor{Id: 11, Handle: "e2e-user", Nickname: "E2E User"}}, nil
}

func (f *fakePlatformRepository) ListAuthors(_ context.Context, _, _, _ int, sort string) ([]*port.AuthorCard, int, error) {
	f.authorSortCalls = append(f.authorSortCalls, sort)
	return []*port.AuthorCard{}, 0, nil
}

func (f *fakePlatformRepository) ListAuthorArticles(_ context.Context, _, _, _ int) ([]*port.ArticleCard, int, error) {
	f.authorArticleCalls = append(f.authorArticleCalls, port.FeedSortLatest)
	return []*port.ArticleCard{}, 0, nil
}

func (f *fakePlatformRepository) ListAuthorArticlesHot(_ context.Context, _, _, _ int) ([]*port.ArticleCard, int, error) {
	f.authorArticleCalls = append(f.authorArticleCalls, port.FeedSortHot)
	return []*port.ArticleCard{}, 0, nil
}

func (f *fakePlatformRepository) ListTopicOverview(_ context.Context, size int) (port.TopicOverview, error) {
	f.topicOverviewSizes = append(f.topicOverviewSizes, size)
	return f.topicOverviewResult, f.topicOverviewErr
}

func (f *fakePlatformRepository) PreviewOwnedContent(context.Context, int, port.StudioContentType, port.StudioFilter) (port.StudioBatchPreview, error) {
	return f.previewResult, f.previewErr
}

func (f *fakePlatformRepository) BatchUpdateOwnedContentStatus(_ context.Context, _ int, contentType port.StudioContentType, scope port.StudioBatchScope, status int, actor port.StudioAuditActor) (port.StudioBatchMutation, error) {
	f.batchStatusCalls++
	f.batchStatusKind = contentType
	f.batchStatusScope = scope
	f.batchStatusActor = actor
	f.batchStatusValue = status
	return f.batchStatusResult, f.batchStatusErr
}

func (f *fakePlatformRepository) BatchDeleteOwnedContent(_ context.Context, _ int, contentType port.StudioContentType, scope port.StudioBatchScope, actor port.StudioAuditActor) (port.StudioBatchMutation, error) {
	f.batchDeleteCalls++
	f.batchDeleteKind = contentType
	f.batchDeleteScope = scope
	f.batchDeleteActor = actor
	return f.batchDeleteResult, f.batchDeleteErr
}

func (f *fakePlatformRepository) RetryScheduledPublication(_ context.Context, _ int, articleID int, _ port.StudioAuditActor) (port.ScheduledPublish, error) {
	f.retryCalls++
	f.retryArticleID = articleID
	return f.retryResult, f.retryErr
}

func (f *fakePlatformRepository) GetStudioProfile(context.Context, int) (port.StudioProfile, error) {
	if f.profileGetErr != nil || f.profileGetResult.Handle != "" {
		return f.profileGetResult, f.profileGetErr
	}
	return port.StudioProfile{
		Handle: f.profileHandle, Nickname: f.profileNickname, Avatar: "https://cdn.example.test/avatar.png",
		Intro: f.profileIntro, Website: f.profileWebsite, About: f.profileAbout, Links: f.profileLinks,
	}, nil
}

func (f *fakePlatformRepository) SyncStudioActivation(_ context.Context, _ int, update port.StudioActivationUpdate) (port.StudioActivation, error) {
	f.activationUpdate = update
	return f.activationResult, f.activationErr
}

func (f *fakePlatformRepository) UpdateAuthorProfile(_ context.Context, _ int, profile port.StudioProfile) error {
	f.profileUpdateCalls++
	f.profileHandle = profile.Handle
	f.profileNickname = profile.Nickname
	f.profileIntro = profile.Intro
	f.profileWebsite = profile.Website
	f.profileAbout = profile.About
	f.profileLinks = profile.Links
	return f.profileUpdateErr
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

type recordingPlatformCache struct {
	fakeServiceCache
	deleted []string
}

func (c *recordingPlatformCache) Delete(_ context.Context, key string) error {
	c.deleted = append(c.deleted, key)
	return nil
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

func profileTestContext(method, target, body string) *gin.Context {
	ctx := platformTestContext(method, target, body)
	ctx.Set("userInfo", model.UserDetailsDTO{
		UserInfoId: 7, Handle: "old-handle", Nickname: "旧昵称", Avatar: "https://cdn.example.test/avatar.png",
		Intro: "旧简介", Website: "https://old.example.com",
	})
	return ctx
}

func TestStudioFilterParsesSeriesAndStatus(t *testing.T) {
	filter, err := studioFilter(platformTestContext(http.MethodGet, "/v1/studio/articles?status=3&seriesId=9&keywords=Go", ""))
	if err != nil {
		t.Fatal(err)
	}
	if filter.Status != 3 || filter.SeriesID != 9 || filter.Keywords != "Go" {
		t.Fatalf("unexpected filter: %+v", filter)
	}
}

func TestPlatformBatchStatusValidatesAndDeduplicates(t *testing.T) {
	repo := &fakePlatformRepository{batchStatusResult: port.StudioBatchMutation{Affected: 2, AuditID: 9}}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.BatchUpdateContentStatus(platformTestContext(
		http.MethodPut,
		"/v1/studio/content/batch-status",
		`{"kind":"article","scope":{"mode":"ids","ids":[3,3,5]},"visibility":"private"}`,
	))
	if !result.Flag || repo.batchStatusCalls != 1 || repo.batchStatusKind != port.StudioContentArticle || repo.batchStatusValue != 2 {
		t.Fatalf("unexpected batch status result: result=%+v repo=%+v", result, repo)
	}
	if len(repo.batchStatusScope.IDs) != 2 || repo.batchStatusScope.IDs[0] != 3 || repo.batchStatusScope.IDs[1] != 5 {
		t.Fatalf("unexpected ids: %v", repo.batchStatusScope.IDs)
	}
}

func TestPlatformBatchPreviewAndFilterScope(t *testing.T) {
	repo := &fakePlatformRepository{previewResult: port.StudioBatchPreview{Count: 23, MaxID: 99, HiddenCount: 2}}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	preview := service.BatchPreviewContent(platformTestContext(
		http.MethodPost,
		"/v1/studio/content/batch-preview",
		`{"kind":"article","status":3,"keywords":"Go"}`,
	))
	if !preview.Flag {
		t.Fatalf("preview failed: %+v", preview)
	}

	result := service.BatchUpdateContentStatus(platformTestContext(
		http.MethodPut,
		"/v1/studio/content/batch-status",
		`{"kind":"article","scope":{"mode":"filter","status":3,"keywords":"Go","maxId":99,"excludeIds":[8],"expectedCount":22},"visibility":"public"}`,
	))
	if !result.Flag || repo.batchStatusScope.Mode != port.StudioBatchScopeFilter || repo.batchStatusScope.MaxID != 99 || repo.batchStatusScope.ExpectedCount != 22 || len(repo.batchStatusScope.ExcludeIDs) != 1 {
		t.Fatalf("unexpected filter scope: result=%+v scope=%+v", result, repo.batchStatusScope)
	}
}

func TestPlatformBatchFilterInvalidatesReturnedArticleIDs(t *testing.T) {
	repo := &fakePlatformRepository{batchStatusResult: port.StudioBatchMutation{Affected: 2, AuditID: 9, ContentIDs: []int{11, 12}}}
	cache := &recordingPlatformCache{}
	service, err := NewPlatformService(PlatformServiceDeps{
		Repo: repo, Articles: &fakeArticleRepository{}, Storage: fakeServiceStorage{}, Cache: cache,
	})
	if err != nil {
		t.Fatal(err)
	}
	result := service.BatchUpdateContentStatus(platformTestContext(
		http.MethodPut,
		"/v1/studio/content/batch-status",
		`{"kind":"article","scope":{"mode":"filter","status":3,"maxId":99,"expectedCount":2},"visibility":"private"}`,
	))
	if !result.Flag {
		t.Fatalf("batch status failed: %+v", result)
	}
	if strings.Join(cache.deleted, ",") != "11,12" {
		t.Fatalf("unexpected cache invalidations: %v", cache.deleted)
	}
}
func TestPlatformBatchStatusRejectsScheduledAndInvalidKinds(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	scheduled := service.BatchUpdateContentStatus(platformTestContext(
		http.MethodPut,
		"/v1/studio/content/batch-status",
		`{"kind":"article","scope":{"mode":"ids","ids":[3]},"visibility":"scheduled"}`,
	))
	if scheduled.Flag || repo.batchStatusCalls != 0 {
		t.Fatalf("scheduled batch status must fail: result=%+v repo=%+v", scheduled, repo)
	}
	invalid := service.BatchUpdateContentStatus(platformTestContext(
		http.MethodPut,
		"/v1/studio/content/batch-status",
		`{"kind":"photo","scope":{"mode":"ids","ids":[3]},"visibility":"public"}`,
	))
	if invalid.Flag || repo.batchStatusCalls != 0 {
		t.Fatalf("invalid batch kind must fail: result=%+v repo=%+v", invalid, repo)
	}
}

func TestPlatformBatchDeleteUsesOwnedContentType(t *testing.T) {
	repo := &fakePlatformRepository{batchDeleteResult: port.StudioBatchMutation{Affected: 2, AuditID: 10}}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.BatchDeleteContent(platformTestContext(
		http.MethodDelete,
		"/v1/studio/content/batch",
		`{"kind":"series","scope":{"mode":"ids","ids":[7,8]}}`,
	))
	if !result.Flag || repo.batchDeleteCalls != 1 || repo.batchDeleteKind != port.StudioContentSeries {
		t.Fatalf("unexpected batch delete result: result=%+v repo=%+v", result, repo)
	}
	if len(repo.batchDeleteScope.IDs) != 2 || repo.batchDeleteScope.IDs[0] != 7 || repo.batchDeleteScope.IDs[1] != 8 {
		t.Fatalf("unexpected delete ids: %v", repo.batchDeleteScope.IDs)
	}
}

func TestPlatformSyncActivationUsesAuthenticatedAccount(t *testing.T) {
	repo := &fakePlatformRepository{activationResult: port.StudioActivation{StartedAt: "2026-09-24T10:00:00Z", Collapsed: true}}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)
	ctx := platformTestContext(http.MethodPut, "/v1/studio/activation", "{\"started\":true,\"collapsed\":true,\"identityComplete\":true,\"contentComplete\":false,\"profileVisited\":false,\"completed\":false}")
	ctx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	result := service.SyncActivation(ctx)
	if !result.Flag || !repo.activationUpdate.Started || !repo.activationUpdate.Collapsed || !repo.activationUpdate.IdentityComplete || repo.activationUpdate.ContentComplete {
		t.Fatalf("activation sync was not forwarded: result=%+v update=%+v", result, repo.activationUpdate)
	}
}

func TestPlatformGetProfileReturnsCurrentPublicIdentity(t *testing.T) {
	repo := &fakePlatformRepository{profileGetResult: port.StudioProfile{
		Handle: "old-handle", Nickname: "旧昵称", Avatar: "https://cdn.example.test/avatar.png",
		Intro: "旧简介", Website: "https://old.example.com",
	}}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.GetProfile(profileTestContext(http.MethodGet, "/v1/studio/profile", ""))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	profile, ok := result.Data.(model.StudioProfileDTO)
	if !ok {
		t.Fatalf("unexpected profile data: %#v", result.Data)
	}
	if profile.Handle != "old-handle" || profile.Nickname != "旧昵称" || profile.Avatar == "" || profile.Intro != "旧简介" || profile.Website != "https://old.example.com" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestPlatformUpdateProfileNormalizesAndReturnsIdentity(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.UpdateProfile(profileTestContext(
		http.MethodPut,
		"/v1/studio/profile",
		`{"handle":"  New-Handle  ","nickname":"  新昵称  ","intro":"  新的简介  ","website":"  https://example.com/about  "}`,
	))
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repo.profileUpdateCalls != 1 || repo.profileHandle != "new-handle" || repo.profileNickname != "新昵称" || repo.profileIntro != "新的简介" || repo.profileWebsite != "https://example.com/about" {
		t.Fatalf("unexpected profile update: %+v", repo)
	}
	profile, ok := result.Data.(model.StudioProfileDTO)
	if !ok || profile.Handle != "new-handle" || profile.Nickname != "新昵称" || profile.Avatar == "" {
		t.Fatalf("unexpected profile response: %#v", result.Data)
	}
}

func TestPlatformUpdateProfileRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "short handle", body: `{"handle":"ab","nickname":"昵称","intro":"","website":""}`},
		{name: "handle punctuation", body: `{"handle":"bad_handle","nickname":"昵称","intro":"","website":""}`},
		{name: "empty nickname", body: `{"handle":"valid-handle","nickname":"   ","intro":"","website":""}`},
		{name: "long nickname", body: `{"handle":"valid-handle","nickname":"` + strings.Repeat("名", 31) + `","intro":"","website":""}`},
		{name: "long intro", body: `{"handle":"valid-handle","nickname":"昵称","intro":"` + strings.Repeat("介", 256) + `","website":""}`},
		{name: "invalid website", body: `{"handle":"valid-handle","nickname":"昵称","intro":"","website":"javascript:alert(1)"}`},
		{name: "long website", body: `{"handle":"valid-handle","nickname":"昵称","intro":"","website":"https://example.com/` + strings.Repeat("a", 240) + `"}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := &fakePlatformRepository{}
			service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)
			result := service.UpdateProfile(profileTestContext(http.MethodPut, "/v1/studio/profile", testCase.body))
			if result.Flag || repo.profileUpdateCalls != 0 {
				t.Fatalf("invalid profile must fail before mutation: result=%+v calls=%d", result, repo.profileUpdateCalls)
			}
		})
	}
}

func TestPlatformUpdateProfileReportsHandleConflict(t *testing.T) {
	repo := &fakePlatformRepository{profileUpdateErr: apperrors.Conflict("platform.profile.handle", "handle already exists")}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	result := service.UpdateProfile(profileTestContext(http.MethodPut, "/v1/studio/profile", `{"handle":"taken-handle","nickname":"昵称","intro":"","website":""}`))
	if result.Flag || result.Message != "该 Handle 已被占用" {
		t.Fatalf("unexpected conflict result: %+v", result)
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

func TestDiscoveryFeedSortDefaultsToLatestAndRejectsUnknownValues(t *testing.T) {
	for _, value := range []string{"", "latest", "Latest", " hot "} {
		if _, ok := discoveryFeedSort(value); !ok {
			t.Fatalf("feed sort %q must be accepted", value)
		}
	}
	if got, _ := discoveryFeedSort(""); got != port.FeedSortLatest {
		t.Fatalf("empty feed sort must default to latest, got %q", got)
	}
	if got, _ := discoveryFeedSort(" HOT "); got != port.FeedSortHot {
		t.Fatalf("feed sort must be normalized, got %q", got)
	}
	if _, ok := discoveryFeedSort("trending"); ok {
		t.Fatal("unknown feed sort must be rejected")
	}
}

func TestDiscoveryAuthorSortDefaultsToArticlesAndRejectsUnknownValues(t *testing.T) {
	if got, _ := discoveryAuthorSort(""); got != port.AuthorSortArticles {
		t.Fatalf("empty author sort must default to articles, got %q", got)
	}
	for _, value := range []string{"followers", "active"} {
		if got, ok := discoveryAuthorSort(value); !ok || got != value {
			t.Fatalf("author sort %q must be accepted, got %q ok=%v", value, got, ok)
		}
	}
	if _, ok := discoveryAuthorSort("popular"); ok {
		t.Fatal("unknown author sort must be rejected")
	}
}

func TestPlatformFeedDispatchesSortWithoutBreakingFeaturedFilter(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	if result := service.Feed(platformTestContext(http.MethodGet, "/v1/public/feed", "")); !result.Flag {
		t.Fatalf("default feed must succeed: %+v", result)
	}
	if result := service.Feed(platformTestContext(http.MethodGet, "/v1/public/feed?sort=hot", "")); !result.Flag {
		t.Fatalf("hot feed must succeed: %+v", result)
	}
	if result := service.Feed(platformTestContext(http.MethodGet, "/v1/public/feed?featured=1", "")); !result.Flag {
		t.Fatalf("featured feed must succeed: %+v", result)
	}
	if result := service.Feed(platformTestContext(http.MethodGet, "/v1/public/feed?sort=trending", "")); result.Flag {
		t.Fatal("unknown feed sort must fail the request")
	}
	want := []string{port.FeedSortLatest, port.FeedSortHot, port.FeedSortFeatured}
	if strings.Join(repo.feedSortCalls, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected feed dispatch: %v", repo.feedSortCalls)
	}
}

func TestPlatformAuthorBoardDispatchesSort(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	for _, target := range []string{"/v1/public/authors", "/v1/public/authors?sort=followers", "/v1/public/authors?sort=active"} {
		if result := service.Authors(platformTestContext(http.MethodGet, target, "")); !result.Flag {
			t.Fatalf("author board %s must succeed: %+v", target, result)
		}
	}
	if result := service.Authors(platformTestContext(http.MethodGet, "/v1/public/authors?sort=popular", "")); result.Flag {
		t.Fatal("unknown author sort must fail the request")
	}
	want := []string{port.AuthorSortArticles, port.AuthorSortFollowers, port.AuthorSortActive}
	if strings.Join(repo.authorSortCalls, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected author dispatch: %v", repo.authorSortCalls)
	}
}

func TestPlatformAuthorArticlesSupportHotButNotFeatured(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	if result := service.AuthorArticles(platformTestContext(http.MethodGet, "/v1/public/authors/e2e-user/articles", "")); !result.Flag {
		t.Fatalf("author articles must succeed: %+v", result)
	}
	if result := service.AuthorArticles(platformTestContext(http.MethodGet, "/v1/public/authors/e2e-user/articles?sort=hot", "")); !result.Flag {
		t.Fatalf("hot author articles must succeed: %+v", result)
	}
	if result := service.AuthorArticles(platformTestContext(http.MethodGet, "/v1/public/authors/e2e-user/articles?sort=featured", "")); result.Flag {
		t.Fatal("featured is not a valid author article sort")
	}
	if strings.Join(repo.authorArticleCalls, ",") != port.FeedSortLatest+","+port.FeedSortHot {
		t.Fatalf("unexpected author article dispatch: %v", repo.authorArticleCalls)
	}
}

func TestPlatformTopicsClampRequestedSize(t *testing.T) {
	repo := &fakePlatformRepository{}
	service := mustPlatformService(t, repo, &fakeArticleRepository{}, nil)

	for _, target := range []string{
		"/v1/public/topics",
		"/v1/public/topics?size=3",
		"/v1/public/topics?size=999",
		"/v1/public/topics?size=abc",
		"/v1/public/topics?size=0",
	} {
		if result := service.Topics(platformTestContext(http.MethodGet, target, "")); !result.Flag {
			t.Fatalf("topics %s must succeed: %+v", target, result)
		}
	}
	want := []int{topicOverviewSizeDefault, 3, topicOverviewSizeMax, topicOverviewSizeDefault, topicOverviewSizeDefault}
	if len(repo.topicOverviewSizes) != len(want) {
		t.Fatalf("unexpected topics calls: %v", repo.topicOverviewSizes)
	}
	for index, size := range want {
		if repo.topicOverviewSizes[index] != size {
			t.Fatalf("topics call %d requested %d, want %d", index, repo.topicOverviewSizes[index], size)
		}
	}
}

func TestPlatformDiscoveryAttachesCountsWhenDependenciesAreWired(t *testing.T) {
	repo := &discoveryPlatformRepository{}
	service, err := NewPlatformService(PlatformServiceDeps{
		Repo: repo, Articles: &fakeArticleRepository{}, Storage: fakeServiceStorage{},
		Reactions: fakeReactionCounter{counts: map[int]port.ReactionCounts{5: {LikeCount: 3, FavoriteCount: 2}}},
		Comments:  fakeCommentCounter{counts: []*port.CommentCount{{Id: 9, CommentCount: 4}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result := service.Feed(platformTestContext(http.MethodGet, "/v1/public/feed?type=talk", "")); !result.Flag {
		t.Fatalf("talk feed must succeed: %+v", result)
	}
	if repo.talkCount == 0 {
		t.Fatal("talk feed must expose comment counts")
	}
}

type discoveryPlatformRepository struct {
	fakePlatformRepository
	talkCount int
}

func (d *discoveryPlatformRepository) ListFeedTalks(context.Context, int, int) ([]*port.Talk, int, error) {
	d.talkCount++
	return []*port.Talk{{Id: 9, Content: "hello"}}, 1, nil
}

type fakeReactionCounter struct {
	counts map[int]port.ReactionCounts
}

func (f fakeReactionCounter) Counts(context.Context, []int) (map[int]port.ReactionCounts, error) {
	return f.counts, nil
}

type fakeCommentCounter struct {
	counts []*port.CommentCount
}

func (f fakeCommentCounter) ListCommentCountsByTypeAndTopicIDs(context.Context, int, []int) ([]*port.CommentCount, error) {
	return f.counts, nil
}
