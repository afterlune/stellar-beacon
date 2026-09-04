package port

import (
	"context"
	"errors"
	"strings"
	"time"
)

// EventEnvelope is the provider-neutral wire and replay identity. Seq is
// allocated by the event store; -1 means the envelope has not been persisted.
// Client-supplied sequence values must never be trusted as replay cursors.
type EventEnvelope struct {
	EventID       string
	SessionID     string
	TurnID        string
	RunID         string
	Seq           int64
	Type          string
	SchemaVersion int
	Replayable    bool
	TraceID       string
	Payload       []byte
	OccurredAt    time.Time
}

func NewEventEnvelope(eventID, sessionID, turnID, eventType string, payload []byte) (EventEnvelope, error) {
	event := EventEnvelope{
		EventID:       strings.TrimSpace(eventID),
		SessionID:     strings.TrimSpace(sessionID),
		TurnID:        strings.TrimSpace(turnID),
		Seq:           -1,
		Type:          strings.TrimSpace(eventType),
		SchemaVersion: 1,
		Replayable:    true,
		Payload:       append([]byte(nil), payload...),
		OccurredAt:    time.Now().UTC(),
	}
	if err := event.Validate(); err != nil {
		return EventEnvelope{}, err
	}
	return event, nil
}

func (e EventEnvelope) Validate() error {
	switch {
	case strings.TrimSpace(e.EventID) == "":
		return errors.New("event id is required")
	case strings.TrimSpace(e.SessionID) == "":
		return errors.New("session id is required")
	case strings.TrimSpace(e.TurnID) == "":
		return errors.New("turn id is required")
	case strings.TrimSpace(e.Type) == "":
		return errors.New("event type is required")
	case e.SchemaVersion <= 0:
		return errors.New("event schema version must be positive")
	case e.Seq < -1:
		return errors.New("event sequence cannot be less than -1")
	case e.OccurredAt.IsZero():
		return errors.New("event occurred time is required")
	default:
		return nil
	}
}

type EventStore interface {
	Append(context.Context, EventEnvelope) (EventEnvelope, bool, error)
	Replay(context.Context, string, int64, int) ([]EventEnvelope, error)
	SequenceBounds(context.Context, string) (int64, int64, bool, error)
}
