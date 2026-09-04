package evaldata

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed dataset.json writing_dataset.json vision_dataset.json
var datasetFS embed.FS

type Case struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Input    string   `json:"input"`
	Expected []string `json:"expected"`
}

func Load() ([]Case, error) {
	data, err := datasetFS.ReadFile("dataset.json")
	if err != nil {
		return nil, fmt.Errorf("read evaluation dataset: %w", err)
	}
	var cases []Case
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("decode evaluation dataset: %w", err)
	}
	return cases, nil
}
