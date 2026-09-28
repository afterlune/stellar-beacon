package service

import (
	"context"
	"strings"

	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type TopicSubscriptionService interface {
	Subscribe(context.Context, int, string, string) error
	Unsubscribe(context.Context, int, string, string) error
	SetMuted(context.Context, int, string, string, bool) error
	ListSubscriptions(context.Context, int, int, int) ([]port.TopicSubscription, int, error)
	ListFeed(context.Context, int, int, int) ([]port.TopicFeedItem, int, error)
}

type MyTopicSubscriptionService struct {
	repo port.TopicSubscriptionRepository
}

func NewTopicSubscriptionService(repo port.TopicSubscriptionRepository) *MyTopicSubscriptionService {
	return &MyTopicSubscriptionService{repo: repo}
}

func normalizeTopic(topicType, topicKey string) (string, string, error) {
	topicType = strings.ToLower(strings.TrimSpace(topicType))
	topicKey = strings.TrimSpace(topicKey)
	if !port.ValidTopicType(topicType) || topicKey == "" {
		return "", "", apperrors.Invalid("topic_subscription.topic", "invalid topic")
	}
	return topicType, topicKey, nil
}

func (s *MyTopicSubscriptionService) Subscribe(ctx context.Context, userID int, topicType, topicKey string) error {
	topicType, topicKey, err := normalizeTopic(topicType, topicKey)
	if err != nil {
		return err
	}
	return s.repo.Subscribe(ctx, userID, topicType, topicKey)
}

func (s *MyTopicSubscriptionService) Unsubscribe(ctx context.Context, userID int, topicType, topicKey string) error {
	topicType, topicKey, err := normalizeTopic(topicType, topicKey)
	if err != nil {
		return err
	}
	return s.repo.Unsubscribe(ctx, userID, topicType, topicKey)
}

func (s *MyTopicSubscriptionService) SetMuted(ctx context.Context, userID int, topicType, topicKey string, muted bool) error {
	topicType, topicKey, err := normalizeTopic(topicType, topicKey)
	if err != nil {
		return err
	}
	return s.repo.SetMuted(ctx, userID, topicType, topicKey, muted)
}

func (s *MyTopicSubscriptionService) ListSubscriptions(ctx context.Context, userID, current, size int) ([]port.TopicSubscription, int, error) {
	return s.repo.ListSubscriptions(ctx, userID, current, size)
}

func (s *MyTopicSubscriptionService) ListFeed(ctx context.Context, userID, current, size int) ([]port.TopicFeedItem, int, error) {
	return s.repo.ListTopicFeed(ctx, userID, current, size)
}
