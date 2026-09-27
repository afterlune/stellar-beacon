package model

import (
	"context"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"log/slog"
	"sync"
	"testing"
)

func TestResultFromErrorDoesNotExposeInfrastructureDetails(t *testing.T) {
	result := ResultFromError(apperrors.Unavailable("article.list", assertErr("secret database detail")))
	if result.Flag {
		t.Fatal("infrastructure error must be a failed result")
	}
	if result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected public message: %q", result.Message)
	}
	if result.Message == "secret database detail" {
		t.Fatal("internal error detail leaked to response")
	}
}

// A canceled request is not a service failure: it must keep the exact wire
// envelope clients already receive while staying out of the error log.
func TestResultFromErrorKeepsCanceledRequestsOutOfTheErrorLog(t *testing.T) {
	handler := installRecordingSlog(t)

	result := ResultFromError(apperrors.Canceled("article.list", context.Canceled))
	if result.Flag {
		t.Fatal("canceled request must not report success")
	}
	if result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected public message: %q", result.Message)
	}
	if wire := resultCode(result.Code, result.Flag); wire != "OPERATION_FAILED" {
		t.Fatalf("canceled request changed the wire code: %q", wire)
	}
	if handler.hasLevel(slog.LevelError) {
		t.Fatal("canceled request must not be logged as an application failure")
	}
}

func TestResultFromErrorStillLogsRealFailures(t *testing.T) {
	handler := installRecordingSlog(t)

	_ = ResultFromError(apperrors.Unavailable("article.list", assertErr("boom")))
	if !handler.hasLevel(slog.LevelError) {
		t.Fatal("infrastructure failure must stay at error level")
	}
}

func installRecordingSlog(t *testing.T) *recordingHandler {
	t.Helper()
	handler := &recordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return handler
}

type recordingHandler struct {
	mu     sync.Mutex
	levels []slog.Level
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.levels = append(h.levels, record.Level)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

func (h *recordingHandler) hasLevel(level slog.Level) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, recorded := range h.levels {
		if recorded == level {
			return true
		}
	}
	return false
}

type assertionError string

func (e assertionError) Error() string { return string(e) }

func assertErr(message string) error { return assertionError(message) }
