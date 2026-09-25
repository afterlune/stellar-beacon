package port

import (
	"context"
	"time"
)

// StudioActivationReminderRepository creates due activation reminders in the
// unified notification inbox. Implementations must make repeated runs
// idempotent because the scheduler may retry a job after a partial failure.
type StudioActivationReminderRepository interface {
	CreateDueStudioActivationReminders(ctx context.Context, now time.Time, limit int) (int, error)
}
