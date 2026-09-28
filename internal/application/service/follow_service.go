package service

import (
	"context"
	"strings"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type FollowService interface {
	Follow(ctx context.Context, followerID, authorID int) error
	Unfollow(ctx context.Context, followerID, authorID int) error
	ListFollowing(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error)
	ListFollowers(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error)
	ListFeed(ctx context.Context, userID int, contentType string, current, size int) ([]port.FollowFeedItem, int, error)
	ListNotifications(ctx context.Context, userID int, group string, current, size int) (port.NotificationPage, error)
	UnreadNotificationCount(ctx context.Context, userID int) (int, error)
	MarkNotificationsRead(ctx context.Context, userID int, cursor port.NotificationCursor) error
}

type MyFollowService struct{ repo port.FollowRepository }

func NewFollowService(repo port.FollowRepository) *MyFollowService {
	return &MyFollowService{repo: repo}
}

func (s *MyFollowService) Follow(ctx context.Context, followerID, authorID int) error {
	if authorID <= 0 || authorID == followerID {
		return apperrors.Invalid("follow.target", "invalid follow target")
	}
	return s.repo.Follow(ctx, followerID, authorID)
}

func (s *MyFollowService) Unfollow(ctx context.Context, followerID, authorID int) error {
	if authorID <= 0 || authorID == followerID {
		return apperrors.Invalid("follow.target", "invalid follow target")
	}
	return s.repo.Unfollow(ctx, followerID, authorID)
}

func (s *MyFollowService) ListFollowing(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error) {
	return s.repo.ListFollowing(ctx, userID, current, size)
}

func (s *MyFollowService) ListFollowers(ctx context.Context, userID, current, size int) ([]port.FollowUser, int, error) {
	return s.repo.ListFollowers(ctx, userID, current, size)
}

func (s *MyFollowService) ListFeed(ctx context.Context, userID int, contentType string, current, size int) ([]port.FollowFeedItem, int, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "all" {
		contentType = ""
	}
	if contentType != "" && contentType != port.FollowContentArticle && contentType != port.FollowContentTalk {
		return nil, 0, apperrors.Invalid("follow.feed.type", "invalid content type")
	}
	return s.repo.ListFollowFeed(ctx, userID, contentType, current, size)
}

func (s *MyFollowService) ListNotifications(ctx context.Context, userID int, group string, current, size int) (port.NotificationPage, error) {
	group = strings.ToLower(strings.TrimSpace(group))
	if group == "" {
		group = port.NotificationGroupAll
	}
	if !port.ValidNotificationGroup(group) {
		return port.NotificationPage{}, apperrors.Invalid("follow.notification.group", "invalid notification group")
	}
	return s.repo.ListNotifications(ctx, userID, group, current, size)
}

func (s *MyFollowService) UnreadNotificationCount(ctx context.Context, userID int) (int, error) {
	return s.repo.UnreadNotificationCount(ctx, userID)
}

func (s *MyFollowService) MarkNotificationsRead(ctx context.Context, userID int, cursor port.NotificationCursor) error {
	return s.repo.MarkNotificationsRead(ctx, userID, cursor)
}

var _ FollowService = (*MyFollowService)(nil)
