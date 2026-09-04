package service

import "testing"

func TestParsePageQueryDefaultsOnlyWhenValuesAreMissing(t *testing.T) {
	tests := []struct {
		name                  string
		current, size         string
		wantCurrent, wantSize int
		wantOK                bool
	}{
		{name: "defaults", wantCurrent: 1, wantSize: 10, wantOK: true},
		{name: "trimmed values", current: " 2 ", size: " 25 ", wantCurrent: 2, wantSize: 25, wantOK: true},
		{name: "malformed current", current: "oops", size: "10", wantOK: false},
		{name: "malformed size", current: "1", size: "oops", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current, size, ok := parsePageQuery(test.current, test.size, 1, 10)
			if current != test.wantCurrent || size != test.wantSize || ok != test.wantOK {
				t.Fatalf("parsePageQuery(%q, %q) = (%d, %d, %v), want (%d, %d, %v)", test.current, test.size, current, size, ok, test.wantCurrent, test.wantSize, test.wantOK)
			}
		})
	}
}
