package evaldata

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"benetnasch/app/domain/port"
)

//go:embed rag_dataset.json
var ragDatasetFS embed.FS

// RAGCase is a small, deterministic quality-evaluation case. Expected article
// IDs are used to judge citation provenance; they are not sent to the model.
type RAGCase struct {
	ID                 string          `json:"id"`
	Query              string          `json:"query"`
	Mode               port.SearchMode `json:"mode"`
	ExpectedArticleIDs []int           `json:"expectedArticleIds"`
	ExpectRefusal      bool            `json:"expectRefusal"`
}

// Normalize trims and validates one case, returning a copy with stable,
// duplicate-free expected article IDs.
func (c RAGCase) Normalize() (RAGCase, error) {
	c.ID = strings.TrimSpace(c.ID)
	c.Query = strings.TrimSpace(c.Query)
	if c.ID == "" {
		return RAGCase{}, fmt.Errorf("RAG evaluation case id is required")
	}
	if c.Query == "" {
		return RAGCase{}, fmt.Errorf("RAG evaluation case %q query is required", c.ID)
	}
	mode, err := port.NormalizeSearchMode(string(c.Mode))
	if err != nil {
		return RAGCase{}, fmt.Errorf("RAG evaluation case %q mode: %w", c.ID, err)
	}
	c.Mode = mode
	ids := make([]int, 0, len(c.ExpectedArticleIDs))
	seen := make(map[int]struct{}, len(c.ExpectedArticleIDs))
	for _, articleID := range c.ExpectedArticleIDs {
		if articleID <= 0 {
			return RAGCase{}, fmt.Errorf("RAG evaluation case %q contains an invalid article ID", c.ID)
		}
		if _, exists := seen[articleID]; exists {
			continue
		}
		seen[articleID] = struct{}{}
		ids = append(ids, articleID)
	}
	c.ExpectedArticleIDs = ids
	if c.ExpectRefusal && len(ids) != 0 {
		return RAGCase{}, fmt.Errorf("RAG evaluation case %q cannot expect citations and refusal", c.ID)
	}
	if !c.ExpectRefusal && len(ids) == 0 {
		return RAGCase{}, fmt.Errorf("RAG evaluation case %q must declare expected citations or refusal", c.ID)
	}
	return c, nil
}

// NormalizeRAGCases returns a canonical copy and rejects duplicate case IDs.
func NormalizeRAGCases(cases []RAGCase) ([]RAGCase, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("RAG evaluation dataset is empty")
	}
	normalized := make([]RAGCase, len(cases))
	seenIDs := make(map[string]struct{}, len(cases))
	positive, refusal := 0, 0
	for index, rawCase := range cases {
		current, err := rawCase.Normalize()
		if err != nil {
			return nil, fmt.Errorf("dataset case %d: %w", index, err)
		}
		if _, exists := seenIDs[current.ID]; exists {
			return nil, fmt.Errorf("duplicate RAG evaluation case ID %q", current.ID)
		}
		seenIDs[current.ID] = struct{}{}
		normalized[index] = current
		if current.ExpectRefusal {
			refusal++
		} else {
			positive++
		}
	}
	if positive == 0 || refusal == 0 {
		return nil, fmt.Errorf("RAG evaluation dataset must contain positive and refusal cases")
	}
	return normalized, nil
}

// LoadRAGCases loads and validates the embedded fixed RAG evaluation set.
func LoadRAGCases() ([]RAGCase, error) {
	raw, err := ragDatasetFS.ReadFile("rag_dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read RAG evaluation dataset: %w", err)
	}
	var cases []RAGCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, fmt.Errorf("decode RAG evaluation dataset: %w", err)
	}
	normalized, err := NormalizeRAGCases(cases)
	if err != nil {
		return nil, fmt.Errorf("validate RAG evaluation dataset: %w", err)
	}
	return normalized, nil
}
