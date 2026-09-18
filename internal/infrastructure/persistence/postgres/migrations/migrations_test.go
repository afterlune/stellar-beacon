package migrations

import "testing"

func TestNormalizeLegacyCron(t *testing.T) {
	tests := map[string]string{
		"0 0/10 * * * ?": "0/10 * * * *",
		"0 0 3 * * ?":    "0 3 * * *",
		"*/30 * * * *":   "*/30 * * * *",
	}
	for input, want := range tests {
		if got := normalizeLegacyCron(input); got != want {
			t.Fatalf("normalizeLegacyCron(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidStandardCron(t *testing.T) {
	if !validStandardCron("*/30 * * * *") {
		t.Fatal("standard five-field expression should be valid")
	}
	for _, expression := range []string{"0 0/10 * * * ?", "0 3 * * * *", "not a cron"} {
		if validStandardCron(expression) {
			t.Fatalf("legacy or malformed expression %q should be invalid", expression)
		}
	}
}
