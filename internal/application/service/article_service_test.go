package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"strings"
	"testing"
	"time"
)

type fakeArticleRepository struct {
	listErr         error
	listItems       []*port.ArticleCard
	featured        []*port.ArticleCard
	categoryItems   []*port.ArticleCard
	tagItems        []*port.ArticleCard
	article         port.Article
	articleErr      error
	preArticle      port.ArticleCard
	nextArticle     port.ArticleCard
	firstArticle    port.ArticleCard
	lastArticle     port.ArticleCard
	categoryNameArg string
	categoryIDArg   int
	tagNameArg      string
	tagIDArg        int
	pageCurrent     int
	pageSize        int
	archives        []port.ArticleCard
	related         []*port.ArticleCard
	record          entity.TArticle
	recordErr       error
	saveErr         error
	savedArticle    entity.TArticle
	saveCalls       int
	updateResult    entity.TArticle
	updateErr       error
	updateCalls     int
	trashCalls      int
	deleteCalls     int
	adminCount      int
	adminItems      []*port.ArticleAdmin
	adminFilter     port.ArticleFilter
	saveArticle     entity.TArticle
	saveCategory    string
	saveTags        []string
	adminArticle    entity.TArticle
	adminCategory   string
	adminTags       []string
	exportItems     []entity.TArticle
	exportIDs       []int
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
	allowed    bool
	accessErr  error
	grantErr   error
	accessKey  string
	accessItem string
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

func (c *recordingArticleCache) SIsMember(_ context.Context, key string, value any) (bool, error) {
	c.accessKey = key
	c.accessItem = fmt.Sprint(value)
	return c.allowed, c.accessErr
}

func (c *recordingArticleCache) SAdd(_ context.Context, key string, values ...any) (int64, error) {
	c.accessKey = key
	if len(values) > 0 {
		c.accessItem = fmt.Sprint(values[0])
	}
	if c.grantErr != nil {
		return 0, c.grantErr
	}
	return int64(len(values)), nil
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
	return f.featured, nil
}
func (f *fakeArticleRepository) ListArticles(_ context.Context, current, size int) ([]*port.ArticleCard, int, error) {
	f.pageCurrent, f.pageSize = current, size
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	if f.listItems != nil {
		return f.listItems, len(f.listItems), nil
	}
	return []*port.ArticleCard{{Id: 1, ArticleTitle: "test"}}, 1, nil
}
func (f *fakeArticleRepository) GetArticlesByCategoryName(_ context.Context, current, size int, name string) ([]*port.ArticleCard, int, error) {
	f.pageCurrent, f.pageSize, f.categoryNameArg = current, size, name
	return f.categoryItems, len(f.categoryItems), nil
}

func (f *fakeArticleRepository) ListArticlesByTagName(_ context.Context, current, size int, name string) ([]*port.ArticleCard, int, error) {
	f.pageCurrent, f.pageSize, f.tagNameArg = current, size, name
	return f.tagItems, len(f.tagItems), nil
}
func (f *fakeArticleRepository) GetArticlesByCategoryID(_ context.Context, current, size, id int) ([]*port.ArticleCard, int, error) {
	f.pageCurrent, f.pageSize, f.categoryIDArg = current, size, id
	f.categoryNameArg = ""
	return f.categoryItems, len(f.categoryItems), nil
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
	return f.article, f.articleErr
}
func (f *fakeArticleRepository) GetPreArticleByID(context.Context, int) (port.ArticleCard, error) {
	return f.preArticle, nil
}
func (f *fakeArticleRepository) GetNextArticleByID(context.Context, int) (port.ArticleCard, error) {
	return f.nextArticle, nil
}
func (f *fakeArticleRepository) GetFirstArticle(context.Context) (port.ArticleCard, error) {
	return f.firstArticle, nil
}
func (f *fakeArticleRepository) GetLastArticle(context.Context) (port.ArticleCard, error) {
	return f.lastArticle, nil
}
func (f *fakeArticleRepository) ListArticlesByTagID(_ context.Context, current, size, id int) ([]*port.ArticleCard, int, error) {
	f.pageCurrent, f.pageSize, f.tagIDArg = current, size, id
	f.tagNameArg = ""
	return f.tagItems, len(f.tagItems), nil
}
func (f *fakeArticleRepository) ListArchives(context.Context, int, int) ([]port.ArticleCard, int, error) {
	return f.archives, len(f.archives), nil
}
func (f *fakeArticleRepository) CountArticleAdmins(_ context.Context, filter port.ArticleFilter) (int, error) {
	f.adminFilter = filter
	return f.adminCount, nil
}
func (f *fakeArticleRepository) ListArticlesAdmin(_ context.Context, filter port.ArticleFilter) ([]*port.ArticleAdmin, error) {
	f.adminFilter = filter
	return f.adminItems, nil
}
func (f *fakeArticleRepository) ListArticleStatistics(context.Context) ([]port.ArticleStatistics, error) {
	return nil, nil
}
func (f *fakeArticleRepository) GetArticleSearchDocument(context.Context, int) (port.ArticleSearch, bool, error) {
	if f.record.Id == 0 || f.record.IsDelete != 0 || f.record.Status != 1 || f.record.ModerationStatus != "visible" {
		return port.ArticleSearch{}, false, f.recordErr
	}
	return port.ArticleSearch{
		Id: f.record.Id, UserId: f.record.UserId, ArticleCover: f.record.ArticleCover,
		ArticleTitle: f.record.ArticleTitle, ArticleContent: f.record.ArticleContent,
		IsDelete: f.record.IsDelete, Status: f.record.Status, ModerationStatus: f.record.ModerationStatus,
	}, true, nil
}
func (f *fakeArticleRepository) ListPublicArticleSearchDocuments(context.Context) ([]port.ArticleSearch, error) {
	document, public, err := f.GetArticleSearchDocument(context.Background(), f.record.Id)
	if err != nil || !public {
		return []port.ArticleSearch{}, err
	}
	return []port.ArticleSearch{document}, nil
}
func (f *fakeArticleRepository) GetArticleRecord(context.Context, int) (entity.TArticle, error) {
	return f.record, f.recordErr
}
func (f *fakeArticleRepository) SaveOrUpdate(_ context.Context, article entity.TArticle, category string, tags []string) (entity.TArticle, error) {
	f.saveCalls++
	f.saveArticle, f.saveCategory, f.saveTags = article, category, append([]string(nil), tags...)
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
	return f.adminArticle, f.adminCategory, f.adminTags, nil
}
func (f *fakeArticleRepository) Export(_ context.Context, ids []int) ([]entity.TArticle, error) {
	f.exportIDs = append([]int(nil), ids...)
	return f.exportItems, nil
}

func TestArticleReaderReturnsTypedListPage(t *testing.T) {
	repo := &fakeArticleRepository{listItems: []*port.ArticleCard{{Id: 1, ArticleTitle: "one"}, {Id: 2, ArticleTitle: "two"}}}
	page, err := mustArticleService(t, repo, nil).List(context.Background(), PageQuery{Current: 2, Size: 12})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || page.Page != 2 || page.PageSize != 12 || len(page.Items) != 2 {
		t.Fatalf("unexpected article page: %#v", page)
	}
	if repo.pageCurrent != 2 || repo.pageSize != 12 {
		t.Fatalf("pagination not forwarded: current=%d size=%d", repo.pageCurrent, repo.pageSize)
	}
}

func TestArticleReaderNormalizesEmptyCollectionsAndLimitsFeatured(t *testing.T) {
	repo := &fakeArticleRepository{
		listItems: []*port.ArticleCard{},
		featured:  []*port.ArticleCard{{Id: 1}, {Id: 2}, {Id: 3}, {Id: 4}},
	}
	service := mustArticleService(t, repo, nil)
	page, err := service.List(context.Background(), PageQuery{Current: 1, Size: 12})
	if err != nil {
		t.Fatal(err)
	}
	if page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("empty collection should be a non-nil empty slice: %#v", page.Items)
	}
	featured, err := service.ListFeatured(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if featured.TopArticle == nil || featured.TopArticle.Id != 1 || len(featured.FeaturedArticles) != 2 || featured.FeaturedArticles[1].Id != 3 {
		t.Fatalf("featured list exceeded or changed its existing limit: %#v", featured)
	}
}

func TestArticleReaderReturnsRepositoryFailure(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{
		listErr: apperrors.Unavailable("article.list", testServiceError("connection refused: password=secret")),
	}, nil)
	_, err := service.List(context.Background(), PageQuery{Current: 1, Size: 12})
	if err == nil || model.ResultFromError(err).Message != "系统繁忙，请稍后再试" {
		t.Fatalf("repository failure was not mapped safely: %v", err)
	}
}

func TestArticleReaderUsesCategoryAndTagNamesWhenProvided(t *testing.T) {
	repo := &fakeArticleRepository{categoryItems: []*port.ArticleCard{}, tagItems: []*port.ArticleCard{}}
	service := mustArticleService(t, repo, nil)
	page := PageQuery{Current: 1, Size: 12}
	if _, err := service.ListByCategory(context.Background(), CategoryArticleQuery{Page: page, ID: 8, Name: " Engineering "}); err != nil {
		t.Fatal(err)
	}
	if repo.categoryNameArg != "Engineering" || repo.categoryIDArg != 0 {
		t.Fatalf("category name did not take precedence: name=%q id=%d", repo.categoryNameArg, repo.categoryIDArg)
	}
	if _, err := service.ListByTag(context.Background(), TagArticleQuery{Page: page, ID: 9, Name: " Go "}); err != nil {
		t.Fatal(err)
	}
	if repo.tagNameArg != "Go" || repo.tagIDArg != 0 {
		t.Fatalf("tag name did not take precedence: name=%q id=%d", repo.tagNameArg, repo.tagIDArg)
	}
	if _, err := service.ListByCategory(context.Background(), CategoryArticleQuery{Page: page, ID: 8}); err != nil {
		t.Fatal(err)
	}
	if repo.categoryIDArg != 8 || repo.categoryNameArg != "" {
		t.Fatalf("category id fallback was not used: name=%q id=%d", repo.categoryNameArg, repo.categoryIDArg)
	}
	if _, err := service.ListByTag(context.Background(), TagArticleQuery{Page: page, ID: 9}); err != nil {
		t.Fatal(err)
	}
	if repo.tagIDArg != 9 || repo.tagNameArg != "" {
		t.Fatalf("tag id fallback was not used: name=%q id=%d", repo.tagNameArg, repo.tagIDArg)
	}
}

func TestArticleReaderGroupsArchivesByMonthNewestFirst(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{archives: []port.ArticleCard{
		{Id: 1, CreateTime: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		{Id: 2, CreateTime: time.Date(2025, 3, 4, 0, 0, 0, 0, time.UTC)},
		{Id: 3, CreateTime: time.Date(2024, 1, 8, 1, 0, 0, 0, time.UTC)},
	}}, nil)
	page, err := service.ListArchives(context.Background(), PageQuery{Current: 1, Size: 12})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].Time != "2025-3" || page.Items[1].Time != "2024-1" {
		t.Fatalf("unexpected archive order: %#v", page.Items)
	}
	if len(page.Items[1].Articles) != 2 {
		t.Fatalf("same-month articles were not grouped: %#v", page.Items)
	}
}

func TestArticleReaderUsesTypedSearchPort(t *testing.T) {
	searcher := &fakeArticleSearcher{hits: []port.ArticleSearchHit{{
		ArticleSearch:      port.ArticleSearch{Id: 7, ArticleTitle: "raw title", ArticleContent: "raw content"},
		HighlightedTitle:   "<mark>title</mark>",
		HighlightedContent: "<mark>content</mark>",
	}}, total: 7}
	service := mustArticleService(t, &fakeArticleRepository{}, searcher)
	page, err := service.Search(context.Background(), ArticleSearchQuery{Page: PageQuery{Current: 2, Size: 3}, Keywords: " title "})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 7 || page.Page != 2 || page.PageSize != 3 || len(page.Items) != 1 {
		t.Fatalf("unexpected search page: %#v", page)
	}
	hit := page.Items[0]
	if hit.ArticleTitle != "raw title" || hit.ArticleContent != "raw content" ||
		hit.HighlightedTitle != "<mark>title</mark>" || hit.HighlightedContent != "<mark>content</mark>" {
		t.Fatalf("raw or highlighted fields were not preserved: %#v", hit)
	}
	if searcher.gotOffset != 3 || searcher.gotLimit != 3 {
		t.Fatalf("pagination was not forwarded: offset=%d limit=%d", searcher.gotOffset, searcher.gotLimit)
	}
}

func TestArticleReaderEmptySearchSkipsSearcher(t *testing.T) {
	searcher := &fakeArticleSearcher{gotOffset: -1, gotLimit: -1}
	page, err := mustArticleService(t, &fakeArticleRepository{}, searcher).Search(context.Background(), ArticleSearchQuery{
		Page: PageQuery{Current: 2, Size: 9}, Keywords: "  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 0 || page.Page != 2 || page.PageSize != 9 || len(page.Items) != 0 {
		t.Fatalf("unexpected empty search page: %#v", page)
	}
	if searcher.gotOffset != -1 || searcher.gotLimit != -1 {
		t.Fatalf("empty keyword search called the searcher: offset=%d limit=%d", searcher.gotOffset, searcher.gotLimit)
	}
}

func TestArticleReaderCountsCachedPublicViews(t *testing.T) {
	cache := &recordingArticleCache{value: `{"id":7,"status":1,"isDelete":0,"articleTitle":"cached"}`}
	content := &recordingContentAnalyticsRepo{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: &fakeArticleRepository{}, Reactions: &fakeArticleReactionRepository{},
		ContentAnalytics: content, Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	article, err := service.Get(context.Background(), 7, 0)
	if err != nil || article == nil || cache.viewWrites != 1 || content.viewWrites != 1 {
		t.Fatalf("cached public article should count one view: article=%+v error=%v cache=%d analytics=%d", article, err, cache.viewWrites, content.viewWrites)
	}
}

func TestArticleReaderDoesNotCountNonPublicCachedViews(t *testing.T) {
	cache := &recordingArticleCache{value: `{"id":7,"status":3,"isDelete":0,"articleTitle":"draft"}`}
	content := &recordingContentAnalyticsRepo{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: &fakeArticleRepository{}, Reactions: &fakeArticleReactionRepository{},
		ContentAnalytics: content, Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	article, err := service.Get(context.Background(), 7, 0)
	if err != nil || article != nil || cache.viewWrites != 0 || content.viewWrites != 0 {
		t.Fatalf("non-public cached article must not be counted: article=%+v error=%v cache=%d analytics=%d", article, err, cache.viewWrites, content.viewWrites)
	}
}

func TestArticleReaderChecksPasswordGrantUsingAccountID(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 7, Status: 1, Password: "secret"}, article: port.Article{Id: 7, Status: 1}}
	cache := &recordingArticleCache{allowed: true}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: repo, Reactions: &fakeArticleReactionRepository{}, ContentAnalytics: fakeContentAnalyticsRepository{},
		Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	article, err := service.Get(context.Background(), 7, 42)
	if err != nil || article == nil {
		t.Fatalf("authorized article read failed: article=%+v error=%v", article, err)
	}
	if cache.accessKey != ArticleAccess+"42" || cache.accessItem != "7" {
		t.Fatalf("password grant lookup used the wrong key: key=%q item=%q", cache.accessKey, cache.accessItem)
	}
}

func TestArticleReaderRequiresPasswordGrant(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 7, Status: 1, Password: "secret"}}
	cache := &recordingArticleCache{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: repo, Reactions: &fakeArticleReactionRepository{}, ContentAnalytics: fakeContentAnalyticsRepository{},
		Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Get(context.Background(), 7, 42)
	var articleErr *PublicArticleError
	if !errors.As(err, &articleErr) || articleErr.Failure != PublicArticlePasswordRequired {
		t.Fatalf("expected password grant failure, got %v", err)
	}
}

func TestArticleReaderGrantsPasswordAccess(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 7, Password: "secret"}}
	cache := &recordingArticleCache{}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo: repo, Reactions: &fakeArticleReactionRepository{}, ContentAnalytics: fakeContentAnalyticsRepository{},
		Cache: cache, Storage: fakeServiceStorage{}, Search: &fakeArticleSearcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.GrantPasswordAccess(context.Background(), ArticlePasswordAccess{ArticleID: 7, Password: "secret", UserID: 42}); err != nil {
		t.Fatal(err)
	}
	if cache.accessKey != ArticleAccess+"42" || cache.accessItem != "7" {
		t.Fatalf("password grant used the wrong cache key: key=%q item=%q", cache.accessKey, cache.accessItem)
	}
	if err := service.GrantPasswordAccess(context.Background(), ArticlePasswordAccess{ArticleID: 7, Password: "wrong", UserID: 42}); err == nil {
		t.Fatal("wrong password was accepted")
	} else {
		var articleErr *PublicArticleError
		if !errors.As(err, &articleErr) || articleErr.Failure != PublicArticlePasswordInvalid {
			t.Fatalf("unexpected wrong-password error: %v", err)
		}
	}
}

func TestArticleReaderMapsSearchFailureAtHTTPBoundary(t *testing.T) {
	service := mustArticleService(t, &fakeArticleRepository{}, &fakeArticleSearcher{err: apperrors.Unavailable("search.articles", testServiceError("meili unavailable"))})
	_, err := service.Search(context.Background(), ArticleSearchQuery{Page: PageQuery{Current: 1, Size: 12}, Keywords: "title"})
	if result := model.ResultFromError(err); result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected mapped result: %+v", result)
	}
}

type testServiceError string

func (e testServiceError) Error() string { return string(e) }

func TestArticleAdminListReturnsTypedPageMetadata(t *testing.T) {
	repo := &fakeArticleRepository{
		adminCount: 31,
		adminItems: []*port.ArticleAdmin{{Id: 9, ArticleTitle: "draft"}},
	}
	filter := port.ArticleFilter{Current: 2, Size: 12, Keywords: "draft", Status: 3, Category: 4, Tag: 7}
	page, err := mustArticleService(t, repo, nil).ListAdminArticles(context.Background(), filter)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 31 || page.Page != 2 || page.PageSize != 12 || len(page.Items) != 1 || page.Items[0].Id != 9 {
		t.Fatalf("unexpected typed admin page: %#v", page)
	}
	if repo.adminFilter != filter {
		t.Fatalf("admin filter was not forwarded: got %+v want %+v", repo.adminFilter, filter)
	}
}

func TestArticleAdminSaveSanitizesHTMLAndMapsInput(t *testing.T) {
	repo := &fakeArticleRepository{}
	service := mustArticleService(t, repo, nil)
	err := service.SaveAdminArticle(context.Background(), ArticleSaveInput{
		Article: entity.TArticle{
			ArticleTitle: "A story", ArticleContentHTML: `<p>safe</p><script>alert(1)</script>`, Status: 3,
		},
		CategoryName: "Engineering", TagNames: []string{"go", "architecture"}, UserID: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if repo.saveCalls != 1 || repo.saveArticle.UserId != 7 || repo.saveArticle.ArticleContentHTML == "" || repo.saveArticle.ArticleContent != repo.saveArticle.ArticleContentHTML {
		t.Fatalf("article fields were not normalized before persistence: calls=%d article=%+v", repo.saveCalls, repo.saveArticle)
	}
	if strings.Contains(repo.saveArticle.ArticleContentHTML, "<script") || repo.saveCategory != "Engineering" || len(repo.saveTags) != 2 {
		t.Fatalf("unsafe or related article fields were not handled: article=%+v category=%q tags=%v", repo.saveArticle, repo.saveCategory, repo.saveTags)
	}
}

func TestArticleServiceRejectsNonOwnerMutations(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 9, UserId: 2, Status: 1}}
	service := mustArticleService(t, repo, nil)

	if err := service.SaveAdminArticle(context.Background(), ArticleSaveInput{
		Article: entity.TArticle{Id: 9, ArticleTitle: "forbidden", ArticleContent: "body", Status: 1}, UserID: 7,
	}); err == nil || !apperrors.IsKind(err, apperrors.KindForbidden) || repo.saveCalls != 0 {
		t.Fatalf("non-owner save must be forbidden before persistence: error=%v calls=%d", err, repo.saveCalls)
	}

	if err := service.TrashArticles(context.Background(), 7, []int{9}, 1); err == nil || !apperrors.IsKind(err, apperrors.KindForbidden) || repo.trashCalls != 0 {
		t.Fatalf("non-owner trash must be forbidden before persistence: error=%v calls=%d", err, repo.trashCalls)
	}

	if err := service.DeleteArticles(context.Background(), 7, []int{9}); err == nil || !apperrors.IsKind(err, apperrors.KindForbidden) || repo.deleteCalls != 0 {
		t.Fatalf("non-owner delete must be forbidden before persistence: error=%v calls=%d", err, repo.deleteCalls)
	}
}

func TestArticleServiceRejectsRecommendationForHiddenArticle(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{Id: 9, UserId: 2, Status: 1, ModerationStatus: "hidden"}}
	service := mustArticleService(t, repo, nil)
	err := service.SetArticleTopAndFeatured(context.Background(), 9, 0, 1)
	var adminErr *ArticleAdminError
	if !errors.As(err, &adminErr) || adminErr.Failure != ArticleAdminPublicRequired || repo.updateCalls != 0 {
		t.Fatalf("hidden article must not be recommended: error=%v calls=%d", err, repo.updateCalls)
	}
}
