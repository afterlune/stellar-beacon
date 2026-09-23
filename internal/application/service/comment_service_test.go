package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCommentRepository struct{}

type targetRecordingCommentRepository struct {
	fakeCommentRepository
	Type    int
	TopicID int
}

type moderationRecordingCommentRepository struct {
	fakeCommentRepository
	PinnedUserID        int
	PinnedCollectionID  int
	PinnedCommentID     int
	PinnedValue         bool
	DeletedUserID       int
	DeletedCollectionID int
	DeletedCommentID    int
}

func (f *moderationRecordingCommentRepository) SetPinned(_ context.Context, userID, collectionID, commentID int, pinned bool) error {
	f.PinnedUserID = userID
	f.PinnedCollectionID = collectionID
	f.PinnedCommentID = commentID
	f.PinnedValue = pinned
	return nil
}

func (f *moderationRecordingCommentRepository) SoftDeleteOwned(_ context.Context, userID, collectionID, commentID int) error {
	f.DeletedUserID = userID
	f.DeletedCollectionID = collectionID
	f.DeletedCommentID = commentID
	return nil
}

func (f *targetRecordingCommentRepository) ValidateTarget(_ context.Context, commentType, topicID int) error {
	f.Type = commentType
	f.TopicID = topicID
	return nil
}

func (f *fakeCommentRepository) ListComments(context.Context, port.CommentFilter) ([]*port.Comment, int, error) {
	return []*port.Comment{{Id: 10}}, 1, nil
}
func (f *fakeCommentRepository) ListReplies(context.Context, []int, int) ([]*port.Reply, error) {
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
func (f *fakeCommentRepository) Review(context.Context, []int, int) error             { return nil }
func (f *fakeCommentRepository) Delete(context.Context, []int) error                  { return nil }
func (f *fakeCommentRepository) SetPinned(context.Context, int, int, int, bool) error { return nil }
func (f *fakeCommentRepository) SoftDeleteOwned(context.Context, int, int, int) error { return nil }

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

func TestCommentServicePinCollectionCommentRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/studio/collections/7/comments/11/pin", nil)
	result := mustCommentService(t, &moderationRecordingCommentRepository{}).PinCollectionComment(c)
	if result.Flag {
		t.Fatalf("expected no-login result: %+v", result)
	}
}

func TestCommentServicePinAndDeleteOwnedCollectionComment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &moderationRecordingCommentRepository{}
	service := mustCommentService(t, repo)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "collectionId", Value: "7"}, {Key: "commentId", Value: "11"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/studio/collections/7/comments/11/pin", strings.NewReader(`{"pinned":true}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 5})
	if result := service.PinCollectionComment(c); !result.Flag {
		t.Fatalf("unexpected pin result: %+v", result)
	}
	if repo.PinnedUserID != 5 || repo.PinnedCollectionID != 7 || repo.PinnedCommentID != 11 || !repo.PinnedValue {
		t.Fatalf("unexpected pin call: %+v", repo)
	}

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Params = gin.Params{{Key: "collectionId", Value: "7"}, {Key: "commentId", Value: "12"}}
	c2.Request = httptest.NewRequest(http.MethodDelete, "/studio/collections/7/comments/12", nil)
	c2.Set("userInfo", model.UserDetailsDTO{UserInfoId: 5})
	if result := service.DeleteOwnedCollectionComment(c2); !result.Flag {
		t.Fatalf("unexpected delete result: %+v", result)
	}
	if repo.DeletedUserID != 5 || repo.DeletedCollectionID != 7 || repo.DeletedCommentID != 12 {
		t.Fatalf("unexpected delete call: %+v", repo)
	}
}
