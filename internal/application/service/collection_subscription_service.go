package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type CollectionSubscriptionService interface {
	Subscribe(ctx context.Context, userID, collectionID int) error
	Unsubscribe(ctx context.Context, userID, collectionID int) error
	SetMuted(ctx context.Context, userID, collectionID int, muted bool) error
	GetStatus(ctx context.Context, userID, collectionID int) (port.CollectionSubscriptionStatus, error)
	ListSubscriptions(ctx context.Context, userID, current, size int) ([]port.CollectionSubscription, int, error)
	ListFeed(ctx context.Context, userID, current, size int) ([]port.CollectionFeedItem, int, error)
}

type MyCollectionSubscriptionService struct {
	repo port.CollectionSubscriptionRepository
}

func NewCollectionSubscriptionService(repo port.CollectionSubscriptionRepository) *MyCollectionSubscriptionService {
	return &MyCollectionSubscriptionService{repo: repo}
}

func (s *MyCollectionSubscriptionService) Subscribe(ctx context.Context, userID, collectionID int) error {
	return s.repo.Subscribe(ctx, userID, collectionID)
}

func (s *MyCollectionSubscriptionService) Unsubscribe(ctx context.Context, userID, collectionID int) error {
	return s.repo.Unsubscribe(ctx, userID, collectionID)
}

func (s *MyCollectionSubscriptionService) SetMuted(ctx context.Context, userID, collectionID int, muted bool) error {
	return s.repo.SetMuted(ctx, userID, collectionID, muted)
}

func (s *MyCollectionSubscriptionService) GetStatus(ctx context.Context, userID, collectionID int) (port.CollectionSubscriptionStatus, error) {
	return s.repo.GetStatus(ctx, userID, collectionID)
}

func (s *MyCollectionSubscriptionService) ListSubscriptions(ctx context.Context, userID, current, size int) ([]port.CollectionSubscription, int, error) {
	return s.repo.ListSubscriptions(ctx, userID, current, size)
}

func (s *MyCollectionSubscriptionService) ListFeed(ctx context.Context, userID, current, size int) ([]port.CollectionFeedItem, int, error) {
	return s.repo.ListFeed(ctx, userID, current, size)
}
