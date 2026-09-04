package task

import (
	"context"
	"errors"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

func TestAllowlistedJobRunnerOnlyRunsRegisteredTargets(t *testing.T) {
	called := 0
	runner, err := NewAllowlistedJobRunnerWithTimeout(map[string]OneShotHandler{
		port.ManualJobTargetContentUnderstanding: func(context.Context) (bool, error) {
			called++
			return true, nil
		},
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !runner.CanRun(port.ManualJobTargetContentUnderstanding) || runner.CanRun("os/exec") {
		t.Fatal("runner exposed an unsafe or missing target")
	}
	outcome, err := runner.Run(context.Background(), port.JobRunRequest{
		ID:           7,
		InvokeTarget: port.ManualJobTargetContentUnderstanding,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Processed || outcome.JobID != 7 || called != 1 {
		t.Fatalf("outcome=%+v called=%d", outcome, called)
	}
}

func TestAllowlistedJobRunnerRejectsUnknownTargetAndTimesOut(t *testing.T) {
	runner, err := NewAllowlistedJobRunnerWithTimeout(map[string]OneShotHandler{
		port.ManualJobTargetDream: func(ctx context.Context) (bool, error) {
			<-ctx.Done()
			return false, ctx.Err()
		},
	}, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), port.JobRunRequest{ID: 1, InvokeTarget: "arbitrary.target"}); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("unknown target error=%v", err)
	}
	_, err = runner.Run(context.Background(), port.JobRunRequest{ID: 1, InvokeTarget: port.ManualJobTargetDream})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error=%v", err)
	}
}

func TestAllowlistedJobRunnerCopiesHandlerMap(t *testing.T) {
	handlers := map[string]OneShotHandler{
		port.ManualJobTargetDream: func(context.Context) (bool, error) { return false, nil },
	}
	runner, err := NewAllowlistedJobRunner(handlers)
	if err != nil {
		t.Fatal(err)
	}
	delete(handlers, port.ManualJobTargetDream)
	if !runner.CanRun(port.ManualJobTargetDream) {
		t.Fatal("runner was affected by caller map mutation")
	}
}
