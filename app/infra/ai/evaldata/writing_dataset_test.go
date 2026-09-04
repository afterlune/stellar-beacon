package evaldata

import (
	"testing"

	"benetnasch/app/domain/port"
)

func TestLoadWritingDatasetCoversEveryOperation(t *testing.T) {
	cases, err := LoadWritingDataset()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 10 {
		t.Fatalf("writing case count = %d, want 10", len(cases))
	}
	operations := make(map[port.WritingOperation]int)
	for _, testCase := range cases {
		operations[testCase.Operation]++
		if testCase.ID == "" || testCase.Content == "" || len(testCase.MustPreserve) == 0 || len(testCase.ExpectedBehaviors) == 0 {
			t.Fatalf("incomplete writing case: %+v", testCase)
		}
	}
	for _, operation := range []port.WritingOperation{
		port.WritingOperationContinue,
		port.WritingOperationPolish,
		port.WritingOperationSummary,
		port.WritingOperationTitle,
		port.WritingOperationCorrect,
	} {
		if operations[operation] == 0 {
			t.Fatalf("operation %q has no evaluation case", operation)
		}
	}
}

func TestNormalizeWritingCasesRejectsDuplicateAndMissingCoverage(t *testing.T) {
	base := WritingCase{
		ID:                "case-1",
		Operation:         port.WritingOperationContinue,
		Content:           "正文",
		Instruction:       "要求",
		MustPreserve:      []string{"事实"},
		ExpectedBehaviors: []string{"续写"},
	}
	if _, err := NormalizeWritingCases([]WritingCase{base, base}); err == nil {
		t.Fatal("duplicate case IDs were accepted")
	}
	if _, err := NormalizeWritingCases([]WritingCase{base}); err == nil {
		t.Fatal("dataset with missing operations was accepted")
	}
	base.MustPreserve = []string{"事实", "事实"}
	if _, err := base.Normalize(); err != nil {
		t.Fatal(err)
	}
}

func TestAggregateWritingScoresUsesEqualCaseWeightAndPassGate(t *testing.T) {
	cases, err := LoadWritingDataset()
	if err != nil {
		t.Fatal(err)
	}
	scores := make([]HumanWritingScore, 0, len(cases)*2)
	for _, testCase := range cases {
		for _, rater := range []string{"rater-a", "rater-b"} {
			scores = append(scores, HumanWritingScore{
				CaseID:               testCase.ID,
				RunID:                "run-" + testCase.ID,
				RaterID:              rater,
				Factuality:           5,
				InstructionFollowing: 4,
				Fluency:              4,
				EditFidelity:         5,
				Safety:               5,
				Overall:              4,
			})
		}
	}
	report, err := AggregateWritingScores(cases, scores)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || !report.Pass || report.TotalCases != 10 || report.AnnotatedCases != 10 || report.Annotations != 20 || report.Raters != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Factuality != 5 || report.Instruction != 4 || report.Overall != 4 {
		t.Fatalf("unexpected means: %+v", report)
	}

	incomplete, err := AggregateWritingScores(cases, scores[:len(scores)-1])
	if err != nil {
		t.Fatal(err)
	}
	if incomplete.Complete || incomplete.Pass {
		t.Fatalf("incomplete report passed: %+v", incomplete)
	}
}

func TestAggregateWritingScoresRejectsDuplicateUnknownAndOutOfRangeAnnotations(t *testing.T) {
	cases, err := LoadWritingDataset()
	if err != nil {
		t.Fatal(err)
	}
	base := HumanWritingScore{
		CaseID:               cases[0].ID,
		RunID:                "run-1",
		RaterID:              "rater-a",
		Factuality:           4,
		InstructionFollowing: 4,
		Fluency:              4,
		EditFidelity:         4,
		Safety:               4,
		Overall:              4,
	}
	for name, scores := range map[string][]HumanWritingScore{
		"duplicate":    {base, base},
		"unknown case": {HumanWritingScore{CaseID: "missing", RunID: "run", RaterID: "rater", Factuality: 4, InstructionFollowing: 4, Fluency: 4, EditFidelity: 4, Safety: 4, Overall: 4}},
		"out of range": {func() HumanWritingScore { value := base; value.Safety = 6; return value }()},
	} {
		if _, err := AggregateWritingScores(cases, scores); err == nil {
			t.Fatalf("%s annotations were accepted", name)
		}
	}
}
