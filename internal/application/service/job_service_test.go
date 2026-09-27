package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type jobRepoStub struct {
	saved   entity.TJob
	job     entity.TJob
	jobs    []entity.TJob
	deleted []int
	status  int
}

func (r *jobRepoStub) Get(context.Context, int) (entity.TJob, error) { return r.job, nil }
func (r *jobRepoStub) List(context.Context, int, int, port.JobFilter) ([]entity.TJob, int, error) {
	return r.jobs, len(r.jobs), nil
}
func (r *jobRepoStub) ListEnabled(context.Context) ([]entity.TJob, error) { return nil, nil }
func (r *jobRepoStub) ListGroups(context.Context) ([]string, error)       { return []string{"system"}, nil }
func (r *jobRepoStub) SaveOrUpdate(_ context.Context, job entity.TJob) error {
	r.saved = job
	return nil
}
func (r *jobRepoStub) Delete(_ context.Context, ids []int) error {
	r.deleted = ids
	return nil
}
func (r *jobRepoStub) UpdateStatus(_ context.Context, _ int, status int) error {
	r.status = status
	return nil
}

type jobSchedulerStub struct {
	targets   []port.JobTarget
	supported bool
	next      time.Time
	run       port.JobRunResult
	reloadErr error
	reloads   int
}

func (s *jobSchedulerStub) Targets() []port.JobTarget  { return s.targets }
func (s *jobSchedulerStub) SupportsTarget(string) bool { return s.supported }
func (s *jobSchedulerStub) NextRun(string, time.Time) (time.Time, error) {
	if !s.supported {
		return time.Time{}, errors.New("unsupported target")
	}
	return s.next, nil
}
func (s *jobSchedulerStub) Reload(context.Context) error {
	s.reloads++
	return s.reloadErr
}
func (s *jobSchedulerStub) Run(context.Context, entity.TJob, string) (port.JobRunResult, error) {
	return s.run, nil
}
func (s *jobSchedulerStub) Start(context.Context) error { return nil }
func (s *jobSchedulerStub) Stop(context.Context) error  { return nil }
func (s *jobSchedulerStub) Ready() bool                 { return true }

func TestMyJobServiceSaveJobNormalizesInputAndReloads(t *testing.T) {
	repository := &jobRepoStub{}
	scheduler := &jobSchedulerStub{supported: true}
	service := NewJobService(repository, scheduler)
	input := JobInput{
		JobName: "  daily digest ", JobGroup: " system ", InvokeTarget: " digest.run ",
		CronExpression: " 0 0 * * * ", Concurrent: 1, Status: 1, Remark: " keep ",
	}

	if err := service.SaveJob(context.Background(), input); err != nil {
		t.Fatalf("SaveJob() error = %v", err)
	}
	if repository.saved.JobName != "daily digest" || repository.saved.JobGroup != "system" ||
		repository.saved.InvokeTarget != "digest.run" || repository.saved.CronExpression != "0 0 * * *" ||
		repository.saved.Remark != "keep" || repository.saved.MisfirePolicy != 3 {
		t.Fatalf("saved job was not normalized: %+v", repository.saved)
	}
	if scheduler.reloads != 1 {
		t.Fatalf("scheduler reload count = %d, want 1", scheduler.reloads)
	}
}

func TestMyJobServiceRejectsUnregisteredTarget(t *testing.T) {
	repository := &jobRepoStub{}
	service := NewJobService(repository, &jobSchedulerStub{})
	err := service.SaveJob(context.Background(), JobInput{
		JobName: "digest", JobGroup: "system", InvokeTarget: "unknown", CronExpression: "0 0 * * *",
	})
	if err == nil {
		t.Fatal("SaveJob() accepted an unregistered target")
	}
	if repository.saved.JobName != "" {
		t.Fatalf("repository was called for invalid input: %+v", repository.saved)
	}
}

func TestMyJobServiceListJobsAddsRuntimeDetails(t *testing.T) {
	next := time.Date(2026, time.September, 27, 12, 0, 0, 0, time.UTC)
	repository := &jobRepoStub{jobs: []entity.TJob{{Id: 7, InvokeTarget: "digest.run", CronExpression: "0 0 * * *"}}}
	service := NewJobService(repository, &jobSchedulerStub{supported: true, next: next})

	jobs, count, err := service.ListJobs(context.Background(), 1, 10, port.JobFilter{})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if count != 1 || len(jobs) != 1 || !jobs[0].CanRunOnce || jobs[0].NextValidTime == nil || !jobs[0].NextValidTime.Equal(next) {
		t.Fatalf("unexpected job details: count=%d jobs=%+v", count, jobs)
	}
}

func TestMyJobServiceRunJobReturnsTypedOutcome(t *testing.T) {
	repository := &jobRepoStub{job: entity.TJob{Id: 9, InvokeTarget: "digest.run"}}
	scheduler := &jobSchedulerStub{supported: true, run: port.JobRunResult{Processed: true, Message: "sent"}}
	service := NewJobService(repository, scheduler)

	outcome, err := service.RunJob(context.Background(), 9)
	if err != nil {
		t.Fatalf("RunJob() error = %v", err)
	}
	if outcome.JobID != 9 || outcome.Target != "digest.run" || !outcome.Processed || outcome.Message != "sent" {
		t.Fatalf("unexpected run outcome: %+v", outcome)
	}
}
