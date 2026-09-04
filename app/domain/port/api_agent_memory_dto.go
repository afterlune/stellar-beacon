package port

import (
	"time"
)

// AgentMemoryAssertionDTO is only returned from the authenticated admin
// review surface. It contains the source and lifecycle metadata needed to
// review a fact, but no provider or session internals.
type AgentMemoryAssertionDTO struct {
	ID         string     `json:"id"`
	SubjectKey string     `json:"subjectKey"`
	Predicate  string     `json:"predicate"`
	Object     string     `json:"object"`
	SourceType string     `json:"sourceType"`
	SourceID   string     `json:"sourceId"`
	Version    int        `json:"version"`
	Confidence float64    `json:"confidence"`
	Status     string     `json:"status"`
	ValidFrom  time.Time  `json:"validFrom"`
	ValidUntil *time.Time `json:"validUntil,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type AgentMemoryAssertionRevisionDTO struct {
	AssertionID string     `json:"assertionId"`
	RevisionNo  int        `json:"revisionNo"`
	SubjectKey  string     `json:"subjectKey"`
	Predicate   string     `json:"predicate"`
	Object      string     `json:"object"`
	SourceType  string     `json:"sourceType"`
	SourceID    string     `json:"sourceId"`
	Version     int        `json:"version"`
	Confidence  float64    `json:"confidence"`
	Status      string     `json:"status"`
	ValidFrom   time.Time  `json:"validFrom"`
	ValidUntil  *time.Time `json:"validUntil,omitempty"`
	Reason      string     `json:"reason"`
	ChangedAt   time.Time  `json:"changedAt"`
}

type AgentMemoryConflictMemberDTO struct {
	AssertionID string    `json:"assertionId"`
	Role        string    `json:"role"`
	AddedAt     time.Time `json:"addedAt"`
}

type AgentMemoryConflictDTO struct {
	ID                string                         `json:"id"`
	SubjectKey        string                         `json:"subjectKey"`
	Predicate         string                         `json:"predicate"`
	Status            string                         `json:"status"`
	WinnerAssertionID string                         `json:"winnerAssertionId,omitempty"`
	Resolution        string                         `json:"resolution,omitempty"`
	CreatedAt         time.Time                      `json:"createdAt"`
	ResolvedAt        *time.Time                     `json:"resolvedAt,omitempty"`
	UpdatedAt         time.Time                      `json:"updatedAt"`
	Members           []AgentMemoryConflictMemberDTO `json:"members"`
}

// AgentMemoryAssertionPageDTO deliberately reports hasMore instead of a
// misleading total count: the persistence port currently provides bounded
// page reads, not a separate COUNT query.
type AgentMemoryAssertionPageDTO struct {
	Items   []AgentMemoryAssertionDTO `json:"items"`
	Current int                       `json:"current"`
	Size    int                       `json:"size"`
	HasMore bool                      `json:"hasMore"`
}

type AgentMemoryHistoryDTO struct {
	AssertionID string                            `json:"assertionId"`
	Items       []AgentMemoryAssertionRevisionDTO `json:"items"`
}

type AgentMemoryConflictPageDTO struct {
	Items   []AgentMemoryConflictDTO `json:"items"`
	Current int                      `json:"current"`
	Size    int                      `json:"size"`
	HasMore bool                     `json:"hasMore"`
}

func NewAgentMemoryAssertionDTO(assertion AgentMemoryAssertion) AgentMemoryAssertionDTO {
	return AgentMemoryAssertionDTO{
		ID:         assertion.ID,
		SubjectKey: assertion.SubjectKey,
		Predicate:  assertion.Predicate,
		Object:     assertion.Object,
		SourceType: assertion.SourceType,
		SourceID:   assertion.SourceID,
		Version:    assertion.Version,
		Confidence: assertion.Confidence,
		Status:     string(assertion.Status),
		ValidFrom:  assertion.ValidFrom.UTC(),
		ValidUntil: agentMemoryOptionalTime(assertion.ValidUntil),
		CreatedAt:  assertion.CreatedAt.UTC(),
		UpdatedAt:  assertion.UpdatedAt.UTC(),
	}
}

func NewAgentMemoryAssertionRevisionDTO(revision AgentMemoryAssertionRevision) AgentMemoryAssertionRevisionDTO {
	return AgentMemoryAssertionRevisionDTO{
		AssertionID: revision.AssertionID,
		RevisionNo:  revision.RevisionNo,
		SubjectKey:  revision.SubjectKey,
		Predicate:   revision.Predicate,
		Object:      revision.Object,
		SourceType:  revision.SourceType,
		SourceID:    revision.SourceID,
		Version:     revision.Version,
		Confidence:  revision.Confidence,
		Status:      string(revision.Status),
		ValidFrom:   revision.ValidFrom.UTC(),
		ValidUntil:  agentMemoryOptionalTime(revision.ValidUntil),
		Reason:      revision.Reason,
		ChangedAt:   revision.ChangedAt.UTC(),
	}
}

func NewAgentMemoryConflictDTO(conflict AgentMemoryConflict) AgentMemoryConflictDTO {
	members := make([]AgentMemoryConflictMemberDTO, 0, len(conflict.Members))
	for _, member := range conflict.Members {
		members = append(members, AgentMemoryConflictMemberDTO{
			AssertionID: member.AssertionID,
			Role:        string(member.Role),
			AddedAt:     member.AddedAt.UTC(),
		})
	}
	return AgentMemoryConflictDTO{
		ID:                conflict.ID,
		SubjectKey:        conflict.SubjectKey,
		Predicate:         conflict.Predicate,
		Status:            string(conflict.Status),
		WinnerAssertionID: conflict.WinnerAssertionID,
		Resolution:        conflict.Resolution,
		CreatedAt:         conflict.CreatedAt.UTC(),
		ResolvedAt:        agentMemoryOptionalTime(conflict.ResolvedAt),
		UpdatedAt:         conflict.UpdatedAt.UTC(),
		Members:           members,
	}
}

func agentMemoryOptionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	value = value.UTC()
	return &value
}
