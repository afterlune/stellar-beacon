package repository

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.AgentMemoryAssertionRepository = (*MyAgentMemoryRepository)(nil)
var _ port.AgentMemoryHistoryRepository = (*MyAgentMemoryRepository)(nil)
var _ port.AgentMemoryConflictRepository = (*MyAgentMemoryRepository)(nil)

type MyAgentMemoryRepository struct {
	engine *xorm.Engine
}

func NewAgentMemoryRepository(engine *xorm.Engine) *MyAgentMemoryRepository {
	return &MyAgentMemoryRepository{engine: engine}
}

type agentMemoryRow struct {
	ID         string     `xorm:"id"`
	SubjectKey string     `xorm:"subject_key"`
	Predicate  string     `xorm:"predicate"`
	Object     string     `xorm:"object"`
	SourceType string     `xorm:"source_type"`
	SourceID   string     `xorm:"source_id"`
	Version    int        `xorm:"version"`
	Confidence float64    `xorm:"confidence"`
	Status     string     `xorm:"status"`
	ValidFrom  time.Time  `xorm:"valid_from"`
	ValidUntil *time.Time `xorm:"valid_until"`
	CreatedAt  time.Time  `xorm:"created_at"`
	UpdatedAt  time.Time  `xorm:"updated_at"`
}

type agentMemoryRevisionRow struct {
	AssertionID string     `xorm:"assertion_id"`
	RevisionNo  int        `xorm:"revision_no"`
	SubjectKey  string     `xorm:"subject_key"`
	Predicate   string     `xorm:"predicate"`
	Object      string     `xorm:"object"`
	SourceType  string     `xorm:"source_type"`
	SourceID    string     `xorm:"source_id"`
	Version     int        `xorm:"version"`
	Confidence  float64    `xorm:"confidence"`
	Status      string     `xorm:"status"`
	ValidFrom   time.Time  `xorm:"valid_from"`
	ValidUntil  *time.Time `xorm:"valid_until"`
	Reason      string     `xorm:"reason"`
	ChangedAt   time.Time  `xorm:"changed_at"`
}

type agentMemoryConflictRow struct {
	ID                string     `xorm:"id"`
	SubjectKey        string     `xorm:"subject_key"`
	Predicate         string     `xorm:"predicate"`
	Status            string     `xorm:"status"`
	WinnerAssertionID string     `xorm:"winner_assertion_id"`
	Resolution        string     `xorm:"resolution"`
	CreatedAt         time.Time  `xorm:"created_at"`
	ResolvedAt        *time.Time `xorm:"resolved_at"`
	UpdatedAt         time.Time  `xorm:"updated_at"`
}

type agentMemoryConflictMemberRow struct {
	ConflictID  string    `xorm:"conflict_id"`
	AssertionID string    `xorm:"assertion_id"`
	Role        string    `xorm:"role"`
	AddedAt     time.Time `xorm:"added_at"`
}

const agentMemoryColumns = `id, subject_key, predicate, object, source_type, source_id,
version, confidence, status, valid_from, valid_until, created_at, updated_at`

const agentMemoryRevisionColumns = `assertion_id, revision_no, subject_key, predicate, object,
source_type, source_id, version, confidence, status, valid_from, valid_until, reason, changed_at`

func (r agentMemoryRow) assertion() port.AgentMemoryAssertion {
	result := port.AgentMemoryAssertion{
		ID:         r.ID,
		SubjectKey: r.SubjectKey,
		Predicate:  r.Predicate,
		Object:     r.Object,
		SourceType: r.SourceType,
		SourceID:   r.SourceID,
		Version:    r.Version,
		Confidence: r.Confidence,
		Status:     port.AgentMemoryAssertionStatus(r.Status),
		ValidFrom:  r.ValidFrom.UTC(),
		CreatedAt:  r.CreatedAt.UTC(),
		UpdatedAt:  r.UpdatedAt.UTC(),
	}
	if r.ValidUntil != nil {
		result.ValidUntil = r.ValidUntil.UTC()
	}
	return result
}

func (r agentMemoryRevisionRow) revision() port.AgentMemoryAssertionRevision {
	result := port.AgentMemoryAssertionRevision{
		AssertionID: r.AssertionID,
		RevisionNo:  r.RevisionNo,
		SubjectKey:  r.SubjectKey,
		Predicate:   r.Predicate,
		Object:      r.Object,
		SourceType:  r.SourceType,
		SourceID:    r.SourceID,
		Version:     r.Version,
		Confidence:  r.Confidence,
		Status:      port.AgentMemoryAssertionStatus(r.Status),
		ValidFrom:   r.ValidFrom.UTC(),
		Reason:      r.Reason,
		ChangedAt:   r.ChangedAt.UTC(),
	}
	if r.ValidUntil != nil {
		result.ValidUntil = r.ValidUntil.UTC()
	}
	return result
}

func (r agentMemoryConflictRow) conflict() port.AgentMemoryConflict {
	result := port.AgentMemoryConflict{
		ID:                r.ID,
		SubjectKey:        r.SubjectKey,
		Predicate:         r.Predicate,
		Status:            port.AgentMemoryConflictStatus(r.Status),
		WinnerAssertionID: r.WinnerAssertionID,
		Resolution:        r.Resolution,
		CreatedAt:         r.CreatedAt.UTC(),
		UpdatedAt:         r.UpdatedAt.UTC(),
	}
	if r.ResolvedAt != nil {
		result.ResolvedAt = r.ResolvedAt.UTC()
	}
	return result
}

func (r *MyAgentMemoryRepository) Upsert(ctx context.Context, input port.AgentMemoryAssertion) error {
	assertion, err := normalizeAgentMemoryAssertion(input)
	if err != nil {
		return apperrors.Invalid("agent_memory.upsert", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_memory.upsert.database", nil)
	}
	return repoTx(r.engine, ctx, "agent_memory.upsert", func(session *xorm.Session) error {
		var existing agentMemoryRow
		found, err := session.SQL(`SELECT `+agentMemoryColumns+`
FROM t_agent_memory_assertion
WHERE subject_key = ? AND predicate = ? AND source_type = ? AND source_id = ? AND version = ?
FOR UPDATE`, assertion.SubjectKey, assertion.Predicate, assertion.SourceType, assertion.SourceID, assertion.Version).Get(&existing)
		if err != nil {
			return err
		}
		if found && existing.Status != string(port.AgentMemoryActive) {
			if memoryAssertionMatches(existing, assertion) {
				return nil
			}
			return apperrors.Conflict("agent_memory.upsert", "only active memory assertions can be revised")
		}
		if found {
			assertion.ID = existing.ID
			if memoryAssertionMatches(existing, assertion) {
				return nil
			}

			var conflicting []agentMemoryRow
			openConflictID := ""
			if assertion.Status == port.AgentMemoryActive || assertion.Status == port.AgentMemoryConflicted {
				if err := session.SQL(`SELECT `+agentMemoryColumns+`
FROM t_agent_memory_assertion
WHERE subject_key = ? AND predicate = ? AND status = ? AND object <> ? AND id <> ?
FOR UPDATE`, assertion.SubjectKey, assertion.Predicate, string(port.AgentMemoryActive), assertion.Object, assertion.ID).Find(&conflicting); err != nil {
					return err
				}
				openConflictID, err = findOpenMemoryConflict(session, assertion.SubjectKey, assertion.Predicate)
				if err != nil {
					return err
				}
				if assertion.Status == port.AgentMemoryActive && (len(conflicting) > 0 || openConflictID != "") {
					assertion.Status = port.AgentMemoryConflicted
				}
				if assertion.Status == port.AgentMemoryConflicted && len(conflicting) == 0 && openConflictID == "" {
					return apperrors.Invalid("agent_memory.upsert", "conflicted assertion requires another conflict member")
				}
			}
			if err := appendMemoryRevision(session, existing, "revised_previous", assertion.UpdatedAt); err != nil {
				return err
			}
			if _, err := session.Exec(`
UPDATE t_agent_memory_assertion
SET object = ?, confidence = ?, status = ?, valid_from = ?, valid_until = ?, updated_at = ?
WHERE id = ?`, assertion.Object, assertion.Confidence, string(assertion.Status), assertion.ValidFrom,
				nullableMemoryTime(assertion.ValidUntil), assertion.UpdatedAt, assertion.ID); err != nil {
				return err
			}
			updated := memoryRowFromAssertion(assertion)
			if err := appendMemoryRevision(session, updated, "revised", assertion.UpdatedAt); err != nil {
				return err
			}
			if assertion.Status != port.AgentMemoryConflicted {
				return nil
			}
			for index := range conflicting {
				previous := &conflicting[index]
				previous.Status = string(port.AgentMemoryConflicted)
				previous.UpdatedAt = assertion.UpdatedAt
				if _, err := session.Exec("UPDATE t_agent_memory_assertion SET status = ?, updated_at = ? WHERE id = ?", string(port.AgentMemoryConflicted), assertion.UpdatedAt, previous.ID); err != nil {
					return err
				}
				if err := appendMemoryRevision(session, *previous, "conflict_detected", assertion.UpdatedAt); err != nil {
					return err
				}
			}
			if openConflictID == "" {
				openConflictID, err = createOpenMemoryConflict(session, assertion.SubjectKey, assertion.Predicate, assertion.UpdatedAt)
				if err != nil {
					return err
				}
			}
			if _, err := session.Exec("INSERT INTO t_agent_memory_conflict_member (conflict_id, assertion_id, role, added_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", openConflictID, assertion.ID, string(port.AgentMemoryConflictCandidate), assertion.UpdatedAt); err != nil {
				return err
			}
			for _, previous := range conflicting {
				if _, err := session.Exec("INSERT INTO t_agent_memory_conflict_member (conflict_id, assertion_id, role, added_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", openConflictID, previous.ID, string(port.AgentMemoryConflictCandidate), assertion.UpdatedAt); err != nil {
					return err
				}
			}
			_, err = session.Exec("UPDATE t_agent_memory_conflict SET updated_at = ? WHERE id = ?", assertion.UpdatedAt, openConflictID)
			return err
		}

		var conflicting []agentMemoryRow
		openConflictID := ""
		if assertion.Status == port.AgentMemoryActive || assertion.Status == port.AgentMemoryConflicted {
			if err := session.SQL(`SELECT `+agentMemoryColumns+`
FROM t_agent_memory_assertion
WHERE subject_key = ? AND predicate = ? AND status = ? AND object <> ?
FOR UPDATE`, assertion.SubjectKey, assertion.Predicate, string(port.AgentMemoryActive), assertion.Object).Find(&conflicting); err != nil {
				return err
			}
			openConflictID, err = findOpenMemoryConflict(session, assertion.SubjectKey, assertion.Predicate)
			if err != nil {
				return err
			}
			if assertion.Status == port.AgentMemoryActive && (len(conflicting) > 0 || openConflictID != "") {
				assertion.Status = port.AgentMemoryConflicted
			}
			if assertion.Status == port.AgentMemoryConflicted && len(conflicting) == 0 && openConflictID == "" {
				return apperrors.Invalid("agent_memory.upsert", "conflicted assertion requires another conflict member")
			}
		}

		if _, err = session.Exec(`
INSERT INTO t_agent_memory_assertion
    (id, subject_key, predicate, object, source_type, source_id, version, confidence, status, valid_from, valid_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			assertion.ID, assertion.SubjectKey, assertion.Predicate, assertion.Object, assertion.SourceType, assertion.SourceID,
			assertion.Version, assertion.Confidence, string(assertion.Status), assertion.ValidFrom, nullableMemoryTime(assertion.ValidUntil),
			assertion.CreatedAt, assertion.UpdatedAt); err != nil {
			return err
		}
		newRow := memoryRowFromAssertion(assertion)
		if err := appendMemoryRevision(session, newRow, memoryCreationReason(assertion.Status), assertion.UpdatedAt); err != nil {
			return err
		}
		if assertion.Status != port.AgentMemoryConflicted {
			return nil
		}

		for index := range conflicting {
			previous := &conflicting[index]
			previous.Status = string(port.AgentMemoryConflicted)
			previous.UpdatedAt = assertion.UpdatedAt
			if _, err := session.Exec("UPDATE t_agent_memory_assertion SET status = ?, updated_at = ? WHERE id = ?", string(port.AgentMemoryConflicted), assertion.UpdatedAt, previous.ID); err != nil {
				return err
			}
			if err := appendMemoryRevision(session, *previous, "conflict_detected", assertion.UpdatedAt); err != nil {
				return err
			}
		}
		if openConflictID == "" {
			openConflictID, err = createOpenMemoryConflict(session, assertion.SubjectKey, assertion.Predicate, assertion.UpdatedAt)
			if err != nil {
				return err
			}
		}
		for _, previous := range conflicting {
			if _, err := session.Exec("INSERT INTO t_agent_memory_conflict_member (conflict_id, assertion_id, role, added_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", openConflictID, previous.ID, string(port.AgentMemoryConflictCandidate), assertion.UpdatedAt); err != nil {
				return err
			}
		}
		if _, err := session.Exec("INSERT INTO t_agent_memory_conflict_member (conflict_id, assertion_id, role, added_at) VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING", openConflictID, assertion.ID, string(port.AgentMemoryConflictCandidate), assertion.UpdatedAt); err != nil {
			return err
		}
		_, err = session.Exec("UPDATE t_agent_memory_conflict SET updated_at = ? WHERE id = ?", assertion.UpdatedAt, openConflictID)
		return err
	})
}

func (r *MyAgentMemoryRepository) History(ctx context.Context, assertionID string) ([]port.AgentMemoryAssertionRevision, error) {
	assertionID, err := normalizeMemoryID(assertionID, "memory assertion id")
	if err != nil {
		return nil, apperrors.Invalid("agent_memory.history", err.Error())
	}
	if r == nil {
		return nil, apperrors.Unavailable("agent_memory.history.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_memory.history")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []agentMemoryRevisionRow
	if err := session.SQL("SELECT "+agentMemoryRevisionColumns+" FROM t_agent_memory_assertion_revision WHERE assertion_id = ? ORDER BY revision_no ASC", assertionID).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_memory.history", err)
	}
	result := make([]port.AgentMemoryAssertionRevision, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.revision())
	}
	return result, nil
}

func (r *MyAgentMemoryRepository) ListConflicts(ctx context.Context, input port.AgentMemoryConflictFilter) ([]port.AgentMemoryConflict, error) {
	filter, err := normalizeAgentMemoryConflictFilter(input)
	if err != nil {
		return nil, apperrors.Invalid("agent_memory.conflicts", err.Error())
	}
	if r == nil {
		return nil, apperrors.Unavailable("agent_memory.conflicts.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_memory.conflicts")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	where := "status = ?"
	args := []any{string(filter.Status)}
	if filter.SubjectKey != "" {
		where += " AND subject_key = ?"
		args = append(args, filter.SubjectKey)
	}
	if filter.Predicate != "" {
		where += " AND predicate = ?"
		args = append(args, filter.Predicate)
	}
	args = append(args, filter.Size, (filter.Current-1)*filter.Size)
	var rows []agentMemoryConflictRow
	if err := session.SQL("SELECT id, subject_key, predicate, status, winner_assertion_id, resolution, created_at, resolved_at, updated_at FROM t_agent_memory_conflict WHERE "+where+" ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?", args...).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_memory.conflicts", err)
	}
	result := make([]port.AgentMemoryConflict, 0, len(rows))
	for _, row := range rows {
		var members []agentMemoryConflictMemberRow
		if err := session.SQL("SELECT conflict_id, assertion_id, role, added_at FROM t_agent_memory_conflict_member WHERE conflict_id = ? ORDER BY added_at ASC, assertion_id ASC", row.ID).Find(&members); err != nil {
			return nil, apperrors.Unavailable("agent_memory.conflicts.members", err)
		}
		conflict := row.conflict()
		conflict.Members = make([]port.AgentMemoryConflictMember, 0, len(members))
		for _, member := range members {
			conflict.Members = append(conflict.Members, port.AgentMemoryConflictMember{
				AssertionID: member.AssertionID,
				Role:        port.AgentMemoryConflictMemberRole(member.Role),
				AddedAt:     member.AddedAt.UTC(),
			})
		}
		result = append(result, conflict)
	}
	return result, nil
}

func (r *MyAgentMemoryRepository) ResolveConflict(ctx context.Context, conflictID, winnerAssertionID, resolution string) error {
	return r.finalizeConflict(ctx, conflictID, winnerAssertionID, resolution, port.AgentMemoryConflictResolved)
}

func (r *MyAgentMemoryRepository) RejectConflict(ctx context.Context, conflictID, resolution string) error {
	return r.finalizeConflict(ctx, conflictID, "", resolution, port.AgentMemoryConflictRejected)
}

func (r *MyAgentMemoryRepository) finalizeConflict(ctx context.Context, conflictID, winnerAssertionID, resolution string, status port.AgentMemoryConflictStatus) error {
	var err error
	conflictID, err = normalizeMemoryID(conflictID, "memory conflict id")
	if err != nil {
		return apperrors.Invalid("agent_memory.conflict", err.Error())
	}
	winnerAssertionID = strings.TrimSpace(winnerAssertionID)
	if status == port.AgentMemoryConflictResolved {
		if _, err := normalizeMemoryID(winnerAssertionID, "winning assertion id"); err != nil {
			return apperrors.Invalid("agent_memory.conflict", err.Error())
		}
	} else {
		winnerAssertionID = ""
	}
	resolution = strings.TrimSpace(resolution)
	if resolution == "" {
		if status == port.AgentMemoryConflictResolved {
			resolution = "winner_selected"
		} else {
			resolution = "all_candidates_rejected"
		}
	}
	if len(resolution) > 256 || strings.ContainsAny(resolution, "\r\n\x00") {
		return apperrors.Invalid("agent_memory.conflict", "conflict resolution is invalid")
	}
	if r == nil {
		return apperrors.Unavailable("agent_memory.conflict.database", nil)
	}
	now := time.Now().UTC()
	return repoTx(r.engine, ctx, "agent_memory.conflict", func(session *xorm.Session) error {
		var conflict agentMemoryConflictRow
		found, err := session.SQL("SELECT id, subject_key, predicate, status, winner_assertion_id, resolution, created_at, resolved_at, updated_at FROM t_agent_memory_conflict WHERE id = ? FOR UPDATE", conflictID).Get(&conflict)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("agent_memory.conflict")
		}
		if conflict.Status != string(port.AgentMemoryConflictOpen) {
			return apperrors.Conflict("agent_memory.conflict", "memory conflict has already been resolved")
		}

		var members []agentMemoryConflictMemberRow
		if err := session.SQL("SELECT conflict_id, assertion_id, role, added_at FROM t_agent_memory_conflict_member WHERE conflict_id = ? ORDER BY assertion_id ASC FOR UPDATE", conflictID).Find(&members); err != nil {
			return err
		}
		if len(members) == 0 {
			return apperrors.Conflict("agent_memory.conflict", "memory conflict has no members")
		}
		memberIDs := make(map[string]struct{}, len(members))
		for _, member := range members {
			memberIDs[member.AssertionID] = struct{}{}
		}
		if status == port.AgentMemoryConflictResolved {
			if _, ok := memberIDs[winnerAssertionID]; !ok {
				return apperrors.Invalid("agent_memory.conflict", "winning assertion is not a conflict member")
			}
		}

		var assertions []agentMemoryRow
		if err := session.SQL("SELECT a.id, a.subject_key, a.predicate, a.object, a.source_type, a.source_id, a.version, a.confidence, a.status, a.valid_from, a.valid_until, a.created_at, a.updated_at FROM t_agent_memory_assertion a JOIN t_agent_memory_conflict_member m ON m.assertion_id = a.id WHERE m.conflict_id = ? ORDER BY a.id ASC FOR UPDATE", conflictID).Find(&assertions); err != nil {
			return err
		}
		if len(assertions) != len(members) {
			return apperrors.Conflict("agent_memory.conflict", "a conflict member no longer exists")
		}
		for _, assertion := range assertions {
			if assertion.Status != string(port.AgentMemoryConflicted) {
				return apperrors.Conflict("agent_memory.conflict", "a conflict member changed outside the review flow")
			}
		}
		for index := range assertions {
			assertion := &assertions[index]
			nextStatus := port.AgentMemoryRetracted
			if status == port.AgentMemoryConflictResolved {
				nextStatus = port.AgentMemoryStale
				if assertion.ID == winnerAssertionID {
					nextStatus = port.AgentMemoryActive
				}
			}
			assertion.Status = string(nextStatus)
			assertion.UpdatedAt = now
			if _, err := session.Exec("UPDATE t_agent_memory_assertion SET status = ?, updated_at = ? WHERE id = ?", string(nextStatus), now, assertion.ID); err != nil {
				return err
			}
			if err := appendMemoryRevision(session, *assertion, conflictResolutionReason(status), now); err != nil {
				return err
			}
		}

		if status == port.AgentMemoryConflictResolved {
			if _, err := session.Exec("UPDATE t_agent_memory_conflict_member SET role = CASE WHEN assertion_id = ? THEN ? ELSE ? END WHERE conflict_id = ?", winnerAssertionID, string(port.AgentMemoryConflictWinner), string(port.AgentMemoryConflictRejectedMember), conflictID); err != nil {
				return err
			}
		} else if _, err := session.Exec("UPDATE t_agent_memory_conflict_member SET role = ? WHERE conflict_id = ?", string(port.AgentMemoryConflictRejectedMember), conflictID); err != nil {
			return err
		}
		_, err = session.Exec("UPDATE t_agent_memory_conflict SET status = ?, winner_assertion_id = ?, resolution = ?, resolved_at = ?, updated_at = ? WHERE id = ?", string(status), nullableMemoryID(winnerAssertionID), resolution, now, now, conflictID)
		return err
	})
}

func (r *MyAgentMemoryRepository) List(ctx context.Context, filter port.AgentMemoryFilter) ([]port.AgentMemoryAssertion, error) {
	filter, err := normalizeAgentMemoryFilter(filter)
	if err != nil {
		return nil, apperrors.Invalid("agent_memory.list", err.Error())
	}
	if r == nil {
		return nil, apperrors.Unavailable("agent_memory.list.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "agent_memory.list")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	where := "status = ?"
	args := []any{string(filter.Status)}
	if filter.SubjectKey != "" {
		where += " AND subject_key = ?"
		args = append(args, filter.SubjectKey)
	}
	if filter.Predicate != "" {
		where += " AND predicate = ?"
		args = append(args, filter.Predicate)
	}
	args = append(args, filter.Size, (filter.Current-1)*filter.Size)
	var rows []agentMemoryRow
	if err := session.SQL("SELECT "+agentMemoryColumns+" FROM t_agent_memory_assertion WHERE "+where+" ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?", args...).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("agent_memory.list", err)
	}
	result := make([]port.AgentMemoryAssertion, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.assertion())
	}
	return result, nil
}

func (r *MyAgentMemoryRepository) MarkStale(ctx context.Context, now time.Time) (int, error) {
	if r == nil {
		return 0, apperrors.Unavailable("agent_memory.mark_stale.database", nil)
	}
	now = normalizeTime(now)
	count := 0
	err := repoTx(r.engine, ctx, "agent_memory.mark_stale", func(session *xorm.Session) error {
		var rows []agentMemoryRow
		if err := session.SQL(`SELECT `+agentMemoryColumns+`
FROM t_agent_memory_assertion
WHERE status = ? AND valid_until IS NOT NULL AND valid_until <= ?
ORDER BY id ASC
FOR UPDATE`, string(port.AgentMemoryActive), now).Find(&rows); err != nil {
			return err
		}
		for index := range rows {
			row := &rows[index]
			row.Status = string(port.AgentMemoryStale)
			row.UpdatedAt = now
			if _, err := session.Exec("UPDATE t_agent_memory_assertion SET status = ?, updated_at = ? WHERE id = ?", string(port.AgentMemoryStale), now, row.ID); err != nil {
				return err
			}
			if err := appendMemoryRevision(session, *row, "expired", now); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

func (r *MyAgentMemoryRepository) SetStatus(ctx context.Context, id string, status port.AgentMemoryAssertionStatus) error {
	var err error
	id, err = normalizeMemoryID(id, "memory assertion id")
	if err != nil {
		return apperrors.Invalid("agent_memory.status", err.Error())
	}
	normalizedStatus, err := port.NormalizeAgentMemoryStatus(string(status))
	if err != nil {
		return apperrors.Invalid("agent_memory.status", err.Error())
	}
	if r == nil {
		return apperrors.Unavailable("agent_memory.status.database", nil)
	}
	now := time.Now().UTC()
	return repoTx(r.engine, ctx, "agent_memory.status", func(session *xorm.Session) error {
		var existing agentMemoryRow
		found, err := session.SQL("SELECT "+agentMemoryColumns+" FROM t_agent_memory_assertion WHERE id = ? FOR UPDATE", id).Get(&existing)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("agent_memory.status")
		}
		if existing.Status == string(port.AgentMemoryRetracted) && normalizedStatus != port.AgentMemoryRetracted {
			return apperrors.Conflict("agent_memory.status", "retracted memory cannot be reactivated")
		}
		if existing.Status == string(normalizedStatus) {
			return nil
		}
		existing.Status = string(normalizedStatus)
		existing.UpdatedAt = now
		if _, err := session.Exec("UPDATE t_agent_memory_assertion SET status = ?, updated_at = ? WHERE id = ?", string(normalizedStatus), now, id); err != nil {
			return err
		}
		return appendMemoryRevision(session, existing, "status_changed", now)
	})
}

func normalizeAgentMemoryAssertion(input port.AgentMemoryAssertion) (port.AgentMemoryAssertion, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	var err error
	input.SubjectKey, err = port.NormalizeAgentMemorySubject(input.SubjectKey)
	if err != nil {
		return port.AgentMemoryAssertion{}, err
	}
	input.Predicate = strings.TrimSpace(input.Predicate)
	input.Object = strings.TrimSpace(input.Object)
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.SourceID = strings.TrimSpace(input.SourceID)
	if len(input.ID) > 128 || input.Predicate == "" || len(input.Predicate) > 128 || input.Object == "" || len([]rune(input.Object)) > 100_000 || input.SourceType == "" || len(input.SourceType) > 64 || input.SourceID == "" || len(input.SourceID) > 128 || strings.ContainsAny(input.Predicate+input.SourceType+input.SourceID, "\r\n\x00") {
		return port.AgentMemoryAssertion{}, stderrors.New("memory assertion identity or content is invalid")
	}
	if input.Version <= 0 {
		return port.AgentMemoryAssertion{}, stderrors.New("memory assertion version must be positive")
	}
	if err := port.ValidateAgentMemoryConfidence(input.Confidence); err != nil {
		return port.AgentMemoryAssertion{}, err
	}
	if input.Status == "" {
		input.Status = port.AgentMemoryActive
	}
	normalizedStatus, err := port.NormalizeAgentMemoryStatus(string(input.Status))
	if err != nil {
		return port.AgentMemoryAssertion{}, err
	}
	input.Status = normalizedStatus
	if input.ValidFrom.IsZero() {
		input.ValidFrom = time.Now().UTC()
	}
	input.ValidFrom = input.ValidFrom.UTC()
	if !input.ValidUntil.IsZero() {
		input.ValidUntil = input.ValidUntil.UTC()
		if !input.ValidUntil.After(input.ValidFrom) {
			return port.AgentMemoryAssertion{}, stderrors.New("memory assertion validity range is invalid")
		}
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = input.ValidFrom
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	return input, nil
}

func normalizeAgentMemoryFilter(filter port.AgentMemoryFilter) (port.AgentMemoryFilter, error) {
	filter.SubjectKey = strings.TrimSpace(filter.SubjectKey)
	if filter.SubjectKey != "" {
		var err error
		filter.SubjectKey, err = port.NormalizeAgentMemorySubject(filter.SubjectKey)
		if err != nil {
			return port.AgentMemoryFilter{}, err
		}
	}
	filter.Predicate = strings.TrimSpace(filter.Predicate)
	if len(filter.Predicate) > 128 || strings.ContainsAny(filter.Predicate, "\r\n\x00") {
		return port.AgentMemoryFilter{}, stderrors.New("memory predicate is invalid")
	}
	if filter.Status == "" {
		filter.Status = port.AgentMemoryActive
	}
	normalizedStatus, err := port.NormalizeAgentMemoryStatus(string(filter.Status))
	if err != nil {
		return port.AgentMemoryFilter{}, err
	}
	filter.Status = normalizedStatus
	if filter.Current <= 0 {
		filter.Current = 1
	}
	if filter.Size <= 0 {
		filter.Size = 50
	}
	if filter.Size > 100 {
		filter.Size = 100
	}
	return filter, nil
}

func normalizeAgentMemoryConflictFilter(filter port.AgentMemoryConflictFilter) (port.AgentMemoryConflictFilter, error) {
	filter.SubjectKey = strings.TrimSpace(filter.SubjectKey)
	if filter.SubjectKey != "" {
		var err error
		filter.SubjectKey, err = port.NormalizeAgentMemorySubject(filter.SubjectKey)
		if err != nil {
			return port.AgentMemoryConflictFilter{}, err
		}
	}
	filter.Predicate = strings.TrimSpace(filter.Predicate)
	if len(filter.Predicate) > 128 || strings.ContainsAny(filter.Predicate, "\r\n\x00") {
		return port.AgentMemoryConflictFilter{}, stderrors.New("memory conflict predicate is invalid")
	}
	if filter.Status == "" {
		filter.Status = port.AgentMemoryConflictOpen
	}
	normalizedStatus, err := port.NormalizeAgentMemoryConflictStatus(string(filter.Status))
	if err != nil {
		return port.AgentMemoryConflictFilter{}, err
	}
	filter.Status = normalizedStatus
	if filter.Current <= 0 {
		filter.Current = 1
	}
	if filter.Size <= 0 {
		filter.Size = 50
	}
	if filter.Size > 100 {
		filter.Size = 100
	}
	return filter, nil
}

func normalizeMemoryID(value, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.ContainsAny(value, "\r\n\x00") {
		return "", stderrors.New(label + " is invalid")
	}
	return value, nil
}

func memoryRowFromAssertion(assertion port.AgentMemoryAssertion) agentMemoryRow {
	row := agentMemoryRow{
		ID:         assertion.ID,
		SubjectKey: assertion.SubjectKey,
		Predicate:  assertion.Predicate,
		Object:     assertion.Object,
		SourceType: assertion.SourceType,
		SourceID:   assertion.SourceID,
		Version:    assertion.Version,
		Confidence: assertion.Confidence,
		Status:     string(assertion.Status),
		ValidFrom:  assertion.ValidFrom,
		CreatedAt:  assertion.CreatedAt,
		UpdatedAt:  assertion.UpdatedAt,
	}
	if !assertion.ValidUntil.IsZero() {
		validUntil := assertion.ValidUntil
		row.ValidUntil = &validUntil
	}
	return row
}

func memoryAssertionMatches(row agentMemoryRow, assertion port.AgentMemoryAssertion) bool {
	return row.SubjectKey == assertion.SubjectKey &&
		row.Predicate == assertion.Predicate &&
		row.Object == assertion.Object &&
		row.SourceType == assertion.SourceType &&
		row.SourceID == assertion.SourceID &&
		row.Version == assertion.Version &&
		row.Confidence == assertion.Confidence &&
		row.Status == string(assertion.Status) &&
		row.ValidFrom.Equal(assertion.ValidFrom) &&
		memoryTimesEqual(row.ValidUntil, assertion.ValidUntil) &&
		row.CreatedAt.Equal(assertion.CreatedAt)
}

func memoryTimesEqual(value *time.Time, other time.Time) bool {
	if value == nil {
		return other.IsZero()
	}
	return value.Equal(other)
}

func appendMemoryRevision(session *xorm.Session, row agentMemoryRow, reason string, changedAt time.Time) error {
	if session == nil {
		return stderrors.New("memory revision session is nil")
	}
	if changedAt.IsZero() {
		changedAt = time.Now().UTC()
	}
	changedAt = changedAt.UTC()
	reason = strings.TrimSpace(reason)
	if len(reason) > 256 || strings.ContainsAny(reason, "\r\n\x00") {
		return stderrors.New("memory revision reason is invalid")
	}
	var latest struct {
		RevisionNo int `xorm:"revision_no"`
	}
	found, err := session.SQL("SELECT revision_no FROM t_agent_memory_assertion_revision WHERE assertion_id = ? ORDER BY revision_no DESC LIMIT 1 FOR UPDATE", row.ID).Get(&latest)
	if err != nil {
		return err
	}
	nextRevision := 1
	if found {
		nextRevision = latest.RevisionNo + 1
	}
	_, err = session.Exec(`INSERT INTO t_agent_memory_assertion_revision
    (assertion_id, revision_no, subject_key, predicate, object, source_type, source_id, version, confidence, status, valid_from, valid_until, reason, changed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.ID, nextRevision, row.SubjectKey, row.Predicate, row.Object, row.SourceType, row.SourceID,
		row.Version, row.Confidence, row.Status, row.ValidFrom, nullableMemoryTime(pointerTimeValue(row.ValidUntil)), reason, changedAt)
	return err
}

func pointerTimeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func findOpenMemoryConflict(session *xorm.Session, subjectKey, predicate string) (string, error) {
	var row struct {
		ID string `xorm:"id"`
	}
	found, err := session.SQL("SELECT id FROM t_agent_memory_conflict WHERE subject_key = ? AND predicate = ? AND status = ? FOR UPDATE", subjectKey, predicate, string(port.AgentMemoryConflictOpen)).Get(&row)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}
	return row.ID, nil
}

func createOpenMemoryConflict(session *xorm.Session, subjectKey, predicate string, now time.Time) (string, error) {
	conflictID := uuid.NewString()
	if _, err := session.Exec(`INSERT INTO t_agent_memory_conflict
    (id, subject_key, predicate, status, resolution, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO NOTHING`, conflictID, subjectKey, predicate, string(port.AgentMemoryConflictOpen), "", now, now); err != nil {
		return "", err
	}
	return findOpenMemoryConflict(session, subjectKey, predicate)
}

func memoryCreationReason(status port.AgentMemoryAssertionStatus) string {
	if status == port.AgentMemoryConflicted {
		return "conflict_detected"
	}
	return "created"
}

func conflictResolutionReason(status port.AgentMemoryConflictStatus) string {
	if status == port.AgentMemoryConflictResolved {
		return "conflict_resolved"
	}
	return "conflict_rejected"
}

func nullableMemoryTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
}

func nullableMemoryID(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
