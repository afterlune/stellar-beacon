package evaldata

import "testing"

func TestLoadFaultCasesCoversAllRecoveryBoundaries(t *testing.T) {
	cases, err := LoadFaultCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 8 {
		t.Fatalf("fault cases = %d, want 8", len(cases))
	}
	seenKinds := make(map[FaultKind]struct{})
	for _, current := range cases {
		seenKinds[current.Kind] = struct{}{}
	}
	if len(seenKinds) != len(faultKinds) {
		t.Fatalf("fault kinds = %#v, want all recovery boundaries", seenKinds)
	}
}

func TestEvaluateFaultObservationsRequiresEveryInvariant(t *testing.T) {
	cases, err := LoadFaultCases()
	if err != nil {
		t.Fatal(err)
	}
	observations := make([]FaultObservation, 0, len(cases))
	for _, current := range cases {
		observations = append(observations, FaultObservation{CaseID: current.ID, Satisfied: append([]FaultInvariant(nil), current.Invariants...)})
	}
	report, err := EvaluateFaultObservations(cases, observations)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || report.Passed != len(cases) || report.Failed != 0 {
		t.Fatalf("complete report = %#v", report)
	}

	observations[0].Satisfied = observations[0].Satisfied[:1]
	report, err = EvaluateFaultObservations(cases, observations)
	if err != nil {
		t.Fatal(err)
	}
	if report.Complete || report.Cases[0].Passed || len(report.Cases[0].Missing) == 0 {
		t.Fatalf("incomplete report = %#v", report)
	}
}

func TestEvaluateFaultObservationsRejectsUnknownOrDuplicateEvidence(t *testing.T) {
	cases, err := LoadFaultCases()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EvaluateFaultObservations(cases, []FaultObservation{{CaseID: "missing"}}); err == nil {
		t.Fatal("unknown fault case was accepted")
	}
	if _, err := EvaluateFaultObservations(cases, []FaultObservation{
		{CaseID: cases[0].ID, Satisfied: []FaultInvariant{"not-an-invariant"}},
	}); err == nil {
		t.Fatal("unknown fault invariant was accepted")
	}
	if _, err := EvaluateFaultObservations(cases, []FaultObservation{
		{CaseID: cases[0].ID},
		{CaseID: cases[0].ID},
	}); err == nil {
		t.Fatal("duplicate fault observation was accepted")
	}
	if _, err := EvaluateFaultObservations(cases, []FaultObservation{
		{CaseID: cases[0].ID, Satisfied: []FaultInvariant{cases[0].Invariants[0], cases[0].Invariants[0]}},
	}); err == nil {
		t.Fatal("repeated fault invariant was accepted")
	}
}
