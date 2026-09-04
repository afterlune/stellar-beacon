package evaldata

import (
	"encoding/json"
	"fmt"
	"strings"

	"benetnasch/app/domain/port"
)

// WritingCase is an original Chinese article sample for human evaluation.
// It deliberately stores expectations and facts to preserve instead of a
// single "gold" answer: several writing operations can be valid while still
// being judged for factuality, instruction following and edit fidelity.
type WritingCase struct {
	ID                string                `json:"id"`
	Operation         port.WritingOperation `json:"operation"`
	Title             string                `json:"title"`
	Content           string                `json:"content"`
	Instruction       string                `json:"instruction"`
	MustPreserve      []string              `json:"mustPreserve"`
	ExpectedBehaviors []string              `json:"expectedBehaviors"`
}

func (c WritingCase) Normalize() (WritingCase, error) {
	c.ID = strings.TrimSpace(c.ID)
	c.Title = strings.TrimSpace(c.Title)
	c.Content = strings.TrimSpace(c.Content)
	c.Instruction = strings.TrimSpace(c.Instruction)
	if c.ID == "" {
		return WritingCase{}, fmt.Errorf("writing evaluation case ID is required")
	}
	operation, err := port.NormalizeWritingOperation(string(c.Operation))
	if err != nil {
		return WritingCase{}, fmt.Errorf("writing evaluation case %q operation: %w", c.ID, err)
	}
	c.Operation = operation
	if c.Content == "" {
		return WritingCase{}, fmt.Errorf("writing evaluation case %q content is required", c.ID)
	}
	if c.Instruction == "" {
		return WritingCase{}, fmt.Errorf("writing evaluation case %q instruction is required", c.ID)
	}
	c.MustPreserve, err = normalizeWritingLabels(c.MustPreserve, "must-preserve fact", c.ID)
	if err != nil {
		return WritingCase{}, err
	}
	c.ExpectedBehaviors, err = normalizeWritingLabels(c.ExpectedBehaviors, "expected behavior", c.ID)
	if err != nil {
		return WritingCase{}, err
	}
	return c, nil
}

func normalizeWritingLabels(values []string, label, caseID string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("writing evaluation case %q requires at least one %s", caseID, label)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("writing evaluation case %q contains an empty %s", caseID, label)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

// NormalizeWritingCases validates uniqueness and guarantees that every
// supported writing operation is represented by at least one sample.
func NormalizeWritingCases(cases []WritingCase) ([]WritingCase, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("writing evaluation dataset is empty")
	}
	result := make([]WritingCase, len(cases))
	seenIDs := make(map[string]struct{}, len(cases))
	seenOperations := make(map[port.WritingOperation]struct{}, len(cases))
	for index, input := range cases {
		canonical, err := input.Normalize()
		if err != nil {
			return nil, fmt.Errorf("writing dataset case %d: %w", index, err)
		}
		if _, exists := seenIDs[canonical.ID]; exists {
			return nil, fmt.Errorf("duplicate writing evaluation case ID %q", canonical.ID)
		}
		seenIDs[canonical.ID] = struct{}{}
		seenOperations[canonical.Operation] = struct{}{}
		result[index] = canonical
	}
	operations := []port.WritingOperation{
		port.WritingOperationContinue,
		port.WritingOperationPolish,
		port.WritingOperationSummary,
		port.WritingOperationTitle,
		port.WritingOperationCorrect,
	}
	for _, operation := range operations {
		if _, exists := seenOperations[operation]; !exists {
			return nil, fmt.Errorf("writing evaluation dataset has no %q case", operation)
		}
	}
	return result, nil
}

func ValidateWritingCases(cases []WritingCase) error {
	_, err := NormalizeWritingCases(cases)
	return err
}

// LoadWritingDataset loads the fixed, embedded, offline Chinese writing
// samples. It never calls a model and never reads application data.
func LoadWritingDataset() ([]WritingCase, error) {
	data, err := datasetFS.ReadFile("writing_dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read writing evaluation dataset: %w", err)
	}
	var cases []WritingCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("decode writing evaluation dataset: %w", err)
	}
	normalized, err := NormalizeWritingCases(cases)
	if err != nil {
		return nil, fmt.Errorf("validate writing evaluation dataset: %w", err)
	}
	return normalized, nil
}
