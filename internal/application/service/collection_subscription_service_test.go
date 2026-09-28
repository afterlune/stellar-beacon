package service

import (
	"context"
	"errors"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type fakeCollectionSubscriptionRepository struct {
	subscribeErr error
	mutedSet     bool
	feed         []port.CollectionFeedItem
}

func (f *fakeCollectionSubscriptionRepository) Subscribe(context.Context, int, int) error {
	return f.subscribeErr
}
func (f *fakeCollectionSubscriptionRepository) Unsubscribe(context.Context, int, int) error {
	return nil
}
func (f *fakeCollectionSubscriptionRepository) SetMuted(_ context.Context, _, _ int, muted bool) error {
	f.mutedSet = muted
	return nil
}
func (f *fakeCollectionSubscriptionRepository) GetStatus(context.Context, int, int) (port.CollectionSubscriptionStatus, error) {
	return port.CollectionSubscriptionStatus{}, nil
}
func (f *fakeCollectionSubscriptionRepository) ListSubscriptions(context.Context, int, int, int) ([]port.CollectionSubscription, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectionSubscriptionRepository) ListFeed(context.Context, int, int, int) ([]port.CollectionFeedItem, int, error) {
	return f.feed, len(f.feed), nil
}

func TestCollectionSubscriptionServiceDelegatesTypedCommands(t *testing.T) {
	ctx := context.Background()
	repo := &fakeCollectionSubscriptionRepository{feed: []port.CollectionFeedItem{{EventId: 4, CollectionID: 9, ArticleID: 3}}}
	svc := NewCollectionSubscriptionService(repo)

	if err := svc.Subscribe(ctx, 7, 9); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := svc.SetMuted(ctx, 7, 9, true); err != nil || !repo.mutedSet {
		t.Fatalf("set muted: err=%v muted=%v", err, repo.mutedSet)
	}
	feed, count, err := svc.ListFeed(ctx, 7, 1, 12)
	if err != nil || count != 1 || len(feed) != 1 || feed[0].EventId != 4 {
		t.Fatalf("unexpected feed: records=%+v count=%d err=%v", feed, count, err)
	}
}

func TestCollectionSubscriptionServiceReturnsRepositoryErrors(t *testing.T) {
	want := errors.New("repository unavailable")
	svc := NewCollectionSubscriptionService(&fakeCollectionSubscriptionRepository{subscribeErr: want})
	if err := svc.Subscribe(context.Background(), 7, 9); !errors.Is(err, want) {
		t.Fatalf("expected repository error to be preserved, got %v", err)
	}
}
