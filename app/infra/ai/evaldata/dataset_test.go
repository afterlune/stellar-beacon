package evaldata

import "testing"

func TestDatasetIsReadableAndStable(t *testing.T) {
	cases, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 3 {
		t.Fatalf("dataset contains %d cases, want 3", len(cases))
	}
	seen := make(map[string]struct{}, len(cases))
	for _, testCase := range cases {
		if testCase.ID == "" || testCase.Kind == "" || len(testCase.Expected) == 0 {
			t.Fatalf("incomplete evaluation case: %+v", testCase)
		}
		if _, exists := seen[testCase.ID]; exists {
			t.Fatalf("duplicate evaluation case id: %s", testCase.ID)
		}
		seen[testCase.ID] = struct{}{}
	}
}
