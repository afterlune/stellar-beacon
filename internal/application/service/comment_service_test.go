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
	BatchAction         string
	BatchCommentIDs     []int
	RestoredCommentIDs  []int
	AdminFilter         port.CommentFilter
	RestoreAdminIDs     []int
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

func (f *moderationRecordingCommentRepository) BatchModerateOwned(_ context.Context, _, collectionID int, action string, commentIDs []int) (port.ModerationBatchResult, error) {
	f.BatchAction = action
	f.BatchCommentIDs = append([]int{}, commentIDs...)
	return port.ModerationBatchResult{Succeeded: append([]int{}, commentIDs...), Failed: []port.ModerationFailure{}}, nil
}

func (f *moderationRecordingCommentRepository) ListCommentsAdmin(_ context.Context, filter port.CommentFilter) ([]*port.CommentAdmin, error) {
	f.AdminFilter = filter
	return []*port.CommentAdmin{{Id: 1, Type: 6, IsDelete: 1}}, nil
}

func (f *moderationRecordingCommentRepository) CountComments(_ context.Context, filter port.CommentFilter) (int64, error) {
	f.AdminFilter = filter
	return 1, nil
}

func (f *moderationRecordingCommentRepository) GetByID(_ context.Context, id int) (entity.TComment, error) {
	return entity.TComment{Id: id, Type: 6, TopicId: 42, IsDelete: 1, IsReview: 1}, nil
}

func (f *moderationRecordingCommentRepository) RestoreAsAdmin(_ context.Context, _, _ int, commentIDs []int) (port.ModerationBatchResult, error) {
	f.RestoreAdminIDs = append([]int{}, commentIDs...)
	return port.ModerationBatchResult{Succeeded: append([]int{}, commentIDs...), Failed: []port.ModerationFailure{}}, nil
}

func (f *moderationRecordingCommentRepository) RestoreOwned(_ context.Context, _, collectionID int, commentIDs []int) (port.ModerationBatchResult, error) {
	f.RestoredCommentIDs = append([]int{}, commentIDs...)
	return port.ModerationBatchResult{Succeeded: append([]int{}, commentIDs...), Failed: []port.ModerationFailure{}}, nil
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
func (f *fakeCommentRepository) BatchModerateOwned(context.Context, int, int, string, []int) (port.ModerationBatchResult, error) {
	return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, nil
}
func (f *fakeCommentRepository) RestoreOwned(context.Context, int, int, []int) (port.ModerationBatchResult, error) {
	return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, nil
}
func (f *fakeCommentRepository) RestoreAsAdmin(context.Context, int, int, []int) (port.ModerationBatchResult, error) {
	return port.ModerationBatchResult{Succeeded: []int{}, Failed: []port.ModerationFailure{}}, nil
}
func (f *fakeCommentRepository) ListOwnedCollectionComments(context.Context, int, int, int, int, string, bool) ([]*port.OwnedComment, int, error) {
	return []*port.OwnedComment{{Id: 10}}, 1, nil
}
func (f *fakeCommentRepository) CountPendingReportsByCommentIDs(context.Context, []int) (map[int]int, error) {
	return map[int]int{}, nil
}
func (f *fakeCommentRepository) WriteModerationAudit(context.Context, port.ModerationAuditEntry) error {
	return nil
}

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
func TestCommentServiceBatchModerateCollectionComments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &moderationRecordingCommentRepository{}
	service := mustCommentService(t, repo)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "collectionId", Value: "7"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/studio/collections/7/comments/batch", strings.NewReader(`{"action":"delete","commentIds":[3,1,2]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 5})
	result := service.BatchModerateCollectionComments(c)
	if !result.Flag {
		t.Fatalf("unexpected batch result: %+v", result)
	}
	if repo.BatchAction != "delete" || len(repo.BatchCommentIDs) != 3 {
		t.Fatalf("unexpected batch call: action=%q ids=%v", repo.BatchAction, repo.BatchCommentIDs)
	}

	restoreCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	restoreCtx.Params = gin.Params{{Key: "collectionId", Value: "7"}}
	restoreCtx.Request = httptest.NewRequest(http.MethodPost, "/studio/collections/7/comments/restore", strings.NewReader(`{"commentIds":[9]}`))
	restoreCtx.Request.Header.Set("Content-Type", "application/json")
	restoreCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 5})
	if restored := service.RestoreOwnedCollectionComments(restoreCtx); !restored.Flag {
		t.Fatalf("unexpected restore result: %+v", restored)
	}
	if len(repo.RestoredCommentIDs) != 1 || repo.RestoredCommentIDs[0] != 9 {
		t.Fatalf("unexpected restore call: %v", repo.RestoredCommentIDs)
	}
}

func TestCommentServiceBatchModerateCollectionCommentsRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "collectionId", Value: "7"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/studio/collections/7/comments/batch", strings.NewReader(`{"action":"delete","commentIds":[1]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	result := mustCommentService(t, &moderationRecordingCommentRepository{}).BatchModerateCollectionComments(c)
	if result.Flag {
		t.Fatalf("expected login requirement: %+v", result)
	}
}

func TestCommentServiceListOwnedCollectionComments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "collectionId", Value: "7"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/studio/collections/7/comments?current=1&size=10&includeDeleted=1", nil)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 5})
	result := mustCommentService(t, &fakeCommentRepository{}).ListOwnedCollectionComments(c)
	if !result.Flag || result.Data == nil {
		t.Fatalf("unexpected governance list result: %+v", result)
	}
}
func TestCommentServiceListCollectionCommentsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &moderationRecordingCommentRepository{}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "collectionId", Value: "7"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/collections/7/comments?current=1&size=20", nil)
	result := mustCommentService(t, repo).ListCollectionCommentsAdmin(c)
	if !result.Flag {
		t.Fatalf("unexpected admin list result: %+v", result)
	}
	if repo.AdminFilter.CollectionID != 7 || repo.AdminFilter.Type != 6 {
		t.Fatalf("admin list must be scoped to the reading list: %+v", repo.AdminFilter)
	}
}

func TestCommentServiceRestoreCommentsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &moderationRecordingCommentRepository{}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Params = gin.Params{{Key: "commentId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/comments/9/restore", nil)
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 3})
	result := mustCommentService(t, repo).RestoreCommentsAdmin(c)
	if !result.Flag {
		t.Fatalf("unexpected admin restore result: %+v", result)
	}
	if len(repo.RestoreAdminIDs) != 1 || repo.RestoreAdminIDs[0] != 9 {
		t.Fatalf("admin restore must target the requested comment: %v", repo.RestoreAdminIDs)
	}
}
