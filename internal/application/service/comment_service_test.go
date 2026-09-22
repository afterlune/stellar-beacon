package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCommentRepository struct{}

type targetRecordingCommentRepository struct {
	fakeCommentRepository
	Type    int
	TopicID int
}

func (f *targetRecordingCommentRepository) ValidateTarget(_ context.Context, commentType, topicID int) error {
	f.Type = commentType
	f.TopicID = topicID
	return nil
}

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
func (f *fakeCommentRepository) ResolveCommentPage(context.Context, int, int, int, int) (int, error) {
	return 1, nil
}
func (f *fakeCommentRepository) MarkNotificationDispatched(context.Context, int) error { return nil }
func (f *fakeCommentRepository) Create(context.Context, entity.TComment) (int, error)  { return 0, nil }
func (f *fakeCommentRepository) GetByID(context.Context, int) (entity.TComment, error) {
	return entity.TComment{}, nil
}
func (f *fakeCommentRepository) Review(context.Context, []int, int) error { return nil }
func (f *fakeCommentRepository) Delete(context.Context, []int) error      { return nil }

func TestCommentServiceAttachesRepliesUsingPortData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/comments?current=1&size=10", nil)
	result := mustCommentService(t, &fakeCommentRepository{}).ListComments(c)
	if !result.Flag {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Data == nil {
		t.Fatal("expected page data")
	}
}

func TestCommentServiceAcceptsCollectionTopic(t *testing.T) {
	repo := &targetRecordingCommentRepository{}
	service := mustCommentService(t, repo)
	if err := service.checkComment(context.Background(), model.CommentVO{Type: Collection, TopicId: "42"}); err != nil {
		t.Fatalf("collection comment must validate: %v", err)
	}
	if repo.Type != Collection || repo.TopicID != 42 {
		t.Fatalf("unexpected target validation: type=%d topic=%d", repo.Type, repo.TopicID)
	}
}
