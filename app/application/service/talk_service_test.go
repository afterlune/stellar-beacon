package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeTalkRepository struct {
	err error
}

func (f *fakeTalkRepository) Count(_ context.Context, _ port.TalkFilter) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return 1, nil
}

func (f *fakeTalkRepository) List(context.Context, int, int) ([]*port.Talk, error) {
	return []*port.Talk{{Id: 1, Content: "hello"}}, nil
}

func (f *fakeTalkRepository) Get(_ context.Context, _ int) (port.Talk, error) {
	return port.Talk{}, f.err
}

func (f *fakeTalkRepository) ListAdmin(context.Context, int, int, port.TalkFilter) ([]*port.TalkAdmin, error) {
	return nil, nil
}

func (f *fakeTalkRepository) GetAdmin(context.Context, int) (port.TalkAdmin, error) {
	return port.TalkAdmin{}, nil
}

func (f *fakeTalkRepository) SaveOrUpdate(context.Context, entity.TTalk) error { return nil }
func (f *fakeTalkRepository) Delete(context.Context, []int) error              { return nil }

func talkTestContext(path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	return c
}

func TestTalkServiceMapsRepositoryFailure(t *testing.T) {
	result := NewTalkService(&fakeTalkRepository{
		err: apperrors.Unavailable("talk.count", testServiceError("database password=secret")),
	}).ListTalks(talkTestContext("/talks?current=1&size=10"))
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTalkServiceKeepsNotFoundMessage(t *testing.T) {
	c := talkTestContext("/talks/1")
	c.Params = gin.Params{{Key: "talkId", Value: "1"}}
	result := NewTalkService(&fakeTalkRepository{err: apperrors.NotFound("talk.get")}).GetTalkById(c)
	if result.Flag || result.Message != "说说不存在" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
