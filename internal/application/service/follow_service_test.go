package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

type fakeFollowRepository struct {
	port.FollowRepository
	followCalls   int
	unfollowCalls int
	readCalls     int
	readCursor    port.NotificationCursor
	followerID    int
	authorID      int
	unread        int
}

func (f *fakeFollowRepository) Follow(_ context.Context, followerID, authorID int) error {
	f.followCalls++
	f.followerID = followerID
	f.authorID = authorID
	return nil
}

func (f *fakeFollowRepository) Unfollow(_ context.Context, followerID, authorID int) error {
	f.unfollowCalls++
	f.followerID = followerID
	f.authorID = authorID
	return nil
}

func (f *fakeFollowRepository) UnreadNotificationCount(context.Context, int) (int, error) {
	return f.unread, nil
}

func (f *fakeFollowRepository) MarkNotificationsRead(_ context.Context, _ int, cursor port.NotificationCursor) error {
	f.readCalls++
	f.readCursor = cursor
	f.unread = 0
	return nil
}

func TestFollowServiceFollowsTarget(t *testing.T) {
	repo := &fakeFollowRepository{}
	service := NewFollowService(repo)
	ctx := platformTestContext(http.MethodPut, "/v1/auth/me/following/9", "")
	ctx.Params = gin.Params{{Key: "authorId", Value: "9"}}
	result := service.Follow(ctx)
	if !result.Flag || repo.followCalls != 1 || repo.followerID != 7 || repo.authorID != 9 {
		t.Fatalf("unexpected follow result: result=%+v repo=%+v", result, repo)
	}
}

func TestFollowServiceRejectsSelfFollow(t *testing.T) {
	repo := &fakeFollowRepository{}
	service := NewFollowService(repo)
	ctx := platformTestContext(http.MethodPut, "/v1/auth/me/following/7", "")
	ctx.Params = gin.Params{{Key: "authorId", Value: "7"}}
	ctx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	result := service.Follow(ctx)
	if result.Flag || repo.followCalls != 0 {
		t.Fatalf("self follow must fail: result=%+v repo=%+v", result, repo)
	}
}

func TestFollowServiceUnreadAndMarkRead(t *testing.T) {
	repo := &fakeFollowRepository{unread: 3}
	service := NewFollowService(repo)
	unread := service.UnreadNotificationCount(platformTestContext(http.MethodGet, "/v1/auth/me/notifications/unread-count", ""))
	if !unread.Flag || unread.Data.(map[string]int)["count"] != 3 {
		t.Fatalf("unexpected unread result: %+v", unread)
	}
	read := service.MarkNotificationsRead(platformTestContext(http.MethodPost, "/v1/auth/me/notifications/read", `{"publishEventId":4,"interactionId":9}`))
	if !read.Flag || repo.readCalls != 1 || repo.unread != 0 || repo.readCursor.PublishEventId != 4 || repo.readCursor.InteractionId != 9 {
		t.Fatalf("unexpected mark read result: result=%+v repo=%+v", read, repo)
	}
}
