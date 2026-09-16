package port

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"time"
)

const (
	NewsletterPending      = "pending"
	NewsletterActive       = "active"
	NewsletterUnsubscribed = "unsubscribed"

	DeliveryQueued  = "queued"
	DeliverySending = "sending"
	DeliverySent    = "sent"
	DeliveryFailed  = "failed"
)

type NewsletterFilter struct {
	Current int
	Size    int
	Status  string
	Keyword string
}

type DeliveryFilter struct {
	Current int
	Size    int
	Status  string
}

type NewsletterStats struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	Pending      int64 `json:"pending"`
	Unsubscribed int64 `json:"unsubscribed"`
	Queued       int64 `json:"queued"`
	Sending      int64 `json:"sending"`
	Sent         int64 `json:"sent"`
	Failed       int64 `json:"failed"`
}

type NewsletterDeliveryTrend struct {
	Period string `json:"period"`
	Sent   int64  `json:"sent"`
	Failed int64  `json:"failed"`
}

type NewsletterRepository interface {
	FindSubscriberByEmail(ctx context.Context, email string) (entity.TNewsletterSubscriber, error)
	FindSubscriberByID(ctx context.Context, id int) (entity.TNewsletterSubscriber, error)
	FindSubscriberByToken(ctx context.Context, field, tokenHash string) (entity.TNewsletterSubscriber, error)
	SaveSubscriber(ctx context.Context, subscriber entity.TNewsletterSubscriber) error
	ActivateSubscriber(ctx context.Context, id int, confirmedAt time.Time) error
	UnsubscribeSubscriber(ctx context.Context, id int) error
	SetSubscriberStatus(ctx context.Context, id int, status string) error
	ListSubscribers(ctx context.Context, filter NewsletterFilter) ([]entity.TNewsletterSubscriber, int, error)
	ListDeliveries(ctx context.Context, filter DeliveryFilter) ([]entity.TNewsletterDelivery, int, error)
	Stats(ctx context.Context) (NewsletterStats, error)
	DeliveryTrend(ctx context.Context, since time.Time, unit string) ([]NewsletterDeliveryTrend, error)
	CreateDeliveriesForArticle(ctx context.Context, articleID int) error
	ClaimNextDelivery(ctx context.Context) (entity.TNewsletterDelivery, bool, error)
	MarkDeliverySent(ctx context.Context, id int, sentAt time.Time) error
	MarkDeliveryFailed(ctx context.Context, id int, message string, retry bool) error
	RetryDelivery(ctx context.Context, id int) error
	RetryFailedDeliveries(ctx context.Context) (int, error)
}

type GrowthSummary struct {
	EventName string    `json:"eventName"`
	Day       string    `json:"day"`
	Count     int64     `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

type GrowthTrend struct {
	Period            string `json:"period"`
	ShareClicks       int64  `json:"shareClicks"`
	SubscribeStarts   int64  `json:"subscribeStarts"`
	SubscribeConfirms int64  `json:"subscribeConfirms"`
	Unsubscribes      int64  `json:"unsubscribes"`
}

type GrowthRepository interface {
	RecordEvent(ctx context.Context, event entity.TGrowthEvent) error
	Summary(ctx context.Context, since time.Time) ([]GrowthSummary, error)
	SummaryByPeriod(ctx context.Context, since time.Time, unit string) ([]GrowthTrend, error)
	Cleanup(ctx context.Context, before time.Time) error
}
