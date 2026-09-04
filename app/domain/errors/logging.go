package errors

import (
	"context"
	stderrors "errors"
)

// SafeCode returns a stable, detail-free error code for operational logs.
// Error strings can contain provider response bodies, user input, credentials,
// SQL fragments, or other request data, so callers must not pass the raw
// error as a slog attribute at a trust boundary.
func SafeCode(err error) string {
	if err == nil {
		return ""
	}
	if code := AICodeOf(err); code != "" {
		return string(code)
	}
	if stderrors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return "context_deadline_exceeded"
	}
	switch KindOf(err) {
	case KindValidation:
		return "validation_error"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindUnavailable:
		return "unavailable"
	default:
		return "internal_error"
	}
}
