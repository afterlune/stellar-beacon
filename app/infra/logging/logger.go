package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config describes the process-wide slog output. The file writer is kept
// separate from the console writer so operational logs remain readable while
// error logs remain machine searchable.
type Config struct {
	ConsoleLevel slog.Level
	FilePath     string
	MaxSize      int
	MaxBackups   int
	MaxAge       int
	Compress     bool
}

// Init installs the process-wide slog logger and returns an idempotent close
// function for the rotating file writer. No files are created at package init
// time; callers decide when logging is ready and where runtime files belong.
func Init(config Config) (func() error, error) {
	if strings.TrimSpace(config.FilePath) == "" {
		return nil, fmt.Errorf("log file path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(config.FilePath), 0o750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	if config.MaxSize <= 0 {
		config.MaxSize = 50
	}
	if config.MaxBackups < 0 {
		config.MaxBackups = 5
	}
	if config.MaxAge < 0 {
		config.MaxAge = 7
	}

	fileWriter := &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	options := &slog.HandlerOptions{
		AddSource:   true,
		ReplaceAttr: redactSensitiveAttr,
	}
	fileOptions := *options
	fileOptions.Level = slog.LevelError
	consoleOptions := *options
	consoleOptions.Level = config.ConsoleLevel

	fileHandler := slog.NewJSONHandler(fileWriter, &fileOptions)
	consoleHandler := slog.NewTextHandler(os.Stdout, &consoleOptions)
	slog.SetDefault(slog.New(newFanoutHandler(fileHandler, consoleHandler)))

	var closeOnce sync.Once
	var closeErr error
	return func() error {
		closeOnce.Do(func() {
			closeErr = fileWriter.Close()
		})
		return closeErr
	}, nil
}

type fanoutHandler struct {
	handlers []slog.Handler
}

func newFanoutHandler(handlers ...slog.Handler) slog.Handler {
	return &fanoutHandler{handlers: handlers}
}

func (h *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *fanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var firstErr error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return newFanoutHandler(handlers...)
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return newFanoutHandler(handlers...)
}

func redactSensitiveAttr(groups []string, attr slog.Attr) slog.Attr {
	key := strings.ToLower(attr.Key)
	for _, sensitive := range []string{
		"password",
		"secret",
		"token",
		"authorization",
		"api_key",
		"apikey",
	} {
		if strings.Contains(key, sensitive) {
			return slog.String(attr.Key, "[REDACTED]")
		}
	}
	return attr
}
