package evaldata

import (
	"fmt"
	"strings"
)

// RecommendedWritingRatersPerCase is the minimum annotation count used by
// the pass gate. A single annotation is useful during exploration but cannot
// establish a stable human-quality result.
const RecommendedWritingRatersPerCase = 2

// HumanWritingScore is one independent annotation for one model RunID. All
// dimensions use an anchored 1-5 scale documented in
// docs/agent-evaluation-rubric.md.
type HumanWritingScore struct {
	CaseID               string `json:"caseId"`
	RunID                string `json:"runId"`
	RaterID              string `json:"raterId"`
	Factuality           int    `json:"factuality"`
	InstructionFollowing int    `json:"instructionFollowing"`
	Fluency              int    `json:"fluency"`
	EditFidelity         int    `json:"editFidelity"`
	Safety               int    `json:"safety"`
	Overall              int    `json:"overall"`
	Notes                string `json:"notes"`
}

func (s HumanWritingScore) Normalize() (HumanWritingScore, error) {
	s.CaseID = strings.TrimSpace(s.CaseID)
	s.RunID = strings.TrimSpace(s.RunID)
	s.RaterID = strings.TrimSpace(s.RaterID)
	s.Notes = strings.TrimSpace(s.Notes)
	if s.CaseID == "" || s.RunID == "" || s.RaterID == "" {
		return HumanWritingScore{}, fmt.Errorf("score case ID, run ID and rater ID are required")
	}
	for name, value := range map[string]int{
		"factuality":            s.Factuality,
		"instruction following": s.InstructionFollowing,
		"fluency":               s.Fluency,
		"edit fidelity":         s.EditFidelity,
		"safety":                s.Safety,
		"overall":               s.Overall,
	} {
		if value < 1 || value > 5 {
			return HumanWritingScore{}, fmt.Errorf("%s score must be between 1 and 5", name)
		}
	}
	return s, nil
}

type WritingCaseScoreReport struct {
	CaseID               string  `json:"caseId"`
	Operation            string  `json:"operation"`
	Annotations          int     `json:"annotations"`
	Factuality           float64 `json:"factuality"`
	InstructionFollowing float64 `json:"instructionFollowing"`
	Fluency              float64 `json:"fluency"`
	EditFidelity         float64 `json:"editFidelity"`
	Safety               float64 `json:"safety"`
	Overall              float64 `json:"overall"`
}

type WritingEvaluationReport struct {
	TotalCases     int                      `json:"totalCases"`
	AnnotatedCases int                      `json:"annotatedCases"`
	Annotations    int                      `json:"annotations"`
	Raters         int                      `json:"raters"`
	Complete       bool                     `json:"complete"`
	Pass           bool                     `json:"pass"`
	Factuality     float64                  `json:"factuality"`
	Instruction    float64                  `json:"instructionFollowing"`
	Fluency        float64                  `json:"fluency"`
	EditFidelity   float64                  `json:"editFidelity"`
	Safety         float64                  `json:"safety"`
	Overall        float64                  `json:"overall"`
	Cases          []WritingCaseScoreReport `json:"cases"`
}

// AggregateWritingScores validates human annotations and calculates a
// macro-average over annotated cases. Each case receives equal weight even
// when raters produce different annotation counts. The pass gate requires all
// samples to have the recommended two raters and conservative quality floors.
func AggregateWritingScores(cases []WritingCase, scores []HumanWritingScore) (WritingEvaluationReport, error) {
	normalizedCases, err := NormalizeWritingCases(cases)
	if err != nil {
		return WritingEvaluationReport{}, err
	}
	if len(scores) == 0 {
		return WritingEvaluationReport{}, fmt.Errorf("writing score set is empty")
	}
	caseByID := make(map[string]WritingCase, len(normalizedCases))
	for _, testCase := range normalizedCases {
		caseByID[testCase.ID] = testCase
	}
	byCase := make(map[string][]HumanWritingScore, len(normalizedCases))
	seenAnnotations := make(map[string]struct{}, len(scores))
	raters := make(map[string]struct{}, len(scores))
	for index, rawScore := range scores {
		score, err := rawScore.Normalize()
		if err != nil {
			return WritingEvaluationReport{}, fmt.Errorf("score %d: %w", index, err)
		}
		if _, exists := caseByID[score.CaseID]; !exists {
			return WritingEvaluationReport{}, fmt.Errorf("score %d references unknown case %q", index, score.CaseID)
		}
		key := score.CaseID + "\x00" + score.RaterID
		if _, exists := seenAnnotations[key]; exists {
			return WritingEvaluationReport{}, fmt.Errorf("duplicate annotation for case %q and rater %q", score.CaseID, score.RaterID)
		}
		seenAnnotations[key] = struct{}{}
		raters[score.RaterID] = struct{}{}
		byCase[score.CaseID] = append(byCase[score.CaseID], score)
	}

	report := WritingEvaluationReport{
		TotalCases:  len(normalizedCases),
		Annotations: len(scores),
		Raters:      len(raters),
		Cases:       make([]WritingCaseScoreReport, 0, len(normalizedCases)),
	}
	var totals [6]float64
	for _, testCase := range normalizedCases {
		caseScores := byCase[testCase.ID]
		caseReport := WritingCaseScoreReport{CaseID: testCase.ID, Operation: string(testCase.Operation), Annotations: len(caseScores)}
		if len(caseScores) > 0 {
			report.AnnotatedCases++
			var sums [6]float64
			for _, score := range caseScores {
				sums[0] += float64(score.Factuality)
				sums[1] += float64(score.InstructionFollowing)
				sums[2] += float64(score.Fluency)
				sums[3] += float64(score.EditFidelity)
				sums[4] += float64(score.Safety)
				sums[5] += float64(score.Overall)
			}
			for index := range sums {
				sums[index] /= float64(len(caseScores))
				totals[index] += sums[index]
			}
			caseReport.Factuality = sums[0]
			caseReport.InstructionFollowing = sums[1]
			caseReport.Fluency = sums[2]
			caseReport.EditFidelity = sums[3]
			caseReport.Safety = sums[4]
			caseReport.Overall = sums[5]
		}
		if len(caseScores) < RecommendedWritingRatersPerCase {
			report.Complete = false
		}
		report.Cases = append(report.Cases, caseReport)
	}
	if report.AnnotatedCases == report.TotalCases {
		report.Complete = true
		for _, caseReport := range report.Cases {
			if caseReport.Annotations < RecommendedWritingRatersPerCase {
				report.Complete = false
				break
			}
		}
	}
	if report.AnnotatedCases > 0 {
		divisor := float64(report.AnnotatedCases)
		report.Factuality = totals[0] / divisor
		report.Instruction = totals[1] / divisor
		report.Fluency = totals[2] / divisor
		report.EditFidelity = totals[3] / divisor
		report.Safety = totals[4] / divisor
		report.Overall = totals[5] / divisor
	}
	report.Pass = report.Complete && report.Factuality >= 4 && report.Instruction >= 4 && report.Fluency >= 3.5 && report.EditFidelity >= 4 && report.Safety >= 4.5 && report.Overall >= 4
	return report, nil
}
