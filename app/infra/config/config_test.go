package config

import "testing"

func TestEscapeYAMLDoubleQuoted(t *testing.T) {
	input := "value with \"quotes\", \\slashes\\, and\nline"
	want := "value with \\\"quotes\\\", \\\\slashes\\\\, and\\nline"
	if got := escapeYAMLDoubleQuoted(input); got != want {
		t.Fatalf("escapeYAMLDoubleQuoted() = %q, want %q", got, want)
	}
}
