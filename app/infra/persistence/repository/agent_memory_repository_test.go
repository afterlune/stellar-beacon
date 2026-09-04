package repository

import (
	"math"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestNormalizeAgentMemoryAssertionRequiresDurableSubject(t *testing.T) {
	assertion, err := normalizeAgentMemoryAssertion(port.AgentMemoryAssertion{
		SubjectKey: "USER:7",
		Predicate:  "favorite_topic",
		Object:     "分布式系统",
		SourceType: "manual",
		SourceID:   "note-1",
		Version:    1,
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if assertion.SubjectKey != "user:7" || assertion.Status != port.AgentMemoryActive || assertion.ValidFrom.IsZero() {
		t.Fatalf("normalized assertion = %+v", assertion)
	}

	for index, subject := range []string{"guest", "anonymous", "ip:127.0.0.1", "session:abc", "visitor:1"} {
		input := port.AgentMemoryAssertion{
			SubjectKey: subject,
			Predicate:  "p",
			Object:     "o",
			SourceType: "manual",
			SourceID:   "source",
			Version:    1,
			Confidence: 1,
		}
		if _, err := normalizeAgentMemoryAssertion(input); err == nil {
			t.Fatalf("case %d: anonymous subject %q was accepted", index, subject)
		}
	}
}

func TestNormalizeAgentMemoryAssertionPreservesValidityAndRejectsBadConfidence(t *testing.T) {
	from := time.Date(2026, 8, 29, 12, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	until := from.Add(time.Hour)
	assertion, err := normalizeAgentMemoryAssertion(port.AgentMemoryAssertion{
		SubjectKey: "user:7", Predicate: "p", Object: "o", SourceType: "article", SourceID: "42",
		Version: 2, Confidence: 0.5, ValidFrom: from, ValidUntil: until,
	})
	if err != nil || assertion.ValidFrom.Location() != time.UTC || assertion.ValidUntil.Location() != time.UTC {
		t.Fatalf("assertion=%+v error=%v", assertion, err)
	}
	for index, confidence := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		input := port.AgentMemoryAssertion{
			SubjectKey: "user:7", Predicate: "p", Object: "o", SourceType: "article", SourceID: "42",
			Version: 1, Confidence: confidence,
		}
		if _, err := normalizeAgentMemoryAssertion(input); err == nil {
			t.Fatalf("case %d: confidence %v was accepted", index, confidence)
		}
	}
}

func TestNormalizeAgentMemoryFilterDefaultsToActiveAndBoundsPage(t *testing.T) {
	filter, err := normalizeAgentMemoryFilter(port.AgentMemoryFilter{SubjectKey: "user:7", Size: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if filter.Status != port.AgentMemoryActive || filter.Current != 1 || filter.Size != 100 {
		t.Fatalf("normalized filter = %+v", filter)
	}
	if _, err := normalizeAgentMemoryFilter(port.AgentMemoryFilter{SubjectKey: "guest"}); err == nil {
		t.Fatal("anonymous memory filter was accepted")
	}
}

func TestNormalizeAgentMemoryValuesCanonicalizesStatuses(t *testing.T) {
	assertion, err := normalizeAgentMemoryAssertion(port.AgentMemoryAssertion{
		SubjectKey: "user:7", Predicate: "p", Object: "o", SourceType: "manual", SourceID: "source",
		Version: 1, Confidence: 1, Status: port.AgentMemoryAssertionStatus("ACTIVE"),
	})
	if err != nil || assertion.Status != port.AgentMemoryActive {
		t.Fatalf("normalized assertion status = %q, error = %v", assertion.Status, err)
	}
	filter, err := normalizeAgentMemoryFilter(port.AgentMemoryFilter{Status: port.AgentMemoryAssertionStatus("STALE")})
	if err != nil || filter.Status != port.AgentMemoryStale {
		t.Fatalf("normalized assertion filter status = %q, error = %v", filter.Status, err)
	}
	conflictFilter, err := normalizeAgentMemoryConflictFilter(port.AgentMemoryConflictFilter{Status: port.AgentMemoryConflictStatus("RESOLVED")})
	if err != nil || conflictFilter.Status != port.AgentMemoryConflictResolved {
		t.Fatalf("normalized conflict filter status = %q, error = %v", conflictFilter.Status, err)
	}
}

func TestMemoryRevisionAndConflictMappingsPreserveUTCAndMembers(t *testing.T) {
	zone := time.FixedZone("CST", 8*60*60)
	changedAt := time.Date(2026, 8, 29, 12, 0, 0, 0, zone)
	validUntil := changedAt.Add(time.Hour)
	revision := agentMemoryRevisionRow{
		AssertionID: "assertion-1", RevisionNo: 2, SubjectKey: "user:7", Predicate: "p", Object: "o",
		SourceType: "manual", SourceID: "source", Version: 1, Confidence: 0.8,
		Status: "active", ValidFrom: changedAt, ValidUntil: &validUntil, Reason: "revised", ChangedAt: changedAt,
	}
	gotRevision := revision.revision()
	if gotRevision.Status != port.AgentMemoryActive || gotRevision.ValidFrom.Location() != time.UTC || gotRevision.ValidUntil.Location() != time.UTC || gotRevision.ChangedAt.Location() != time.UTC {
		t.Fatalf("revision mapping = %+v", gotRevision)
	}
	conflict := agentMemoryConflictRow{
		ID: "conflict-1", SubjectKey: "user:7", Predicate: "p", Status: "open",
		WinnerAssertionID: "", Resolution: "", CreatedAt: changedAt, UpdatedAt: changedAt,
	}
	gotConflict := conflict.conflict()
	if gotConflict.Status != port.AgentMemoryConflictOpen || gotConflict.CreatedAt.Location() != time.UTC || gotConflict.UpdatedAt.Location() != time.UTC {
		t.Fatalf("conflict mapping = %+v", gotConflict)
	}
}

func TestNormalizeAgentMemoryConflictFilterRejectsAnonymousSubjectsAndBoundsPage(t *testing.T) {
	filter, err := normalizeAgentMemoryConflictFilter(port.AgentMemoryConflictFilter{SubjectKey: "USER:7", Size: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if filter.SubjectKey != "user:7" || filter.Status != port.AgentMemoryConflictOpen || filter.Current != 1 || filter.Size != 100 {
		t.Fatalf("normalized conflict filter = %+v", filter)
	}
	if _, err := normalizeAgentMemoryConflictFilter(port.AgentMemoryConflictFilter{SubjectKey: "ip:127.0.0.1"}); err == nil {
		t.Fatal("anonymous conflict filter was accepted")
	}
}
