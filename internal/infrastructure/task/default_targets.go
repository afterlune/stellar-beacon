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
	Articles   port.ArticleRepository
	Newsletter port.NewsletterEnqueuer
	Growth     port.GrowthRepository
	JobLogs    port.JobLogRepository
	UserAreas  port.UserAreaRefresher
}

func RegisterDefaultTargets(scheduler *Scheduler, deps DefaultTargetsDeps) error {
	if scheduler == nil {
		return errors.New("scheduler is nil")
	}
	if deps.Articles == nil || deps.Growth == nil || deps.JobLogs == nil || deps.UserAreas == nil {
		return errors.New("default job dependencies are incomplete")
	}
	targets := []struct {
		meta    port.JobTarget
		handler Handler
	}{
		{
			meta: port.JobTarget{Target: "article.publishScheduled", Name: "Publish scheduled articles", Description: "Publish due scheduled articles and enqueue subscriber notifications.", CronExample: "* * * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				ids, err := deps.Articles.PublishDueArticles(ctx)
				if err != nil {
					return port.JobRunResult{}, err
				}
				for _, id := range ids {
					if deps.Newsletter == nil {
						continue
					}
					if err := deps.Newsletter.EnqueueArticle(ctx, id); err != nil {
						slog.Warn("enqueue scheduled article notification failed", "articleId", id, "error", err)
					}
				}
				return port.JobRunResult{Processed: len(ids) > 0, Message: fmt.Sprintf("published %d articles", len(ids))}, nil
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
			meta: port.JobTarget{Target: "jobLogs.cleanup", Name: "Clean job logs", Description: "Purge the job execution log.", CronExample: "0 4 * * *"},
			handler: func(ctx context.Context) (port.JobRunResult, error) {
				if err := deps.JobLogs.Clean(ctx); err != nil {
					return port.JobRunResult{}, err
				}
				return port.JobRunResult{Processed: true, Message: "job logs cleaned"}, nil
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
