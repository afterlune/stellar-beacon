package ormInit

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	xormlog "xorm.io/xorm/log"
)

func TestSlogXORMLoggerDoesNotEmitSQLArgumentsByDefault(t *testing.T) {
	var output bytes.Buffer
	logger := newSlogXORMLogger(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	logger.SetLevel(xormlog.LOG_INFO)

	logger.AfterSQL(xormlog.LogContext{
		Ctx:         context.Background(),
		SQL:         "SELECT * FROM t_user WHERE password = ?",
		Args:        []any{"password=super-secret"},
		ExecuteTime: time.Millisecond,
	})
	if output.Len() != 0 {
		t.Fatalf("SQL should be disabled by default, got %q", output.String())
	}
}

func TestSlogXORMLoggerLogsStructuredSQLWithoutArgumentsWhenEnabled(t *testing.T) {
	var output bytes.Buffer
	logger := newSlogXORMLogger(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	logger.SetLevel(xormlog.LOG_INFO)
	logger.ShowSQL(true)
	logger.AfterSQL(xormlog.LogContext{
		Ctx:         context.Background(),
		SQL:         "SELECT * FROM t_user WHERE password = ?",
		Args:        []any{"password=super-secret"},
		ExecuteTime: time.Millisecond,
	})
	text := output.String()
	if !strings.Contains(text, "database query") || !strings.Contains(text, "arg_count=1") || !strings.Contains(text, "password = ?") {
		t.Fatalf("unexpected structured SQL log: %q", text)
	}
	if strings.Contains(text, "super-secret") || strings.Contains(text, "password=super-secret") {
		t.Fatalf("SQL argument leaked into log: %q", text)
	}
}

func TestSlogXORMLoggerRoutesQueryErrorsToSlog(t *testing.T) {
	var output bytes.Buffer
	logger := newSlogXORMLogger(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	logger.SetLevel(xormlog.LOG_INFO)
	logger.ShowSQL(true)
	secret := "database response password=super-secret"
	logger.AfterSQL(xormlog.LogContext{SQL: "SELECT 1", Err: errors.New(secret)})
	if !strings.Contains(output.String(), "database query failed") || !strings.Contains(output.String(), "error_code=internal_error") {
		t.Fatalf("query error was not routed to slog: %q", output.String())
	}
	if strings.Contains(output.String(), secret) || strings.Contains(output.String(), "super-secret") {
		t.Fatalf("query error detail leaked into log: %q", output.String())
	}
}
