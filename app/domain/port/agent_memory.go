package port

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

// AgentMemoryAssertionStatus is deliberately explicit: a contradiction is
// retained as data for review, rather than silently overwriting a prior fact.
type AgentMemoryAssertionStatus string

const (
	AgentMemoryActive     AgentMemoryAssertionStatus = "active"
	AgentMemoryStale      AgentMemoryAssertionStatus = "stale"
	AgentMemoryConflicted AgentMemoryAssertionStatus = "conflicted"
	AgentMemoryRetracted  AgentMemoryAssertionStatus = "retracted"
)

func NormalizeAgentMemoryStatus(value string) (AgentMemoryAssertionStatus, error) {
	status := AgentMemoryAssertionStatus(strings.ToLower(strings.TrimSpace(value)))
	switch status {
	case AgentMemoryActive, AgentMemoryStale, AgentMemoryConflicted, AgentMemoryRetracted:
		return status, nil
	default:
		return "", errors.New("agent memory status is invalid")
	}
}

// NormalizeAgentMemorySubject only permits a durable account identity. This
// keeps anonymous visitors, IPs and short-lived chat sessions out of the
// learning path even if a caller accidentally passes one as a subject.
func NormalizeAgentMemorySubject(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n\x00") || !strings.HasPrefix(value, "user:") {
		return "", errors.New("agent memory subject must be a durable user identity")
	}
	identity := strings.TrimPrefix(value, "user:")
	if identity == "" || identity == "guest" || identity == "anonymous" || strings.HasPrefix(identity, "ip:") || strings.HasPrefix(identity, "session:") {
		return "", errors.New("anonymous agent memory subject is not allowed")
	}
	return value, nil
}

type AgentMemoryAssertion struct {
	ID         string
	SubjectKey string
	Predicate  string
	Object     string
	SourceType string
	SourceID   string
	Version    int
	Confidence float64
	Status     AgentMemoryAssertionStatus
	ValidFrom  time.Time
	ValidUntil time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AgentMemoryFilter struct {
	SubjectKey string
	Predicate  string
	Status     AgentMemoryAssertionStatus
	Current    int
	Size       int
}

// AgentMemoryAssertionRevision is an immutable audit snapshot. Version is
// the source assertion version supplied by the caller; RevisionNo is the
// append-only change number maintained by the persistence adapter.
type AgentMemoryAssertionRevision struct {
	AssertionID string
	RevisionNo  int
	SubjectKey  string
	Predicate   string
	Object      string
	SourceType  string
	SourceID    string
	Version     int
	Confidence  float64
	Status      AgentMemoryAssertionStatus
	ValidFrom   time.Time
	ValidUntil  time.Time
	Reason      string
	ChangedAt   time.Time
}

type AgentMemoryConflictStatus string

const (
	AgentMemoryConflictOpen     AgentMemoryConflictStatus = "open"
	AgentMemoryConflictResolved AgentMemoryConflictStatus = "resolved"
	AgentMemoryConflictRejected AgentMemoryConflictStatus = "rejected"
)

func NormalizeAgentMemoryConflictStatus(value string) (AgentMemoryConflictStatus, error) {
	status := AgentMemoryConflictStatus(strings.ToLower(strings.TrimSpace(value)))
	switch status {
	case AgentMemoryConflictOpen, AgentMemoryConflictResolved, AgentMemoryConflictRejected:
		return status, nil
	default:
		return "", errors.New("agent memory conflict status is invalid")
	}
}

type AgentMemoryConflictMemberRole string

const (
	AgentMemoryConflictCandidate      AgentMemoryConflictMemberRole = "candidate"
	AgentMemoryConflictWinner         AgentMemoryConflictMemberRole = "winner"
	AgentMemoryConflictRejectedMember AgentMemoryConflictMemberRole = "rejected"
)

type AgentMemoryConflictMember struct {
	AssertionID string
	Role        AgentMemoryConflictMemberRole
	AddedAt     time.Time
}

type AgentMemoryConflict struct {
	ID                string
	SubjectKey        string
	Predicate         string
	Status            AgentMemoryConflictStatus
	WinnerAssertionID string
	Resolution        string
	CreatedAt         time.Time
	ResolvedAt        time.Time
	UpdatedAt         time.Time
	Members           []AgentMemoryConflictMember
}

type AgentMemoryConflictFilter struct {
	SubjectKey string
	Predicate  string
	Status     AgentMemoryConflictStatus
	Current    int
	Size       int
}

// AgentMemoryAssertionRepository is intentionally not part of anonymous chat
// dependencies. Callers must choose a durable subject and an auditable
// source; implementations must preserve versions and conflict states.
type AgentMemoryAssertionRepository interface {
	Upsert(context.Context, AgentMemoryAssertion) error
	List(context.Context, AgentMemoryFilter) ([]AgentMemoryAssertion, error)
	MarkStale(context.Context, time.Time) (int, error)
	SetStatus(context.Context, string, AgentMemoryAssertionStatus) error
}

// AgentMemoryHistoryRepository is an optional refinement so small fakes can
// continue to implement current-assertion reads without pretending to offer
// immutable history.
type AgentMemoryHistoryRepository interface {
	History(context.Context, string) ([]AgentMemoryAssertionRevision, error)
}

// AgentMemoryConflictRepository owns reviewable contradictions. A conflict
// never silently chooses a winner; resolution is an explicit user/operator
// action and must preserve the losing assertions as history.
type AgentMemoryConflictRepository interface {
	ListConflicts(context.Context, AgentMemoryConflictFilter) ([]AgentMemoryConflict, error)
	ResolveConflict(context.Context, string, string, string) error
	RejectConflict(context.Context, string, string) error
}

func ValidateAgentMemoryConfidence(value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return errors.New("agent memory confidence is invalid")
	}
	return nil
}
