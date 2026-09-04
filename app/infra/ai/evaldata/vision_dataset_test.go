package evaldata

import "testing"

func TestLoadVisionCasesValidatesFixedSafetyCoverage(t *testing.T) {
	cases, err := LoadVisionCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 6 {
		t.Fatalf("vision case count = %d, want 6", len(cases))
	}
	critical := 0
	for _, testCase := range cases {
		if testCase.ID == "" || testCase.FixtureID == "" || testCase.Prompt == "" || len(testCase.MustVerify) == 0 || len(testCase.MustNotClaim) == 0 {
			t.Fatalf("incomplete vision case: %+v", testCase)
		}
		if testCase.SafetyCritical {
			critical++
		}
	}
	if critical < 3 {
		t.Fatalf("safety-critical vision case count = %d, want at least 3", critical)
	}
}

func TestNormalizeVisionCasesRejectsInvalidContract(t *testing.T) {
	base := VisionCase{
		ID: "vision-1", Category: "scene", FixtureID: "fixture-1", MIMEType: "image/png",
		Prompt: "描述图片", MustVerify: []string{"对象"}, MustNotClaim: []string{"身份"},
	}
	if _, err := NormalizeVisionCases([]VisionCase{base, base}); err == nil {
		t.Fatal("duplicate vision case IDs were accepted")
	}
	for name, current := range map[string]VisionCase{
		"unsupported MIME":  func() VisionCase { value := base; value.MIMEType = "image/svg+xml"; return value }(),
		"unsafe fixture ID": func() VisionCase { value := base; value.FixtureID = "../fixture"; return value }(),
		"missing criteria":  func() VisionCase { value := base; value.MustVerify = nil; return value }(),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := current.Normalize(); err == nil {
				t.Fatal("invalid vision case was accepted")
			}
		})
	}
}

func TestAggregateVisionScoresUsesEqualCaseWeightAndSafetyGate(t *testing.T) {
	cases, err := LoadVisionCases()
	if err != nil {
		t.Fatal(err)
	}
	scores := make([]HumanVisionScore, 0, len(cases)*2)
	for _, testCase := range cases {
		for _, rater := range []string{"rater-a", "rater-b"} {
			scores = append(scores, HumanVisionScore{
				CaseID: testCase.ID, RunID: "run-vision-1", RaterID: rater,
				ObservationAccuracy: 5, InstructionFollowing: 4,
				UncertaintyCalibration: 4, Safety: 5, Overall: 4,
			})
		}
	}
	report, err := AggregateVisionScores(cases, scores)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Complete || !report.Pass || report.TotalCases != 6 || report.Annotations != 12 || report.Raters != 2 {
		t.Fatalf("unexpected vision report: %+v", report)
	}
	if report.ObservationAccuracy != 5 || report.Safety != 5 || report.Overall != 4 {
		t.Fatalf("unexpected vision means: %+v", report)
	}

	fatalSafety := scores[0]
	fatalSafety.Safety = 1
	blockedScores := append([]HumanVisionScore(nil), scores...)
	blockedScores[0] = fatalSafety
	blocked, err := AggregateVisionScores(cases, blockedScores)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Pass || blocked.SafetyCriticalFailures != 1 {
		t.Fatalf("fatal safety annotation passed: %+v", blocked)
	}

	incomplete, err := AggregateVisionScores(cases, scores[:len(scores)-1])
	if err != nil {
		t.Fatal(err)
	}
	if incomplete.Complete || incomplete.Pass {
		t.Fatalf("incomplete vision report passed: %+v", incomplete)
	}
}

func TestAggregateVisionScoresRejectsMixedRunsAndDuplicates(t *testing.T) {
	cases, err := LoadVisionCases()
	if err != nil {
		t.Fatal(err)
	}
	base := HumanVisionScore{
		CaseID: cases[0].ID, RunID: "run-1", RaterID: "rater-a",
		ObservationAccuracy: 4, InstructionFollowing: 4,
		UncertaintyCalibration: 4, Safety: 5, Overall: 4,
	}
	for name, scores := range map[string][]HumanVisionScore{
		"duplicate":    {base, base},
		"mixed runs":   {base, func() HumanVisionScore { value := base; value.RaterID = "rater-b"; value.RunID = "run-2"; return value }()},
		"out of range": {func() HumanVisionScore { value := base; value.Safety = 6; return value }()},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := AggregateVisionScores(cases, scores); err == nil {
				t.Fatalf("%s annotations were accepted", name)
			}
		})
	}
}
