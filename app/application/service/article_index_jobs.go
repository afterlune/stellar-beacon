package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
)

func (a *MyArticleService) enqueueArticleIndexJob(ctx context.Context, articleID int, action port.ArticleIndexAction, event port.ArticleLifecycleEvent, status, isDelete int, revision time.Time) error {
	if a == nil || a.aiJobs == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if revision.IsZero() {
		revision = time.Now().UTC()
	} else {
		revision = revision.UTC()
	}
	payload, err := port.NewArticleIndexJobPayload(articleID, action, event, status, isDelete, revision)
	if err != nil {
		return apperrors.Invalid("article.index_job.payload", err.Error())
	}
	jobID := uuid.NewString()
	now := time.Now().UTC()
	job := port.AIJob{
		ID:             jobID,
		Kind:           port.AIJobKindArticleIndex,
		IdempotencyKey: articleIndexIdempotencyKey(articleID, action, event, revision),
		Payload:        payload,
		Status:         port.AIJobPending,
		MaxAttempts:    port.DefaultAIJobMaxAttempts,
		RunAfter:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := a.aiJobs.Enqueue(ctx, job); err != nil {
		if apperrors.KindOf(err) != apperrors.KindInternal {
			return err
		}
		return apperrors.Unavailable("article.index_job.enqueue", err)
	}
	return nil
}

func articleIndexIdempotencyKey(articleID int, action port.ArticleIndexAction, event port.ArticleLifecycleEvent, revision time.Time) string {
	return fmt.Sprintf("article:%d:index:%s:%s:%s", articleID, action, event, strconv.FormatInt(revision.UnixNano(), 10))
}

func articleIndexMutation(article port.TArticle, published bool) (port.ArticleIndexAction, port.ArticleLifecycleEvent) {
	if article.IsDelete != 0 {
		return port.ArticleIndexDelete, port.ArticleDeleted
	}
	if port.IsPublicArticle(article.Status, article.IsDelete) {
		if published {
			return port.ArticleIndexUpsert, port.ArticlePublished
		}
		return port.ArticleIndexUpsert, port.ArticleUpdated
	}
	return port.ArticleIndexDelete, port.ArticlePrivate
}

func articleIndexMutationForDelete(isDelete int) (port.ArticleIndexAction, port.ArticleLifecycleEvent) {
	if isDelete == 0 {
		return port.ArticleIndexUpsert, port.ArticleRestored
	}
	return port.ArticleIndexDelete, port.ArticleDeleted
}

func (a *MyArticleService) enqueueArticleIndexJobs(ctx context.Context, ids []int, action port.ArticleIndexAction, event port.ArticleLifecycleEvent, status, isDelete int, revision time.Time) error {
	for _, articleID := range ids {
		if err := a.enqueueArticleIndexJob(ctx, articleID, action, event, status, isDelete, revision); err != nil {
			return err
		}
	}
	return nil
}

func (a *MyArticleService) enqueueContentUnderstandingJob(ctx context.Context, articleID int, revision time.Time) error {
	if a == nil || a.contentUnderstandingJobs == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if articleID <= 0 {
		return apperrors.Invalid("article.content_understanding_job", "article id must be positive")
	}
	if revision.IsZero() {
		revision = time.Now().UTC()
	} else {
		revision = revision.UTC()
	}
	payload, err := port.NewContentUnderstandingJobPayload(articleID, revision)
	if err != nil {
		return apperrors.Invalid("article.content_understanding_job.payload", err.Error())
	}
	now := time.Now().UTC()
	job := port.AIJob{
		ID:             uuid.NewString(),
		Kind:           port.AIJobKindContentUnderstanding,
		IdempotencyKey: fmt.Sprintf("article:%d:content_understanding:%s", articleID, strconv.FormatInt(revision.UnixNano(), 10)),
		Payload:        payload,
		Status:         port.AIJobPending,
		MaxAttempts:    port.DefaultAIJobMaxAttempts,
		RunAfter:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := a.contentUnderstandingJobs.Enqueue(ctx, job); err != nil {
		if apperrors.KindOf(err) != apperrors.KindInternal {
			return err
		}
		return apperrors.Unavailable("article.content_understanding_job.enqueue", err)
	}
	return nil
}

func (a *MyArticleService) enqueueContentUnderstandingJobs(ctx context.Context, ids []int, revision time.Time) error {
	for _, articleID := range ids {
		if err := a.enqueueContentUnderstandingJob(ctx, articleID, revision); err != nil {
			return err
		}
	}
	return nil
}

func validateArticleIndexJobIDs(ids []int) error {
	for _, id := range ids {
		if id <= 0 {
			return apperrors.Invalid("article.index_job", "article id must be positive")
		}
	}
	return nil
}
