package service

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/gin-gonic/gin"
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

func (f *fakeTopicSubscriptionRepository) ListSubscriptions(context.Context, int, int, int) ([]port.TopicSubscription, int, error) {
	return f.subscriptions, len(f.subscriptions), nil
}

func (f *fakeTopicSubscriptionRepository) ListTopicFeed(context.Context, int, int, int) ([]port.TopicFeedItem, int, error) {
	return f.feed, len(f.feed), nil
}

func topicTestContext(method, target, body string, userID int) *gin.Context {
	c := platformTestContext(method, target, body)
	if userID <= 0 {
		c.Set("userInfo", nil)
		return c
	}
	c.Params = gin.Params{}
	for _, segment := range topicPathSegments(target) {
		c.Params = append(c.Params, gin.Param{Key: segment.key, Value: segment.value})
	}
	return c
}

type topicPathSegment struct{ key, value string }

// topicPathSegments mirrors the router parameters for the subscription paths the
// tests exercise, so a service test can drive the real handler logic.
func topicPathSegments(target string) []topicPathSegment {
	path := target
	if index := strings.Index(path, "?"); index >= 0 {
		path = path[:index]
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for index, part := range parts {
		if part != "topic-subscriptions" || index+1 >= len(parts) {
			continue
		}
		if index+2 < len(parts) {
			return []topicPathSegment{{"topicType", parts[index+1]}, {"topicKey", parts[index+2]}}
		}
		return []topicPathSegment{{"topicType", parts[index+1]}}
	}
	return nil
}

func TestTopicSubscriptionRequiresLogin(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	result := service.Subscribe(topicTestContext(http.MethodPut, "/v1/auth/me/topic-subscriptions/tag/go", "", 0))
	if result.Flag {
		t.Fatalf("anonymous subscribe must fail: %+v", result)
	}
	if repo.subscribeCalls != 0 {
		t.Fatalf("anonymous subscribe must not reach the repository: %d", repo.subscribeCalls)
	}
}

func TestTopicSubscriptionRejectsUnknownTopicType(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	result := service.Subscribe(topicTestContext(http.MethodPut, "/v1/auth/me/topic-subscriptions/series/go", "", 7))
	if result.Flag {
		t.Fatalf("series subscriptions are out of scope: %+v", result)
	}
	if repo.subscribeCalls != 0 {
		t.Fatalf("invalid topic type must not reach the repository: %d", repo.subscribeCalls)
	}
}

func TestTopicSubscriptionSubscribeAndUnsubscribeReachRepository(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	if result := service.Subscribe(topicTestContext(http.MethodPut, "/v1/auth/me/topic-subscriptions/tag/Go", "", 7)); !result.Flag {
		t.Fatalf("subscribe failed: %+v", result)
	}
	if repo.lastUserID != 7 || repo.lastTopicType != "tag" || repo.lastTopicKey != "Go" {
		t.Fatalf("unexpected subscribe call: %+v", repo)
	}
	if result := service.Unsubscribe(topicTestContext(http.MethodDelete, "/v1/auth/me/topic-subscriptions/tag/Go", "", 7)); !result.Flag {
		t.Fatalf("unsubscribe failed: %+v", result)
	}
	if repo.unsubscribeCalls != 1 {
		t.Fatalf("unsubscribe must reach the repository once, got %d", repo.unsubscribeCalls)
	}
}

func TestTopicSubscriptionMuteRequiresExplicitValue(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{}
	service := NewTopicSubscriptionService(repo)
	if result := service.SetMuted(topicTestContext(http.MethodPut, "/v1/auth/me/topic-subscriptions/tag/go/mute", `{}`, 7)); result.Flag {
		t.Fatal("mute without an explicit value must fail")
	}
	if repo.muteCalls != 0 {
		t.Fatalf("invalid mute payload must not reach the repository: %d", repo.muteCalls)
	}
	if result := service.SetMuted(topicTestContext(http.MethodPut, "/v1/auth/me/topic-subscriptions/tag/go/mute", `{"muted":1}`, 7)); !result.Flag {
		t.Fatalf("mute failed: %+v", result)
	}
	if repo.muteCalls != 1 || !repo.lastMuted {
		t.Fatalf("unexpected mute call: %+v", repo)
	}
}

func TestTopicSubscriptionListsArePaginated(t *testing.T) {
	repo := &fakeTopicSubscriptionRepository{
		subscriptions: []port.TopicSubscription{{TopicType: port.TopicTypeTag, TopicKey: "go", TopicName: "Go"}},
		feed:          []port.TopicFeedItem{{Topics: []string{"Go"}}},
	}
	service := NewTopicSubscriptionService(repo)
	list := service.ListSubscriptions(topicTestContext(http.MethodGet, "/v1/auth/me/topic-subscriptions?current=1&size=5", "", 7))
	if !list.Flag {
		t.Fatalf("list subscriptions failed: %+v", list)
	}
	feed := service.ListFeed(topicTestContext(http.MethodGet, "/v1/auth/me/topic-feed?current=1&size=5", "", 7))
	if !feed.Flag {
		t.Fatalf("list topic feed failed: %+v", feed)
	}
}
