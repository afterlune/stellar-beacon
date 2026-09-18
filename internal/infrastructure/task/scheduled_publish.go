package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

// scheduledPublishInterval is how often due articles are released. One minute
// keeps the scheduling promise tight without polling the database hard.
const scheduledPublishInterval = time.Minute

// PublishScheduledArticles releases articles whose scheduled time has passed.
// It is the only executor in the project: the t_job records are configuration
// and history, not a scheduler, so scheduled publishing runs on this ticker.
func PublishScheduledArticles(ctx context.Context, repo port.ArticleRepository, newsletter port.NewsletterEnqueuer) {
	if repo == nil {
		slog.Warn("scheduled publishing is disabled: article repository missing")
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(scheduledPublishInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publishDueArticles(ctx, repo, newsletter)
		}
	}
}

func publishDueArticles(ctx context.Context, repo port.ArticleRepository, newsletter port.NewsletterEnqueuer) {
	published, err := repo.PublishDueArticles(ctx)
	if err != nil {
		slog.Error("publish scheduled articles failed", "error", err)
		return
	}
	for _, id := range published {
		slog.Info("scheduled article published", "articleId", id)
		if newsletter == nil {
			continue
		}
		// Notification is best effort: a mail problem must not roll back the
		// publication that already happened.
		if err := newsletter.EnqueueArticle(ctx, id); err != nil {
			slog.Warn("enqueue scheduled article notification failed", "articleId", id, "error", err)
		}
	}
}
