package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type fakeServiceCache struct{}

func (fakeServiceCache) Get(context.Context, string) (string, error) { return "", port.ErrCacheMiss }
func (fakeServiceCache) Set(context.Context, string, any, time.Duration) error {
	return nil
}
func (fakeServiceCache) SetNX(context.Context, string, any, time.Duration) (bool, error) {
	return true, nil
}
func (fakeServiceCache) IncrementWithExpiry(context.Context, string, time.Duration) (int64, error) {
	return 1, nil
}
func (fakeServiceCache) Expire(context.Context, string, time.Duration) (bool, error) {
	return true, nil
}
func (fakeServiceCache) Delete(context.Context, string) error { return nil }
func (fakeServiceCache) HGet(context.Context, string, string) (string, error) {
	return "", port.ErrCacheMiss
}
func (fakeServiceCache) HGetAll(context.Context, string) (map[string]string, error) {
	return map[string]string{}, nil
}
func (fakeServiceCache) HSet(context.Context, string, string, any, time.Duration) error {
	return nil
}
func (fakeServiceCache) HDel(context.Context, string, string) error { return nil }
func (fakeServiceCache) HIncrBy(context.Context, string, string, int64) (int64, error) {
	return 1, nil
}
func (fakeServiceCache) SIsMember(context.Context, string, any) (bool, error) {
	return false, nil
}
func (fakeServiceCache) SAdd(context.Context, string, ...any) (int64, error) {
	return 1, nil
}
func (fakeServiceCache) PFAdd(context.Context, string, ...any) (int64, error) {
	return 1, nil
}
func (fakeServiceCache) PFCount(context.Context, ...string) (int64, error) {
	return 0, nil
}
func (fakeServiceCache) IncrBy(context.Context, string, int64) (int64, error) {
	return 1, nil
}
func (fakeServiceCache) ZIncrBy(context.Context, string, float64, string) (float64, error) {
	return 1, nil
}
func (fakeServiceCache) ZScore(context.Context, string, string) (float64, error) {
	return 0, port.ErrCacheMiss
}
func (fakeServiceCache) ZRevRangeWithScores(context.Context, string, int64, int64) (map[string]float64, error) {
	return map[string]float64{}, nil
}
func (fakeServiceCache) ZRangeWithScores(context.Context, string) (map[string]float64, error) {
	return map[string]float64{}, nil
}

type fakeServiceStorage struct{}

func (fakeServiceStorage) Put(_ context.Context, key string, _ io.Reader) (port.ObjectRef, error) {
	return port.ObjectRef{Key: key, URL: "http://storage.test/" + key}, nil
}

type fakeServiceMailer struct{}

func (fakeServiceMailer) SendHTML(context.Context, port.EmailMessage) error { return nil }

type fakeServiceVisitor struct{}

func (fakeServiceVisitor) Resolve(context.Context, *http.Request) (port.VisitorIdentity, error) {
	return port.VisitorIdentity{}, nil
}

// fakeArticleReactionRepository keeps the article service constructible in
// tests; reaction behaviour itself is covered by the reaction service tests.
type fakeArticleReactionRepository struct {
	counts map[int]port.ReactionCounts
	states map[int]map[string]bool
}

func (f *fakeArticleReactionRepository) Toggle(context.Context, int, int, string) (bool, error) {
	return false, nil
}

func (f *fakeArticleReactionRepository) Counts(_ context.Context, articleIDs []int) (map[int]port.ReactionCounts, error) {
	result := make(map[int]port.ReactionCounts, len(articleIDs))
	for _, id := range articleIDs {
		if f != nil && f.counts != nil {
			result[id] = f.counts[id]
		}
	}
	return result, nil
}

func (f *fakeArticleReactionRepository) States(_ context.Context, _ int, articleIDs []int) (map[int]map[string]bool, error) {
	result := make(map[int]map[string]bool, len(articleIDs))
	for _, id := range articleIDs {
		if f != nil && f.states != nil && f.states[id] != nil {
			result[id] = f.states[id]
		}
	}
	return result, nil
}

func (f *fakeArticleReactionRepository) ListArticleIDsByUser(context.Context, int, string, int, int) ([]int, int64, error) {
	return nil, 0, nil
}

type fakeContentAnalyticsRepository struct{}

func (fakeContentAnalyticsRepository) RecordView(context.Context, int, time.Time) error {
	return nil
}

func (fakeContentAnalyticsRepository) RecordReadSession(context.Context, int, time.Time, int, int, int64) error {
	return nil
}

func (fakeContentAnalyticsRepository) RecordContinuationEvent(context.Context, int, time.Time, port.ContinuationEventType, *port.ContinuationTarget) error {
	return nil
}

func (fakeContentAnalyticsRepository) ListContinuationTargetMetrics(context.Context, string, string, int) ([]port.ContinuationTargetMetric, error) {
	return nil, nil
}
func (fakeContentAnalyticsRepository) ListDailyMetrics(context.Context, string, string) ([]port.ContentDailyMetric, error) {
	return nil, nil
}

func (fakeContentAnalyticsRepository) ListArticleMetrics(context.Context, string, string) ([]port.ContentArticleMetric, error) {
	return nil, nil
}

func (fakeContentAnalyticsRepository) GetArticleDailyMetrics(context.Context, int, string, string) ([]port.ContentDailyMetric, error) {
	return nil, nil
}

type fakeStellarBeaconInfoService struct{}

func (fakeStellarBeaconInfoService) Report(*http.Request) model.ResultVO { return model.ResultOk() }
func (fakeStellarBeaconInfoService) GetBlogHomeInfo(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) GetWebsiteConfig(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) GetBlogBackInfo(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) GetDashboardAnalytics(context.Context, string, string) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) UpdateWebsiteConfig(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) GetAbout(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) UpdateAbout(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) SaveBlogPhotoAlbumCover(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeStellarBeaconInfoService) listArticleRank(context.Context, map[string]float64) ([]model.ArticleRankDTO, error) {
	return nil, nil
}

func mustArticleService(t *testing.T, repo port.ArticleRepository, searcher port.ArticleSearcher) *MyArticleService {
	t.Helper()
	if searcher == nil {
		searcher = &fakeArticleSearcher{}
	}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo:             repo,
		Reactions:        &fakeArticleReactionRepository{},
		ContentAnalytics: fakeContentAnalyticsRepository{},
		Cache:            fakeServiceCache{},
		Storage:          fakeServiceStorage{},
		Search:           searcher,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func mustCommentService(t *testing.T, repo port.CommentRepository) *MyCommentService {
	t.Helper()
	service, err := NewCommentService(CommentServiceDeps{
		Repo:          repo,
		Website:       fakeStellarBeaconInfoService{},
		Users:         notificationUsers{},
		Articles:      &fakeArticleRepository{},
		Notifications: &fakeCommentNotifier{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func mustTalkService(t *testing.T, repo port.TalkRepository) *MyTalkService {
	t.Helper()
	service, err := NewTalkService(TalkServiceDeps{
		Repo:     repo,
		Comments: &fakeCommentRepository{},
		Storage:  fakeServiceStorage{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func mustUserAuthService(t *testing.T, repo port.AuthRepository) *MyUserAuthService {
	t.Helper()
	service, err := NewUserAuthService(UserAuthServiceDeps{
		Repo:    repo,
		Website: fakeStellarBeaconInfoService{},
		Cache:   fakeServiceCache{},
		Mailer:  fakeServiceMailer{},
		Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
