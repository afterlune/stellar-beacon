package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type fakeScheduledPublishRepo struct {
	retryable  []port.ScheduledPublish
	due        []port.ScheduledPublish
	queued     []int
	failed     []fakePublishFailure
	suppressed []int
}

type fakePublishFailure struct {
	recordID int
	retryAt  *time.Time
	message  string
}

func (f *fakeScheduledPublishRepo) PublishDueScheduledArticles(context.Context, time.Time, int) ([]port.ScheduledPublish, error) {
	return f.due, nil
}
func (f *fakeScheduledPublishRepo) ListRetryableScheduledPublishes(context.Context, time.Time, int) ([]port.ScheduledPublish, error) {
	return f.retryable, nil
}
func (f *fakeScheduledPublishRepo) MarkScheduledNotificationQueued(_ context.Context, recordID int, _ time.Time) error {
	f.queued = append(f.queued, recordID)
	return nil
}
func (f *fakeScheduledPublishRepo) MarkScheduledNotificationFailed(_ context.Context, recordID int, message string, retryAt *time.Time) error {
	f.failed = append(f.failed, fakePublishFailure{recordID: recordID, retryAt: retryAt, message: message})
	return nil
}
func (f *fakeScheduledPublishRepo) MarkScheduledNotificationSuppressed(_ context.Context, recordID int, _ string) error {
	f.suppressed = append(f.suppressed, recordID)
	return nil
}

type flakyNewsletter struct {
	failArticleID int
	calls         []int
}

func (f *flakyNewsletter) EnqueueArticle(_ context.Context, articleID int) error {
	f.calls = append(f.calls, articleID)
	if articleID == f.failArticleID {
		return errors.New("smtp unavailable")
	}
	return nil
}

func TestRunScheduledPublishQueuesRetriesAndSuppressesHiddenContent(t *testing.T) {
	repo := &fakeScheduledPublishRepo{
		retryable: []port.ScheduledPublish{{RecordID: 1, ArticleID: 11, NotificationAttempts: 1}},
		due: []port.ScheduledPublish{
			{RecordID: 2, ArticleID: 12, ModerationStatus: "visible"},
			{RecordID: 3, ArticleID: 13, ModerationStatus: "hidden"},
		},
	}
	newsletter := &flakyNewsletter{failArticleID: 12}

	result, err := runScheduledPublish(context.Background(), repo, newsletter)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Processed || result.Message != "published=2 queued=1 retried=1 failed=1 suppressed=1" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(repo.queued) != 1 || repo.queued[0] != 1 {
		t.Fatalf("unexpected queued records: %v", repo.queued)
	}
	if len(repo.failed) != 1 || repo.failed[0].recordID != 2 || repo.failed[0].retryAt == nil {
		t.Fatalf("unexpected failed records: %+v", repo.failed)
	}
	if len(repo.suppressed) != 1 || repo.suppressed[0] != 3 {
		t.Fatalf("unexpected suppressed records: %v", repo.suppressed)
	}
}

func TestRunScheduledPublishUsesTerminalFailureAfterFinalRetryGap(t *testing.T) {
	repo := &fakeScheduledPublishRepo{due: []port.ScheduledPublish{{RecordID: 4, ArticleID: 14, NotificationAttempts: len(scheduledPublishRetryDelays)}}}
	newsletter := &flakyNewsletter{failArticleID: 14}

	if _, err := runScheduledPublish(context.Background(), repo, newsletter); err != nil {
		t.Fatal(err)
	}
	if len(repo.failed) != 1 || repo.failed[0].retryAt != nil {
		t.Fatalf("expected terminal failure, got %+v", repo.failed)
	}
}
