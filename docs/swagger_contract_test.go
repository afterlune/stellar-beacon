package docs

import (
	"encoding/json"
	"testing"
)

func TestAgentRoutesArePresentInSwaggerContract(t *testing.T) {
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal([]byte(SwaggerInfo.ReadDoc()), &document); err != nil {
		t.Fatalf("generated Swagger document is invalid JSON: %v", err)
	}

	required := map[string][]string{
		"/agent/chat":                 {"post"},
		"/agent/features":             {"get"},
		"/agent/vitals":               {"get"},
		"/agent/sessions/{id}":        {"delete"},
		"/agent/sessions/{id}/events": {"get"},
		"/galaxy":                     {"get"},
		"/dreams":                     {"get"},
		"/radio":                      {"get"},
		"/videos":                     {"get"},
		"/capsules":                   {"post"},
		"/capsules/{id}":              {"get"},
		"/capsules/{id}/seal":         {"post"},
		"/admin/agent/emergency":      {"put"},
		"/admin/ai/profile":           {"get", "patch"},
		"/admin/ai/review-policy":     {"get", "patch"},
		"/admin/ai/observability":     {"get"},
		"/admin/ai/memory/assertions": {"get"},
		"/admin/ai/memory/assertions/{id}/history": {"get"},
		"/admin/ai/memory/assertions/{id}":         {"delete"},
		"/admin/ai/memory/conflicts":               {"get"},
		"/admin/ai/memory/conflicts/{id}/resolve":  {"post"},
		"/admin/ai/memory/conflicts/{id}/reject":   {"post"},
	}
	for path, methods := range required {
		operations, ok := document.Paths[path]
		if !ok {
			t.Errorf("Agent route %s is missing from generated Swagger", path)
			continue
		}
		for _, method := range methods {
			if _, ok := operations[method]; !ok {
				t.Errorf("Agent route %s %s operation is missing from generated Swagger", method, path)
			}
		}
	}
}
