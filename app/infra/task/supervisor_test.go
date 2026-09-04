package task

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSupervisorStopsWorkersWithContext(t *testing.T) {
	started := make(chan struct{})
	supervisor := NewSupervisor(context.Background())
	if err := supervisor.Add("test", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Stop(stopCtx); err != nil {
		t.Fatal(err)
	}
}

func TestSupervisorOwnsWorkerFailureWithoutRestartLoop(t *testing.T) {
	supervisor := NewSupervisor(context.Background())
	if err := supervisor.Add("failure", func(context.Context) error { return errors.New("expected") }); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Add("late", func(context.Context) error { return nil }); !errors.Is(err, ErrSupervisorStarted) {
		t.Fatalf("Add after Start() = %v, want %v", err, ErrSupervisorStarted)
	}
}

func TestSupervisorContainsWorkerPanicWithoutRestartingIt(t *testing.T) {
	var runs atomic.Int32
	supervisor := NewSupervisor(context.Background())
	if err := supervisor.Add("panic", func(context.Context) error {
		runs.Add(1)
		panic("expected worker panic")
	}); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}
	if got := runs.Load(); got != 1 {
		t.Fatalf("panicking worker ran %d times, want exactly once", got)
	}
}

func TestSupervisorLogsStableCodesWithoutWorkerErrorDetails(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	secretFailure := "provider response prompt=private-content api_key=should-not-log"
	supervisor := NewSupervisor(context.Background())
	if err := supervisor.Add("failure", func(context.Context) error { return errors.New(secretFailure) }); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}

	output := logs.String()
	if strings.Contains(output, secretFailure) || strings.Contains(output, "private-content") {
		t.Fatalf("worker error detail leaked into logs: %s", output)
	}
	if !strings.Contains(output, "error_code=internal_error") {
		t.Fatalf("stable worker error code missing from logs: %s", output)
	}
}

func TestSupervisorDoesNotLogPanicValue(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	panicDetail := "panic prompt=private-content"
	supervisor := NewSupervisor(context.Background())
	if err := supervisor.Add("panic", func(context.Context) error {
		panic(panicDetail)
	}); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := supervisor.Wait(waitCtx); err != nil {
		t.Fatal(err)
	}

	output := logs.String()
	if strings.Contains(output, panicDetail) || !strings.Contains(output, "error_code=worker_panic") {
		t.Fatalf("panic logging is not sanitized: %s", output)
	}
}
