package agent

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"context"
	"testing"
	"time"
)

func TestForgottenArticleSelectorIsPublicStableAndBounded(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	selector, err := NewForgottenArticleSelector(ForgottenArticleSelectorConfig{
		StaleAfter:    30 * 24 * time.Hour,
		MaxCandidates: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	sources := []port.ArticleIndexSource{
		{Article: entity.TArticle{Id: 9, Status: 1, IsDelete: 0, UpdateTime: now.Add(-60 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 4, Status: 1, IsDelete: 0, UpdateTime: now.Add(-60 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0, CreateTime: now.Add(-90 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 8, Status: 1, IsDelete: 0, UpdateTime: now.Add(-2 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 10, Status: 1, IsDelete: 0, IsTop: 1, UpdateTime: now.Add(-90 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 11, Status: 1, IsDelete: 0, IsFeatured: 1, UpdateTime: now.Add(-90 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 12, Status: 2, IsDelete: 0, UpdateTime: now.Add(-90 * 24 * time.Hour)}},
		{Article: entity.TArticle{Id: 13, Status: 1, IsDelete: 1, UpdateTime: now.Add(-90 * 24 * time.Hour)}},
	}
	got := selector.Select(now, sources)
	if len(got) != 3 {
		t.Fatalf("selected %d candidates, want 3: %+v", len(got), got)
	}
	if got[0].Article.Id != 7 || got[1].Article.Id != 4 || got[2].Article.Id != 9 {
		t.Fatalf("selection order = [%d %d %d], want [7 4 9]", got[0].Article.Id, got[1].Article.Id, got[2].Article.Id)
	}
}

func TestBehaviorSchedulerUsesSeparateForgottenTrigger(t *testing.T) {
	policy := behaviorTestPolicy(t)
	selector, err := NewForgottenArticleSelector(ForgottenArticleSelectorConfig{
		StaleAfter:    30 * 24 * time.Hour,
		MaxCandidates: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	jobs := &behaviorJobsFake{}
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	scheduler, err := NewBehaviorScheduler(BehaviorSchedulerDeps{
		Sources:           behaviorSourceFake{sources: []port.ArticleIndexSource{{Article: entity.TArticle{Id: 7, Status: 1, IsDelete: 0, UpdateTime: now.Add(-90 * 24 * time.Hour)}}}},
		Jobs:              jobs,
		Policy:            policy,
		ForgottenSelector: selector,
		BatchSize:         10,
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	next, count, err := scheduler.ScanForgotten(context.Background(), 0)
	if err != nil || next != 7 || count != 2 {
		t.Fatalf("forgotten scan = next:%d count:%d err:%v", next, count, err)
	}
	for _, job := range jobs.jobs {
		payload, decodeErr := DecodeAgentReadingTask(job.Payload)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if payload.Trigger != port.AgentBehaviorTriggerForgotten {
			t.Fatalf("payload trigger = %q, want forgotten", payload.Trigger)
		}
	}
}
