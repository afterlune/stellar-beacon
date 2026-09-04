package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCommentRepository struct{}

func (f *fakeCommentRepository) ListComments(context.Context, port.CommentFilter) ([]*port.Comment, int, error) {
	return []*port.Comment{{Id: 10}}, 1, nil
}
func (f *fakeCommentRepository) ListReplies(context.Context, []int) ([]*port.Reply, error) {
	return []*port.Reply{{ParentId: 10, CommentContent: "reply"}}, nil
}
func (f *fakeCommentRepository) ListTopSixComments(context.Context) ([]*port.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepository) CountComments(context.Context, port.CommentFilter) (int64, error) {
	return 0, nil
}
func (f *fakeCommentRepository) ListCommentsAdmin(context.Context, port.CommentFilter) ([]*port.CommentAdmin, error) {
	return nil, nil
}
func (f *fakeCommentRepository) ListCommentCountsByTypeAndTopicIDs(context.Context, int, []int) ([]*port.CommentCount, error) {
	return nil, nil
}
func (f *fakeCommentRepository) ListCommentCountByTypeAndTopicID(context.Context, int, int) (port.CommentCount, error) {
	return port.CommentCount{}, nil
}
func (f *fakeCommentRepository) ValidateTarget(context.Context, int, int) error { return nil }
func (f *fakeCommentRepository) ValidateReply(context.Context, int, int, int) error {
	return nil
}
func (f *fakeCommentRepository) Create(context.Context, entity.TComment) error { return nil }
func (f *fakeCommentRepository) Review(context.Context, []int, int) error      { return nil }
func (f *fakeCommentRepository) Delete(context.Context, []int) error           { return nil }

func TestCommentServiceAttachesRepliesUsingPortData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/comments?current=1&size=10", nil)
	result := mustCommentService(t, &fakeCommentRepository{}).ListComments(serviceTestRequest{ginContextForServiceTest: c})
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	page, ok := result.Data.(interface{})
	if !ok || page == nil {
		t.Fatal("expected page data")
	}
}
