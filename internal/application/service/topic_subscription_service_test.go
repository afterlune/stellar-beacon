package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type fakeTopicSubscriptionRepository struct {
	port.TopicSubscriptionRepository
	subscribeCalls   int
	unsubscribeCalls int
	muteCalls        int
	lastUserID       int
	lastTopicType    string
	lastTopicKey     string
	lastMuted        bool
	lastCurrent      int
	lastSize         int
	subscribeErr     error
	muteErr          error
	subscriptions    []port.TopicSubscription
	feed             []port.TopicFeedItem
}

func (f *fakeTopicSubscriptionRepository) Subscribe(_ context.Context, userID int, topicType, topicKey string) error {
	f.subscribeCalls++
	f.lastUserID = userID
	f.lastTopicType = topicType
	f.lastTopicKey = topicKey
	return f.subscribeErr
}

func (f *fakeTopicSubscriptionRepository) Unsubscribe(_ context.Context, userID int, topicType, topicKey string) error {
	f.unsubscribeCalls++
	f.lastUserID = userID
	f.lastTopicType = topicType
	f.lastTopicKey = topicKey
	return nil
}

func (f *fakeTopicSubscriptionRepository) SetMuted(_ context.Context, userID int, topicType, topicKey string, muted bool) error {
	f.muteCalls++
	f.lastUserID = userID
	f.lastTopicType = topicType
	f.lastTopicKey = topicKey
	f.lastMuted = muted
	return f.muteErr
}

func (f *fakeTopicSubscriptionRepository) ListSubscriptions(_ context.Context, userID, current, size int) ([]port.TopicSubscription, int, error) {
	f.lastUserID, f.lastCurrent, f.lastSize = userID, current, size
	return f.subscriptions, len(f.subscriptions), nil
}

func (f *fakeTopicSubscriptionRepository) ListTopicFeed(_ context.Context, userID, current, size int) ([]port.TopicFeedItem, int, error) {
	f.lastUserID, f.lastCurrent, f.lastSize = userID, current, size
	return f.feed, len(f.feed), nil
}

func TestTopicSubscriptionRejectsUnknownTopicType(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	if err := service.Subscribe(context.Background(), 7, "series", "go"); err == nil {
		t.Fatal("series subscriptions are out of scope")
	}
	if repo.subscribeCalls != 0 {
		t.Fatalf("invalid topic type must not reach the repository: %d", repo.subscribeCalls)
	}
}

func TestTopicSubscriptionSubscribeAndUnsubscribeReachRepository(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	if err := service.Subscribe(context.Background(), 7, " TAG ", " Go "); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	if repo.lastUserID != 7 || repo.lastTopicType != "tag" || repo.lastTopicKey != "Go" {
		t.Fatalf("unexpected subscribe call: %+v", repo)
	}
	if err := service.Unsubscribe(context.Background(), 7, "tag", "Go"); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	if repo.unsubscribeCalls != 1 {
		t.Fatalf("unsubscribe must reach the repository once, got %d", repo.unsubscribeCalls)
	}
}

func TestTopicSubscriptionMutePassesDesiredState(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	if err := service.SetMuted(context.Background(), 7, "tag", "go", true); err != nil {
		t.Fatalf("mute failed: %v", err)
	}
	if repo.muteCalls != 1 || !repo.lastMuted {
		t.Fatalf("unexpected mute call: %+v", repo)
	}
}

func TestTopicSubscriptionPropagatesRepositoryErrors(t *testing.T) {
	want := errors.New("repository unavailable")
	repo := &fakeTopicSubscriptionRepository{subscribeErr: want}
	service := NewTopicSubscriptionService(repo)
	if err := service.Subscribe(context.Background(), 7, "tag", "go"); !errors.Is(err, want) {
		t.Fatalf("expected repository error %v, got %v", want, err)
	}
}

func TestTopicSubscriptionListsArePaginated(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{
		subscriptions: []port.TopicSubscription{{TopicType: port.TopicTypeTag, TopicKey: "go", TopicName: "Go"}},
		feed:          []port.TopicFeedItem{{Topics: []string{"Go"}}},
	}
	service := NewTopicSubscriptionService(repo)
	if _, count, err := service.ListSubscriptions(context.Background(), 7, 2, 5); err != nil || count != 1 {
		t.Fatalf("list subscriptions failed: count=%d err=%v", count, err)
	}
	if repo.lastUserID != 7 || repo.lastCurrent != 2 || repo.lastSize != 5 {
		t.Fatalf("unexpected subscription pagination: %+v", repo)
	}
	if _, count, err := service.ListFeed(context.Background(), 7, 3, 9); err != nil || count != 1 {
		t.Fatalf("list topic feed failed: count=%d err=%v", count, err)
	}
	if repo.lastUserID != 7 || repo.lastCurrent != 3 || repo.lastSize != 9 {
		t.Fatalf("unexpected feed pagination: %+v", repo)
	}
}
