package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"

	"github.com/meilisearch/meilisearch-go"
)

const defaultArticleChunksTaskPollInterval = 250 * time.Millisecond

// WaitForArticleChunksTasks waits for every asynchronous provisioning or swap
// task and only returns successfully completed tasks. The caller owns the
// context deadline; this helper deliberately never creates an unbounded
// background wait. It also checks the task's index UID when Meilisearch
// returns one, so a task from another release cannot satisfy this gate.
func WaitForArticleChunksTasks(ctx context.Context, reader meilisearch.TaskReader, indexUID string, tasks []*meilisearch.TaskInfo, interval time.Duration) ([]*meilisearch.Task, error) {
	if reader == nil {
		return nil, apperrors.Unavailable("search.article_chunks.tasks", fmt.Errorf("Meilisearch task reader is not configured"))
	}
	indexUID = strings.TrimSpace(indexUID)
	if indexUID == "" {
		return nil, apperrors.Invalid("search.article_chunks.tasks", "index UID is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = defaultArticleChunksTaskPollInterval
	}
	if len(tasks) == 0 {
		return []*meilisearch.Task{}, nil
	}

	completed := make([]*meilisearch.Task, 0, len(tasks))
	seen := make(map[int64]struct{}, len(tasks))
	for position, taskInfo := range tasks {
		if taskInfo == nil || taskInfo.TaskUID <= 0 {
			return nil, apperrors.Invalid("search.article_chunks.tasks", fmt.Sprintf("task %d has no valid task UID", position))
		}
		if taskInfo.IndexUID != "" && taskInfo.IndexUID != indexUID {
			return nil, apperrors.Conflict("search.article_chunks.tasks", fmt.Sprintf("task %d belongs to index %q, want %q", taskInfo.TaskUID, taskInfo.IndexUID, indexUID))
		}
		if _, duplicate := seen[taskInfo.TaskUID]; duplicate {
			return nil, apperrors.Conflict("search.article_chunks.tasks", fmt.Sprintf("task %d was returned more than once", taskInfo.TaskUID))
		}
		seen[taskInfo.TaskUID] = struct{}{}

		task, err := reader.WaitForTaskWithContext(ctx, taskInfo.TaskUID, interval)
		if err != nil {
			return nil, apperrors.WrapUnavailable("search.article_chunks.tasks.wait", err)
		}
		if task == nil {
			return nil, apperrors.Unavailable("search.article_chunks.tasks.wait", fmt.Errorf("task %d returned an empty response", taskInfo.TaskUID))
		}
		if ArticleChunksTaskUID(task) != taskInfo.TaskUID {
			return nil, apperrors.Conflict("search.article_chunks.tasks", fmt.Sprintf("task response UID does not match requested task %d", taskInfo.TaskUID))
		}
		if task.IndexUID != "" && task.IndexUID != indexUID {
			return nil, apperrors.Conflict("search.article_chunks.tasks", fmt.Sprintf("task %d completed for index %q, want %q", taskInfo.TaskUID, task.IndexUID, indexUID))
		}
		if task.Status != meilisearch.TaskStatusSucceeded {
			return nil, apperrors.Unavailable("search.article_chunks.tasks", fmt.Errorf("task %d finished with status %q", taskInfo.TaskUID, task.Status))
		}
		completed = append(completed, task)
	}
	return completed, nil
}

// Meilisearch's current task response uses `uid`, while older SDK fixtures
// and some compatible servers use `taskUid`. Normalize both at the adapter
// boundary so a successful real task is never reported as UID 0.
func ArticleChunksTaskUID(task *meilisearch.Task) int64 {
	if task == nil {
		return 0
	}
	if task.UID > 0 {
		return task.UID
	}
	return task.TaskUID
}
