package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/cache"
	"github.com/redis/go-redis/v9"
)

type fakeJobRepository struct {
	jobs []entity.TJob
}

func (f *fakeJobRepository) Get(context.Context, int) (entity.TJob, error) {
	return entity.TJob{}, errors.New("not implemented")
}
func (f *fakeJobRepository) List(context.Context, int, int, port.JobFilter) ([]entity.TJob, int, error) {
	return f.jobs, len(f.jobs), nil
}
func (f *fakeJobRepository) ListEnabled(context.Context) ([]entity.TJob, error) {
	return f.jobs, nil
}
func (f *fakeJobRepository) ListGroups(context.Context) ([]string, error)    { return nil, nil }
func (f *fakeJobRepository) SaveOrUpdate(context.Context, entity.TJob) error { return nil }
func (f *fakeJobRepository) Delete(context.Context, []int) error             { return nil }
func (f *fakeJobRepository) UpdateStatus(context.Context, int, int) error    { return nil }

type fakeJobLogs struct {
	mu   sync.Mutex
	logs []entity.TJobLog
}

func (f *fakeJobLogs) List(context.Context, int, int, port.JobLogFilter) ([]entity.TJobLog, int64, error) {
	return nil, 0, nil
}
func (f *fakeJobLogs) Create(_ context.Context, entry entity.TJobLog) error {
	f.mu.Lock()
	f.logs = append(f.logs, entry)
	f.mu.Unlock()
	return nil
}
func (f *fakeJobLogs) Delete(context.Context, []int) error          { return nil }
func (f *fakeJobLogs) Clean(context.Context) error                  { return nil }
func (f *fakeJobLogs) CleanBefore(context.Context, time.Time) error { return nil }
func (f *fakeJobLogs) ListGroups(context.Context) (string, error)   { return "", nil }

func TestSchedulerValidatesStandardCronOnly(t *testing.T) {
	scheduler := NewScheduler(&fakeJobRepository{}, &fakeJobLogs{}, nil)
	next, err := scheduler.NextRun("*/10 * * * *", time.Date(2026, 9, 18, 8, 1, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("valid cron rejected: %v", err)
	}
	if got := next.Format("15:04"); got != "08:10" {
		t.Fatalf("next run = %s, want 08:10", got)
	}
	for _, expression := range []string{"0 0/10 * * * ?", "0 3 * * * *"} {
		if _, err := scheduler.NextRun(expression, time.Now()); err == nil {
			t.Fatalf("legacy expression %q should be rejected", expression)
		}
	}
}

func TestSchedulerRunRecordsLog(t *testing.T) {
	logs := &fakeJobLogs{}
	scheduler := NewScheduler(&fakeJobRepository{}, logs, nil)
	if err := scheduler.Register(port.JobTarget{Target: "test.success", Name: "Success"}, func(context.Context) (port.JobRunResult, error) {
		return port.JobRunResult{Processed: true, Message: "worked"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	result, err := scheduler.Run(context.Background(), entity.TJob{Id: 7, InvokeTarget: "test.success", Concurrent: 0}, "manual")
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !result.Processed || result.Message != "worked" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(logs.logs) != 1 || logs.logs[0].Status != 0 || logs.logs[0].InvokeTarget != "test.success" {
		t.Fatalf("unexpected logs: %+v", logs.logs)
	}
}

func TestSchedulerRunRejectsUnsupportedTarget(t *testing.T) {
	scheduler := NewScheduler(&fakeJobRepository{}, &fakeJobLogs{}, nil)
	if _, err := scheduler.Run(context.Background(), entity.TJob{Id: 1, InvokeTarget: "missing.target"}, "manual"); err == nil {
		t.Fatal("unsupported target should fail")
	}
}

func TestSchedulerRunRecoversHandlerPanic(t *testing.T) {
	logs := &fakeJobLogs{}
	scheduler := NewScheduler(&fakeJobRepository{}, logs, nil)
	if err := scheduler.Register(port.JobTarget{Target: "test.panic", Name: "Panic"}, func(context.Context) (port.JobRunResult, error) {
		panic("boom")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := scheduler.Run(context.Background(), entity.TJob{Id: 8, InvokeTarget: "test.panic"}, "manual"); err == nil {
		t.Fatal("handler panic should become an execution error")
	}
	if len(logs.logs) != 1 || logs.logs[0].Status != 1 {
		t.Fatalf("panic was not logged: %+v", logs.logs)
	}
}

func TestSchedulerPreventsLocalOverlap(t *testing.T) {
	scheduler := NewScheduler(&fakeJobRepository{}, &fakeJobLogs{}, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	if err := scheduler.Register(port.JobTarget{Target: "test.blocking", Name: "Blocking"}, func(context.Context) (port.JobRunResult, error) {
		close(started)
		<-release
		return port.JobRunResult{Processed: true}, nil
	}); err != nil {
		t.Fatal(err)
	}
	job := entity.TJob{Id: 9, InvokeTarget: "test.blocking", Concurrent: 0}
	firstDone := make(chan error, 1)
	go func() {
		_, err := scheduler.Run(context.Background(), job, "manual")
		firstDone <- err
	}()
	<-started
	second, err := scheduler.Run(context.Background(), job, "manual")
	if err != nil {
		t.Fatalf("overlap check failed: %v", err)
	}
	if second.Processed || second.Message == "" {
		t.Fatalf("expected overlap result, got %+v", second)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first run failed: %v", err)
	}
}

func TestSchedulerHonorsRedisLock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	redisCache := cache.NewRedisCacheWithClient(client)
	if acquired, err := redisCache.SetNX(context.Background(), "stellar-beacon:job:11:lock", "1", time.Minute); err != nil || !acquired {
		t.Fatalf("seed lock: acquired=%v err=%v", acquired, err)
	}
	called := false
	scheduler := NewScheduler(&fakeJobRepository{}, &fakeJobLogs{}, redisCache)
	if err := scheduler.Register(port.JobTarget{Target: "test.locked", Name: "Locked"}, func(context.Context) (port.JobRunResult, error) {
		called = true
		return port.JobRunResult{Processed: true}, nil
	}); err != nil {
		t.Fatal(err)
	}
	result, err := scheduler.Run(context.Background(), entity.TJob{Id: 11, InvokeTarget: "test.locked", Concurrent: 0}, "manual")
	if err != nil {
		t.Fatalf("locked run failed: %v", err)
	}
	if called || result.Processed {
		t.Fatalf("locked handler should not run: called=%v result=%+v", called, result)
	}
}

func TestSchedulerReloadRegistersEnabledJobs(t *testing.T) {
	repo := &fakeJobRepository{jobs: []entity.TJob{{Id: 1, InvokeTarget: "test.reload", CronExpression: "* * * * *", Status: 1}}}
	scheduler := NewScheduler(repo, &fakeJobLogs{}, nil)
	if err := scheduler.Register(port.JobTarget{Target: "test.reload", Name: "Reload"}, func(context.Context) (port.JobRunResult, error) {
		return port.JobRunResult{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second)
		defer stopCancel()
		if err := scheduler.Stop(stopCtx); err != nil {
			t.Fatalf("stop scheduler: %v", err)
		}
	}()
	scheduler.mu.Lock()
	entryCount := len(scheduler.entries)
	scheduler.mu.Unlock()
	if entryCount != 1 {
		t.Fatalf("registered entries = %d, want 1", entryCount)
	}
}
