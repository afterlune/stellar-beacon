package evaldata

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	visionPromptMaxRunes           = 4000
	visionLabelMaxRunes            = 300
	visionFixtureIDMaxLen          = 120
	visionScoreNotesLimit          = 2000
	RecommendedVisionRatersPerCase = 2
)

// VisionCase describes one image-understanding evaluation case. FixtureID is
// an opaque reference to a separately managed, licensed/original image
// bundle; the evaluator supplies the bytes and never sends this dataset's
// expected observations to the provider.
type VisionCase struct {
	ID             string   `json:"id"`
	Category       string   `json:"category"`
	FixtureID      string   `json:"fixtureId"`
	MIMEType       string   `json:"mimeType"`
	Prompt         string   `json:"prompt"`
	MustVerify     []string `json:"mustVerify"`
	MustNotClaim   []string `json:"mustNotClaim"`
	SafetyCritical bool     `json:"safetyCritical"`
}

// Normalize validates one vision case and returns a canonical copy. It is
// intentionally independent from any model or image provider so dataset
// validation remains safe to run in CI and before an explicit smoke window.
func (c VisionCase) Normalize() (VisionCase, error) {
	c.ID = strings.TrimSpace(c.ID)
	c.Category = strings.ToLower(strings.TrimSpace(c.Category))
	c.FixtureID = strings.TrimSpace(c.FixtureID)
	c.MIMEType = strings.ToLower(strings.TrimSpace(c.MIMEType))
	c.Prompt = strings.TrimSpace(c.Prompt)
	if !validVisionEvaluationID(c.ID) {
		return VisionCase{}, fmt.Errorf("vision evaluation case ID is invalid")
	}
	switch c.Category {
	case "scene", "ocr", "chart", "ambiguity", "safety", "privacy":
	default:
		return VisionCase{}, fmt.Errorf("vision evaluation case %q has unsupported category %q", c.ID, c.Category)
	}
	if !validVisionEvaluationID(c.FixtureID) || len(c.FixtureID) > visionFixtureIDMaxLen {
		return VisionCase{}, fmt.Errorf("vision evaluation case %q fixture ID is invalid", c.ID)
	}
	switch c.MIMEType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
	default:
		return VisionCase{}, fmt.Errorf("vision evaluation case %q has unsupported MIME type %q", c.ID, c.MIMEType)
	}
	if !validVisionEvaluationText(c.Prompt, visionPromptMaxRunes) {
		return VisionCase{}, fmt.Errorf("vision evaluation case %q prompt is invalid", c.ID)
	}
	var err error
	c.MustVerify, err = normalizeVisionLabels(c.MustVerify, "must-verify criterion", c.ID)
	if err != nil {
		return VisionCase{}, err
	}
	c.MustNotClaim, err = normalizeVisionLabels(c.MustNotClaim, "must-not-claim criterion", c.ID)
	if err != nil {
		return VisionCase{}, err
	}
	return c, nil
}

func normalizeVisionLabels(values []string, label, caseID string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("vision evaluation case %q requires at least one %s", caseID, label)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !validVisionEvaluationText(value, visionLabelMaxRunes) {
			return nil, fmt.Errorf("vision evaluation case %q contains an invalid %s", caseID, label)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func validVisionEvaluationID(value string) bool {
	if value == "" || len(value) > visionFixtureIDMaxLen {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

func validVisionEvaluationText(value string, maxRunes int) bool {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// NormalizeVisionCases returns a canonical copy and rejects duplicate IDs or
// an empty fixed dataset. Multiple cases may intentionally share a fixture
// while testing different prompts and safety boundaries.
func NormalizeVisionCases(cases []VisionCase) ([]VisionCase, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("vision evaluation dataset is empty")
	}
	result := make([]VisionCase, len(cases))
	seenIDs := make(map[string]struct{}, len(cases))
	for index, rawCase := range cases {
		current, err := rawCase.Normalize()
		if err != nil {
			return nil, fmt.Errorf("vision dataset case %d: %w", index, err)
		}
		if _, exists := seenIDs[current.ID]; exists {
			return nil, fmt.Errorf("duplicate vision evaluation case ID %q", current.ID)
		}
		seenIDs[current.ID] = struct{}{}
		result[index] = current
	}
	return result, nil
}

func ValidateVisionCases(cases []VisionCase) error {
	_, err := NormalizeVisionCases(cases)
	return err
}

// LoadVisionCases loads and validates the fixed offline evaluation contract.
// It does not read image bytes, call a Provider, or access application data.
func LoadVisionCases() ([]VisionCase, error) {
	data, err := datasetFS.ReadFile("vision_dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read vision evaluation dataset: %w", err)
	}
	var cases []VisionCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("decode vision evaluation dataset: %w", err)
	}
	normalized, err := NormalizeVisionCases(cases)
	if err != nil {
		return nil, fmt.Errorf("validate vision evaluation dataset: %w", err)
	}
	return normalized, nil
}
