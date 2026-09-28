package service

import (
	"context"
	"testing"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
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
	svc := NewFollowService(repo)
	if err := svc.Follow(context.Background(), 7, 9); err != nil || repo.followCalls != 1 || repo.followerID != 7 || repo.authorID != 9 {
		t.Fatalf("unexpected follow: err=%v repo=%+v", err, repo)
	}
}

func TestFollowServiceRejectsSelfFollow(t *testing.T) {
	repo := &fakeFollowRepository{}
	svc := NewFollowService(repo)
	err := svc.Follow(context.Background(), 7, 7)
	if !apperrors.IsKind(err, apperrors.KindValidation) || repo.followCalls != 0 {
		t.Fatalf("self follow must fail before repository call: err=%v repo=%+v", err, repo)
	}
}

func TestFollowServiceUnreadAndMarkRead(t *testing.T) {
	repo := &fakeFollowRepository{unread: 3}
	svc := NewFollowService(repo)
	unread, err := svc.UnreadNotificationCount(context.Background(), 7)
	if err != nil || unread != 3 {
		t.Fatalf("unexpected unread count: count=%d err=%v", unread, err)
	}
	cursor := port.NotificationCursor{PublishEventId: 4, InteractionId: 9}
	if err := svc.MarkNotificationsRead(context.Background(), 7, cursor); err != nil || repo.readCalls != 1 || repo.unread != 0 || repo.readCursor != cursor {
		t.Fatalf("unexpected mark read: err=%v repo=%+v", err, repo)
	}
}
