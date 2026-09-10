package service

import (
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"context"
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

type fakeBenetnaschInfoService struct{}

func (fakeBenetnaschInfoService) GetBenetnaschHomeInfo() model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) Report(*http.Request) model.ResultVO { return model.ResultOk() }
func (fakeBenetnaschInfoService) GetBlogHomeInfo(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) GetWebsiteConfig(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) GetBlogBackInfo(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) GetDashboardAnalytics(context.Context, string, string) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) UpdateWebsiteConfig(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) GetAbout(context.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) UpdateAbout(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) SaveBlogPhotoAlbumCover(*gin.Context) model.ResultVO {
	return model.ResultOk()
}
func (fakeBenetnaschInfoService) listArticleRank(context.Context, map[string]float64) ([]model.ArticleRankDTO, error) {
	return nil, nil
}

func mustArticleService(t *testing.T, repo port.ArticleRepository, searcher port.ArticleSearcher) *MyArticleService {
	t.Helper()
	if searcher == nil {
		searcher = &fakeArticleSearcher{}
	}
	service, err := NewArticleService(ArticleServiceDeps{
		Repo:    repo,
		Cache:   fakeServiceCache{},
		Storage: fakeServiceStorage{},
		Search:  searcher,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func mustCommentService(t *testing.T, repo port.CommentRepository) *MyCommentService {
	t.Helper()
	service, err := NewCommentService(CommentServiceDeps{
		Repo:    repo,
		Website: fakeBenetnaschInfoService{},
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
		Website: fakeBenetnaschInfoService{},
		Cache:   fakeServiceCache{},
		Mailer:  fakeServiceMailer{},
		Visitor: fakeServiceVisitor{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}
