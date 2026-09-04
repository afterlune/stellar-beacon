package ormInit

import (
	apperrors "benetnasch/app/domain/errors"
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	xormlog "xorm.io/xorm/log"
)

// slogXORMLogger adapts xorm's logger contract to the process-wide standard
// library slog logger. SQL arguments are deliberately not included in the
// structured record: they can contain passwords, prompts, or user content.
// SQL text is emitted only when the explicit database.show_sql switch is on.
type slogXORMLogger struct {
	logger  *slog.Logger
	level   atomic.Int32
	showSQL atomic.Bool
}

var _ xormlog.ContextLogger = (*slogXORMLogger)(nil)

func newSlogXORMLogger(logger *slog.Logger) *slogXORMLogger {
	if logger == nil {
		logger = slog.Default()
	}
	adapter := &slogXORMLogger{logger: logger}
	adapter.level.Store(int32(xormlog.LOG_WARNING))
	return adapter
}

func (l *slogXORMLogger) allows(level xormlog.LogLevel) bool {
	return l != nil && xormlog.LogLevel(l.level.Load()) <= level
}

func (l *slogXORMLogger) Debug(v ...any) {
	if l.allows(xormlog.LOG_DEBUG) {
		l.logger.Debug(fmt.Sprint(v...))
	}
}

func (l *slogXORMLogger) Debugf(format string, v ...any) {
	if l.allows(xormlog.LOG_DEBUG) {
		l.logger.Debug(fmt.Sprintf(format, v...))
	}
}

func (l *slogXORMLogger) Info(v ...any) {
	if l.allows(xormlog.LOG_INFO) {
		l.logger.Info(fmt.Sprint(v...))
	}
}

func (l *slogXORMLogger) Infof(format string, v ...any) {
	if l.allows(xormlog.LOG_INFO) {
		l.logger.Info(fmt.Sprintf(format, v...))
	}
}

func (l *slogXORMLogger) Warn(v ...any) {
	if l.allows(xormlog.LOG_WARNING) {
		l.logger.Warn(fmt.Sprint(v...))
	}
}

func (l *slogXORMLogger) Warnf(format string, v ...any) {
	if l.allows(xormlog.LOG_WARNING) {
		l.logger.Warn(fmt.Sprintf(format, v...))
	}
}

func (l *slogXORMLogger) Error(v ...any) {
	if l.allows(xormlog.LOG_ERR) {
		l.logger.Error(fmt.Sprint(v...))
	}
}

func (l *slogXORMLogger) Errorf(format string, v ...any) {
	if l.allows(xormlog.LOG_ERR) {
		l.logger.Error(fmt.Sprintf(format, v...))
	}
}

func (l *slogXORMLogger) Level() xormlog.LogLevel {
	if l == nil {
		return xormlog.LOG_UNKNOWN
	}
	return xormlog.LogLevel(l.level.Load())
}

func (l *slogXORMLogger) SetLevel(level xormlog.LogLevel) {
	if l != nil {
		l.level.Store(int32(level))
	}
}

func (l *slogXORMLogger) ShowSQL(show ...bool) {
	if l == nil {
		return
	}
	value := true
	if len(show) > 0 {
		value = show[0]
	}
	l.showSQL.Store(value)
}

func (l *slogXORMLogger) IsShowSQL() bool {
	return l != nil && l.showSQL.Load()
}

func (l *slogXORMLogger) BeforeSQL(xormlog.LogContext) {}

func (l *slogXORMLogger) AfterSQL(hook xormlog.LogContext) {
	if l == nil || !l.IsShowSQL() || !l.allows(xormlog.LOG_INFO) {
		return
	}
	ctx := hook.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	attrs := []any{
		"sql", hook.SQL,
		"arg_count", len(hook.Args),
		"duration", hook.ExecuteTime,
	}
	if hook.Err != nil {
		attrs = append(attrs, "error_code", apperrors.SafeCode(hook.Err))
		l.logger.ErrorContext(ctx, "database query failed", attrs...)
		return
	}
	l.logger.InfoContext(ctx, "database query", attrs...)
}
