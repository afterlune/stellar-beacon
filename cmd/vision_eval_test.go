package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"benetnasch/app/infra/ai/evaldata"
)

func TestDecodeVisionScoresAcceptsJSONArrayAndJSONL(t *testing.T) {
	scoreJSON := `{"caseId":"vision-scene-001","runId":"run-1","raterId":"rater-a","observationAccuracy":5,"instructionFollowing":4,"uncertaintyCalibration":4,"safety":5,"overall":4,"notes":"ok"}`
	arrayInput := "[" + scoreJSON + "]"
	jsonlInput := scoreJSON + "\n" + `{"caseId":"vision-ocr-001","runId":"run-1","raterId":"rater-a","observationAccuracy":5,"instructionFollowing":5,"uncertaintyCalibration":5,"safety":5,"overall":5,"notes":""}`

	for name, input := range map[string]string{"array": arrayInput, "jsonl": jsonlInput} {
		t.Run(name, func(t *testing.T) {
			scores, err := decodeVisionScores(strings.NewReader(input))
			if err != nil {
				t.Fatalf("decodeVisionScores() error = %v", err)
			}
			want := 2
			if name == "array" {
				want = 1
			}
			if len(scores) != want {
				t.Fatalf("decoded %d scores, want %d", len(scores), want)
			}
		})
	}
}

func TestDecodeVisionScoresRejectsUnknownFieldsAndOversizedInput(t *testing.T) {
	if _, err := decodeVisionScores(strings.NewReader(`{"caseId":"case","unknown":true}`)); err == nil {
		t.Fatal("decodeVisionScores() accepted an unknown field")
	}
	if _, err := decodeVisionScores(strings.NewReader(strings.Repeat("x", int(maxVisionScoreFileBytes)+1))); err == nil {
		t.Fatal("decodeVisionScores() accepted an oversized score file")
	}
}

func TestVisionEvalReportUsesFixedDataset(t *testing.T) {
	cases, err := evaldata.LoadVisionCases()
	if err != nil {
		t.Fatalf("LoadVisionCases() error = %v", err)
	}
	if len(cases) != 6 {
		t.Fatalf("Vision cases = %d, want 6", len(cases))
	}
}

func TestOpenExternalVisionScoreFileRejectsRepositoryPath(t *testing.T) {
	repoRoot, err := findVisionEvaluationRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	insideRepository := filepath.Join(repoRoot, "cmd", "vision_eval.go")
	if _, err := openExternalVisionScoreFile(insideRepository); err == nil || !strings.Contains(err.Error(), "outside the repository") {
		t.Fatalf("repository score path error = %v", err)
	}
}

func TestOpenExternalVisionScoreFileAcceptsRegularExternalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scores.jsonl")
	if err := os.WriteFile(path, []byte(`{"caseId":"vision-scene-001"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := openExternalVisionScoreFile(path)
	if err != nil {
		t.Fatalf("openExternalVisionScoreFile() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
