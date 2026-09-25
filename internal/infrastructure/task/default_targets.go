package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type DefaultTargetsDeps struct {
	Publishes        port.ScheduledPublishRepository
	Newsletter       port.NewsletterEnqueuer
	Growth           port.GrowthRepository
	JobLogs          port.JobLogRepository
	UserAreas        port.UserAreaRefresher
	StudioActivation port.StudioActivationReminderRepository
}

func RegisterDefaultTargets(scheduler *Scheduler, deps DefaultTargetsDeps) error {
	if scheduler == nil {
		return errors.New("scheduler is nil")
	}
	if deps.Publishes == nil || deps.Growth == nil || deps.JobLogs == nil || deps.UserAreas == nil || deps.StudioActivation == nil {
		return errors.New("default job dependencies are incomplete")
	}
	targets := []struct {
		meta    port.JobTarget
		handler Handler
	}{
		{
			meta: port.JobTarget{Target: "article.publishScheduled", Name: "Publish scheduled articles", Description: "Publish due scheduled articles, enqueue subscriber notifications, and retry failed hand-offs.", CronExample: "* * * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				return runScheduledPublish(ctx, deps.Publishes, deps.Newsletter)
			},
		},
		{
			meta: port.JobTarget{Target: "growth.cleanup", Name: "Clean growth events", Description: "Delete growth events older than 180 days.", CronExample: "20 3 * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				if err := deps.Growth.Cleanup(ctx, time.Now().Add(-180*24*time.Hour)); err != nil {
					return port.JobRunResult{}, err
				}
				return port.JobRunResult{Processed: true, Message: "growth events cleaned"}, nil
			},
		},
		{
			meta: port.JobTarget{Target: "jobLogs.cleanup", Name: "Clean job logs", Description: "Delete job execution logs older than 30 days.", CronExample: "0 4 * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				if err := deps.JobLogs.CleanBefore(ctx, time.Now().Add(-30*24*time.Hour)); err != nil {
					return port.JobRunResult{}, err
				}
				return port.JobRunResult{Processed: true, Message: "job logs cleaned"}, nil
			},
		},
		{
			meta: port.JobTarget{Target: "studio.activationReminders", Name: "Send Studio activation reminders", Description: "Create low-frequency reminders for creators who have not completed activation.", CronExample: "15 * * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				return runStudioActivationReminders(ctx, deps.StudioActivation)
			},
		},
		{
			meta: port.JobTarget{Target: "userArea.refresh", Name: "Refresh user areas", Description: "Recompute user distribution from login metadata.", CronExample: "*/30 * * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				processed, err := deps.UserAreas.RefreshUserAreas(ctx)
				return port.JobRunResult{Processed: processed, Message: "user area cache refreshed"}, err
			},
		},
	}
	for _, target := range targets {
		if err := scheduler.Register(target.meta, target.handler); err != nil {
			return err
		}
	}
	return nil
}

func runStudioActivationReminders(ctx context.Context, repo port.StudioActivationReminderRepository) (port.JobRunResult, error) {
	count, err := repo.CreateDueStudioActivationReminders(ctx, time.Now(), 200)
	return port.JobRunResult{Processed: count > 0, Message: fmt.Sprintf("studio activation reminders=%d", count)}, err
}

var scheduledPublishRetryDelays = [...]time.Duration{
	time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
}

func runScheduledPublish(ctx context.Context, publishes port.ScheduledPublishRepository, newsletter port.NewsletterEnqueuer) (port.JobRunResult, error) {
	now := time.Now()
	retries, err := publishes.ListRetryableScheduledPublishes(ctx, now, 100)
	if err != nil {
		return port.JobRunResult{}, err
	}
	due, err := publishes.PublishDueScheduledArticles(ctx, now, 100)
	if err != nil {
		return port.JobRunResult{}, err
	}
	records := make([]port.ScheduledPublish, 0, len(retries)+len(due))
	records = append(records, retries...)
	records = append(records, due...)

	queued, suppressed, failed := 0, 0, 0
	var runErr error
	for _, record := range records {
		if record.ModerationStatus == "hidden" {
			if err := publishes.MarkScheduledNotificationSuppressed(ctx, record.RecordID, "content hidden by moderation"); err != nil {
				runErr = errors.Join(runErr, err)
				continue
			}
			suppressed++
			continue
		}
		if newsletter == nil {
			if err := publishes.MarkScheduledNotificationSuppressed(ctx, record.RecordID, "newsletter service is not configured"); err != nil {
				runErr = errors.Join(runErr, err)
			}
			suppressed++
			continue
		}
		if err := newsletter.EnqueueArticle(ctx, record.ArticleID); err != nil {
			failed++
			slog.Warn("enqueue scheduled article notification failed", "articleId", record.ArticleID, "attempt", record.NotificationAttempts+1, "error", err)
			var retryAt *time.Time
			if record.NotificationAttempts < len(scheduledPublishRetryDelays) {
				next := now.Add(scheduledPublishRetryDelays[record.NotificationAttempts])
				retryAt = &next
			}
			if markErr := publishes.MarkScheduledNotificationFailed(ctx, record.RecordID, err.Error(), retryAt); markErr != nil {
				runErr = errors.Join(runErr, markErr)
			}
			continue
		}
		if err := publishes.MarkScheduledNotificationQueued(ctx, record.RecordID, time.Now()); err != nil {
			runErr = errors.Join(runErr, err)
			continue
		}
		queued++
	}
	message := fmt.Sprintf("published=%d queued=%d retried=%d failed=%d suppressed=%d", len(due), queued, len(retries), failed, suppressed)
	if runErr != nil {
		return port.JobRunResult{Processed: len(due)+len(retries) > 0, Message: message}, runErr
	}
	return port.JobRunResult{Processed: len(due)+len(retries) > 0, Message: message}, nil
}
