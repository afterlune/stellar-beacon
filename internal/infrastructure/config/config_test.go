package config

import "testing"

func TestEscapeYAMLDoubleQuoted(t *testing.T) {
	input := "value with \"quotes\", \\slashes\\, and\nline"
	want := "value with \\\"quotes\\\", \\\\slashes\\\\, and\\nline"
	if got := escapeYAMLDoubleQuoted(input); got != want {
		t.Fatalf("escapeYAMLDoubleQuoted() = %q, want %q", got, want)
	}
}

func TestConfiguredValuePrefersRenamedEnvironmentVariable(t *testing.T) {
	t.Setenv("STELLAR_BEACON_ENV", "integration")
	t.Setenv("BENETNASCH_ENV", "dev")

	if got := configuredValue("STELLAR_BEACON_ENV", "BENETNASCH_ENV"); got != "integration" {
		t.Fatalf("configuredValue() = %q, want renamed value", got)
	}
}

func TestConfiguredValueFallsBackToFormerEnvironmentVariable(t *testing.T) {
	t.Setenv("STELLAR_BEACON_CONFIG_DIR", "")
	t.Setenv("BENETNASCH_CONFIG_DIR", t.TempDir())

	got := configuredValue("STELLAR_BEACON_CONFIG_DIR", "BENETNASCH_CONFIG_DIR")
	if got == "" {
		t.Fatal("configuredValue() did not return the former environment variable")
	}
}

func TestFindDirectoryAcceptsFormerEnvironmentPrefix(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("STELLAR_BEACON_CONFIG_DIR", "")
	t.Setenv("BENETNASCH_CONFIG_DIR", directory)

	got, ok := findDirectory("STELLAR_BEACON_CONFIG_DIR", "unused-default")
	if !ok || got != directory {
		t.Fatalf("findDirectory() = (%q, %t), want (%q, true)", got, ok, directory)
	}
}
