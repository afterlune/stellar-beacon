package service

import (
	"context"
	"net/http"
	"testing"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
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

func collectionSubscriptionContext(t *testing.T, method, target, body string, collectionID string) *gin.Context {
	t.Helper()
	c := seriesContext(t, method, target, body, gin.Params{{Key: "collectionId", Value: collectionID}})
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	return c
}

func TestCollectionSubscriptionServiceMapsSubscribeErrors(t *testing.T) {
	repo := &fakeCollectionSubscriptionRepository{subscribeErr: apperrors.Invalid("collection_subscription.self", "self")}
	service := NewCollectionSubscriptionService(repo)
	result := service.Subscribe(collectionSubscriptionContext(t, http.MethodPut, "/v1/auth/me/collection-subscriptions/9", "", "9"))
	if result.Flag || result.Message != "不能订阅自己的书单" {
		t.Fatalf("unexpected self-subscription result: %+v", result)
	}

	repo.subscribeErr = apperrors.NotFound("collection_subscription.resolve")
	result = service.Subscribe(collectionSubscriptionContext(t, http.MethodPut, "/v1/auth/me/collection-subscriptions/9", "", "9"))
	if result.Flag || result.Message != "书单不存在或不可订阅" {
		t.Fatalf("unexpected missing-subscription result: %+v", result)
	}
}

func TestCollectionSubscriptionServiceMutesAndListsFeed(t *testing.T) {
	repo := &fakeCollectionSubscriptionRepository{feed: []port.CollectionFeedItem{{EventId: 4, CollectionID: 9, ArticleID: 3}}}
	service := NewCollectionSubscriptionService(repo)
	muted := service.SetMuted(collectionSubscriptionContext(t, http.MethodPut, "/v1/auth/me/collection-subscriptions/9/mute", `{"muted":1}`, "9"))
	if !muted.Flag || !repo.mutedSet {
		t.Fatalf("unexpected mute result: result=%+v muted=%v", muted, repo.mutedSet)
	}
	feed := service.ListFeed(collectionSubscriptionContext(t, http.MethodGet, "/v1/auth/me/collection-feed", "", "9"))
	page, ok := feed.Data.(model.PageResultDTO)
	if !feed.Flag || !ok || page.Count != 1 {
		t.Fatalf("unexpected collection feed result: %+v", feed)
	}
}
