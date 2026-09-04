package evaldata

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed fault_dataset.json
var faultDatasetFS embed.FS

// FaultKind identifies a recovery boundary that must be tested independently.
// The dataset is deliberately provider-neutral so the same cases can be used
// by unit, isolated-integration, and release-window evidence collectors.
type FaultKind string

const (
	FaultKindSessionRecovery FaultKind = "session_recovery"
	FaultKindEventReplay     FaultKind = "event_replay"
	FaultKindApprovalReplay  FaultKind = "approval_replay"
	FaultKindMemoryConflict  FaultKind = "memory_conflict"
)

// FaultInvariant is a closed vocabulary for evidence that a case passed. A
// collector must not invent a free-form success label, otherwise reports from
// different environments cannot be compared safely.
type FaultInvariant string

const (
	FaultInvariantSessionReloaded         FaultInvariant = "session_reloaded"
	FaultInvariantTurnNotDuplicated       FaultInvariant = "turn_not_duplicated"
	FaultInvariantOwnerScopePreserved     FaultInvariant = "owner_scope_preserved"
	FaultInvariantSessionTTLBounded       FaultInvariant = "session_ttl_bounded"
	FaultInvariantCurrentTurnOnly         FaultInvariant = "current_turn_only"
	FaultInvariantAfterSequenceExclusive  FaultInvariant = "after_sequence_exclusive"
	FaultInvariantReplayFlagSet           FaultInvariant = "replay_flag_set"
	FaultInvariantCrossOwnerEmpty         FaultInvariant = "cross_owner_empty"
	FaultInvariantProviderNotCalled       FaultInvariant = "provider_not_called"
	FaultInvariantOneActionPersisted      FaultInvariant = "one_action_persisted"
	FaultInvariantIdempotentResult        FaultInvariant = "idempotent_result"
	FaultInvariantDigestBindingRejected   FaultInvariant = "digest_binding_rejected"
	FaultInvariantConflictRetained        FaultInvariant = "conflict_retained"
	FaultInvariantPriorAssertionPreserved FaultInvariant = "prior_assertion_preserved"
	FaultInvariantAnonymousWriteRejected  FaultInvariant = "anonymous_write_rejected"
	FaultInvariantDurableUserRequired     FaultInvariant = "durable_user_required"
)

var faultKinds = []FaultKind{
	FaultKindSessionRecovery,
	FaultKindEventReplay,
	FaultKindApprovalReplay,
	FaultKindMemoryConflict,
}

var faultInvariants = map[FaultInvariant]struct{}{
	FaultInvariantSessionReloaded:         {},
	FaultInvariantTurnNotDuplicated:       {},
	FaultInvariantOwnerScopePreserved:     {},
	FaultInvariantSessionTTLBounded:       {},
	FaultInvariantCurrentTurnOnly:         {},
	FaultInvariantAfterSequenceExclusive:  {},
	FaultInvariantReplayFlagSet:           {},
	FaultInvariantCrossOwnerEmpty:         {},
	FaultInvariantProviderNotCalled:       {},
	FaultInvariantOneActionPersisted:      {},
	FaultInvariantIdempotentResult:        {},
	FaultInvariantDigestBindingRejected:   {},
	FaultInvariantConflictRetained:        {},
	FaultInvariantPriorAssertionPreserved: {},
	FaultInvariantAnonymousWriteRejected:  {},
	FaultInvariantDurableUserRequired:     {},
}

// FaultCase describes one deterministic failure injection and the facts that
// must be collected before a release can claim the scenario passed.
type FaultCase struct {
	ID         string           `json:"id"`
	Kind       FaultKind        `json:"kind"`
	Trigger    string           `json:"trigger"`
	Invariants []FaultInvariant `json:"invariants"`
}

func (c FaultCase) Normalize() (FaultCase, error) {
	c.ID = strings.TrimSpace(c.ID)
	c.Trigger = strings.TrimSpace(c.Trigger)
	if c.ID == "" {
		return FaultCase{}, fmt.Errorf("fault evaluation case ID is required")
	}
	if c.Trigger == "" {
		return FaultCase{}, fmt.Errorf("fault evaluation case %q trigger is required", c.ID)
	}
	if !isFaultKind(c.Kind) {
		return FaultCase{}, fmt.Errorf("fault evaluation case %q kind %q is invalid", c.ID, c.Kind)
	}
	if len(c.Invariants) == 0 {
		return FaultCase{}, fmt.Errorf("fault evaluation case %q requires invariants", c.ID)
	}
	seen := make(map[FaultInvariant]struct{}, len(c.Invariants))
	canonical := make([]FaultInvariant, 0, len(c.Invariants))
	for _, invariant := range c.Invariants {
		invariant = FaultInvariant(strings.TrimSpace(string(invariant)))
		if _, ok := faultInvariants[invariant]; !ok {
			return FaultCase{}, fmt.Errorf("fault evaluation case %q invariant %q is invalid", c.ID, invariant)
		}
		if _, exists := seen[invariant]; exists {
			continue
		}
		seen[invariant] = struct{}{}
		canonical = append(canonical, invariant)
	}
	c.Invariants = canonical
	return c, nil
}

// NormalizeFaultCases validates IDs and requires all four recovery boundaries
// to be present in the fixed dataset.
func NormalizeFaultCases(cases []FaultCase) ([]FaultCase, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("fault evaluation dataset is empty")
	}
	result := make([]FaultCase, len(cases))
	seenIDs := make(map[string]struct{}, len(cases))
	seenKinds := make(map[FaultKind]struct{}, len(faultKinds))
	for index, raw := range cases {
		current, err := raw.Normalize()
		if err != nil {
			return nil, fmt.Errorf("fault dataset case %d: %w", index, err)
		}
		if _, exists := seenIDs[current.ID]; exists {
			return nil, fmt.Errorf("duplicate fault evaluation case ID %q", current.ID)
		}
		seenIDs[current.ID] = struct{}{}
		seenKinds[current.Kind] = struct{}{}
		result[index] = current
	}
	for _, kind := range faultKinds {
		if _, exists := seenKinds[kind]; !exists {
			return nil, fmt.Errorf("fault evaluation dataset has no %q case", kind)
		}
	}
	return result, nil
}

func isFaultKind(kind FaultKind) bool {
	for _, allowed := range faultKinds {
		if kind == allowed {
			return true
		}
	}
	return false
}

// LoadFaultCases loads the fixed, embedded, offline recovery dataset.
func LoadFaultCases() ([]FaultCase, error) {
	raw, err := faultDatasetFS.ReadFile("fault_dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read fault evaluation dataset: %w", err)
	}
	var cases []FaultCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, fmt.Errorf("decode fault evaluation dataset: %w", err)
	}
	normalized, err := NormalizeFaultCases(cases)
	if err != nil {
		return nil, fmt.Errorf("validate fault evaluation dataset: %w", err)
	}
	return normalized, nil
}

// FaultObservation contains only the invariants actually demonstrated by a
// test or release-window probe. It intentionally does not accept arbitrary
// evidence text as a substitute for a closed-set assertion.
type FaultObservation struct {
	CaseID    string           `json:"caseId"`
	Satisfied []FaultInvariant `json:"satisfied"`
}

type FaultCaseResult struct {
	CaseID  string           `json:"caseId"`
	Kind    FaultKind        `json:"kind"`
	Passed  bool             `json:"passed"`
	Missing []FaultInvariant `json:"missing,omitempty"`
}

type FaultEvaluationReport struct {
	Total    int               `json:"total"`
	Passed   int               `json:"passed"`
	Failed   int               `json:"failed"`
	Complete bool              `json:"complete"`
	Cases    []FaultCaseResult `json:"cases"`
}

// EvaluateFaultObservations compares collected facts with the fixed dataset.
// Missing observations are report failures rather than silently omitted cases;
// unknown case IDs or invariant names are evaluator errors.
func EvaluateFaultObservations(cases []FaultCase, observations []FaultObservation) (FaultEvaluationReport, error) {
	normalized, err := NormalizeFaultCases(cases)
	if err != nil {
		return FaultEvaluationReport{}, err
	}
	caseByID := make(map[string]FaultCase, len(normalized))
	for _, current := range normalized {
		caseByID[current.ID] = current
	}
	observationByID := make(map[string]map[FaultInvariant]struct{}, len(observations))
	for index, observation := range observations {
		caseID := strings.TrimSpace(observation.CaseID)
		current, ok := caseByID[caseID]
		if !ok {
			return FaultEvaluationReport{}, fmt.Errorf("fault observation %d references unknown case %q", index, caseID)
		}
		if _, exists := observationByID[caseID]; exists {
			return FaultEvaluationReport{}, fmt.Errorf("duplicate fault observation for case %q", caseID)
		}
		expected := make(map[FaultInvariant]struct{}, len(current.Invariants))
		for _, invariant := range current.Invariants {
			expected[invariant] = struct{}{}
		}
		satisfied := make(map[FaultInvariant]struct{}, len(observation.Satisfied))
		for _, rawInvariant := range observation.Satisfied {
			invariant := FaultInvariant(strings.TrimSpace(string(rawInvariant)))
			if _, known := faultInvariants[invariant]; !known {
				return FaultEvaluationReport{}, fmt.Errorf("fault observation %q contains unknown invariant %q", caseID, invariant)
			}
			if _, required := expected[invariant]; !required {
				return FaultEvaluationReport{}, fmt.Errorf("fault observation %q asserts non-required invariant %q", caseID, invariant)
			}
			if _, duplicate := satisfied[invariant]; duplicate {
				return FaultEvaluationReport{}, fmt.Errorf("fault observation %q repeats invariant %q", caseID, invariant)
			}
			satisfied[invariant] = struct{}{}
		}
		observationByID[caseID] = satisfied
	}

	report := FaultEvaluationReport{
		Total: len(normalized),
		Cases: make([]FaultCaseResult, 0, len(normalized)),
	}
	for _, current := range normalized {
		satisfied, observed := observationByID[current.ID]
		result := FaultCaseResult{CaseID: current.ID, Kind: current.Kind, Passed: observed}
		for _, invariant := range current.Invariants {
			if !observed {
				result.Missing = append(result.Missing, invariant)
				continue
			}
			if _, ok := satisfied[invariant]; !ok {
				result.Missing = append(result.Missing, invariant)
			}
		}
		result.Passed = len(result.Missing) == 0
		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.Cases = append(report.Cases, result)
	}
	report.Complete = report.Passed == report.Total
	return report, nil
}
