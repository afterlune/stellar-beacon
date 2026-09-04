package evaldata

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"benetnasch/app/domain/port"
)

//go:embed dataset.json
var datasetFS embed.FS

// Case is one deterministic retrieval evaluation query. Expected contains
// the hit IDs that are relevant to the query, in no particular ranking order.
// A no-result case must set ExpectNoResult and leave Expected empty.
type Case struct {
	ID             string          `json:"id"`
	Query          string          `json:"query"`
	Mode           port.SearchMode `json:"mode"`
	Expected       []string        `json:"expected"`
	ExpectNoResult bool            `json:"expectNoResult"`
}

// Normalize trims query and IDs, applies the public search-mode default, and
// removes duplicate expected IDs while preserving their first-seen order.
func (c Case) Normalize() (Case, error) {
	c.ID = strings.TrimSpace(c.ID)
	c.Query = strings.TrimSpace(c.Query)
	if c.ID == "" {
		return Case{}, fmt.Errorf("evaluation case id is required")
	}
	if c.Query == "" {
		return Case{}, fmt.Errorf("evaluation case %q query is required", c.ID)
	}
	mode, err := port.NormalizeSearchMode(string(c.Mode))
	if err != nil {
		return Case{}, fmt.Errorf("evaluation case %q mode: %w", c.ID, err)
	}
	c.Mode = mode

	expected := make([]string, 0, len(c.Expected))
	seen := make(map[string]struct{}, len(c.Expected))
	for _, rawID := range c.Expected {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return Case{}, fmt.Errorf("evaluation case %q contains an empty expected hit ID", c.ID)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		expected = append(expected, id)
	}
	c.Expected = expected
	if c.ExpectNoResult && len(c.Expected) != 0 {
		return Case{}, fmt.Errorf("evaluation case %q cannot expect hits and no result", c.ID)
	}
	if !c.ExpectNoResult && len(c.Expected) == 0 {
		return Case{}, fmt.Errorf("evaluation case %q must declare expected hits or expect no result", c.ID)
	}
	return c, nil
}

// NormalizeCases returns a canonical copy suitable for evaluation.
func NormalizeCases(cases []Case) ([]Case, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("retrieval evaluation dataset is empty")
	}

	normalized := make([]Case, len(cases))
	seenIDs := make(map[string]struct{}, len(cases))
	positive := 0
	noResult := 0
	for index, testCase := range cases {
		canonical, err := testCase.Normalize()
		if err != nil {
			return nil, fmt.Errorf("dataset case %d: %w", index, err)
		}
		if _, exists := seenIDs[canonical.ID]; exists {
			return nil, fmt.Errorf("duplicate evaluation case ID %q", canonical.ID)
		}
		seenIDs[canonical.ID] = struct{}{}
		normalized[index] = canonical
		if canonical.ExpectNoResult {
			noResult++
		} else {
			positive++
		}
	}
	if positive == 0 {
		return nil, fmt.Errorf("retrieval evaluation dataset has no positive cases")
	}
	if noResult == 0 {
		return nil, fmt.Errorf("retrieval evaluation dataset has no no-result cases")
	}
	return normalized, nil
}

// Validate checks the dataset without exposing mutable normalized state.
func Validate(cases []Case) error {
	_, err := NormalizeCases(cases)
	return err
}

// Load returns the versioned, embedded retrieval evaluation dataset. It is
// intentionally offline and does not require PostgreSQL, Redis or Meilisearch.
func Load() ([]Case, error) {
	data, err := datasetFS.ReadFile("dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read retrieval evaluation dataset: %w", err)
	}
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("decode retrieval evaluation dataset: %w", err)
	}
	normalized, err := NormalizeCases(cases)
	if err != nil {
		return nil, fmt.Errorf("validate retrieval evaluation dataset: %w", err)
	}
	return normalized, nil
}
