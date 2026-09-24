package service

import (
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
)

func dashboardGrowthDTO(stats port.NewsletterStats, events []port.GrowthTrend, deliveries []port.NewsletterDeliveryTrend, activation port.StudioActivationFunnel, start, now time.Time, unit string) model.DashboardGrowthDTO {
	confirmedDenominator := stats.Active + stats.Pending
	deliveryDenominator := stats.Sent + stats.Failed
	result := model.DashboardGrowthDTO{
		Subscribers: model.GrowthSubscriberStatsDTO{
			Total: stats.Total, Active: stats.Active, Pending: stats.Pending, Unsubscribed: stats.Unsubscribed,
			ConfirmationRate: ratio(stats.Active, confirmedDenominator),
		},
		Deliveries: model.GrowthDeliveryStatsDTO{
			Queued: stats.Queued, Sending: stats.Sending, Sent: stats.Sent, Failed: stats.Failed,
			SuccessRate: ratio(stats.Sent, deliveryDenominator),
		},
		Activation: model.StudioActivationFunnelDTO{
			Started: activation.Started, IdentityCompleted: activation.IdentityCompleted,
			ContentCompleted: activation.ContentCompleted, ProfileVisited: activation.ProfileVisited,
			Completed: activation.Completed, CompletionRate: ratio(activation.Completed, activation.Started),
		},
		Trend: make([]model.DashboardGrowthTrendDTO, 0),
	}
	eventByPeriod := make(map[string]port.GrowthTrend, len(events))
	for _, event := range events {
		eventByPeriod[event.Period] = event
	}
	deliveryByPeriod := make(map[string]port.NewsletterDeliveryTrend, len(deliveries))
	for _, delivery := range deliveries {
		deliveryByPeriod[delivery.Period] = delivery
	}
	for cursor := start; !cursor.After(now); cursor = nextPeriod(cursor, unit) {
		period := cursor.Format("2006-01-02")
		if unit == "month" {
			period = cursor.Format("2006-01")
		}
		event := eventByPeriod[period]
		delivery := deliveryByPeriod[period]
		result.Trend = append(result.Trend, model.DashboardGrowthTrendDTO{
			Period: period, ShareClicks: event.ShareClicks, SubscribeStarts: event.SubscribeStarts,
			SubscribeConfirms: event.SubscribeConfirms, Unsubscribes: event.Unsubscribes,
			DeliverySent: delivery.Sent, DeliveryFailed: delivery.Failed,
		})
	}
	return result
}

func nextPeriod(value time.Time, unit string) time.Time {
	if unit == "month" {
		return value.AddDate(0, 1, 0)
	}
	return value.AddDate(0, 0, 1)
}

func ratio(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator) * 100
}
