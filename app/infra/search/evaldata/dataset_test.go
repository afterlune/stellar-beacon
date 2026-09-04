package evaldata

import "testing"

func TestDatasetIsReadableAndStable(t *testing.T) {
	cases, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 8 {
		t.Fatalf("dataset contains %d cases, want 8", len(cases))
	}
	wantIDs := []string{
		"search-zh-001",
		"search-zh-002",
		"search-zh-003",
		"search-zh-004",
		"search-zh-005",
		"search-zh-006",
		"search-zh-noresult-001",
		"search-zh-noresult-002",
	}
	for index, testCase := range cases {
		if testCase.ID != wantIDs[index] {
			t.Fatalf("case %d ID = %q, want %q", index, testCase.ID, wantIDs[index])
		}
		if testCase.Query == "" || testCase.Mode == "" {
			t.Fatalf("incomplete evaluation case: %+v", testCase)
		}
	}
}

func TestNormalizeCasesRejectsInvalidDataset(t *testing.T) {
	tests := []struct {
		name  string
		cases []Case
	}{
		{
			name: "empty",
		},
		{
			name:  "no positive case",
			cases: []Case{{ID: "no-result", Query: "nothing", ExpectNoResult: true}},
		},
		{
			name:  "no no-result case",
			cases: []Case{{ID: "positive", Query: "something", Expected: []string{"doc-1"}}},
		},
		{
			name: "conflicting annotation",
			cases: []Case{
				{ID: "positive", Query: "something", Expected: []string{"doc-1"}},
				{ID: "conflict", Query: "nothing", Expected: []string{"doc-2"}, ExpectNoResult: true},
				{ID: "no-result", Query: "nothing else", ExpectNoResult: true},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := Validate(test.cases); err == nil {
				t.Fatal("Validate() returned nil")
			}
		})
	}
}

func TestNormalizeCaseTrimsAndDeduplicatesExpectedIDs(t *testing.T) {
	canonical, err := (Case{
		ID:       " case-1 ",
		Query:    " query ",
		Expected: []string{" doc-1 ", "doc-1", "doc-2"},
	}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if canonical.ID != "case-1" || canonical.Query != "query" || canonical.Mode != "keyword" {
		t.Fatalf("canonical case = %+v", canonical)
	}
	if len(canonical.Expected) != 2 || canonical.Expected[0] != "doc-1" || canonical.Expected[1] != "doc-2" {
		t.Fatalf("canonical expected IDs = %#v", canonical.Expected)
	}
}
