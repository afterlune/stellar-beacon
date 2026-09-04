package logging

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWritesOnlyErrorsToRotatingJSONFile(t *testing.T) {
	previous := slog.Default()
	defer slog.SetDefault(previous)
	path := filepath.Join(t.TempDir(), "nested", "error.log")
	closeLogging, err := Init(Config{
		ConsoleLevel: slog.LevelError,
		FilePath:     path,
		MaxSize:      1,
		MaxBackups:   1,
		MaxAge:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	slog.Info("info should stay on console")
	slog.Error("database operation failed", "password", "secret", "operation", "list")
	if err := closeLogging(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "database operation failed") {
		t.Fatalf("error log missing from file: %s", content)
	}
	if strings.Contains(content, "info should stay on console") {
		t.Fatalf("info log unexpectedly written to error file: %s", content)
	}
	if strings.Contains(content, "secret") {
		t.Fatalf("sensitive field was not redacted: %s", content)
	}

	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("file output is not JSON: %v", err)
	}
	if record["level"] != "ERROR" {
		t.Fatalf("unexpected log level: %#v", record["level"])
	}
}

func TestInitRejectsEmptyFilePath(t *testing.T) {
	if _, err := Init(Config{}); err == nil {
		t.Fatal("expected empty file path to be rejected")
	}
}

func TestRedactRemovesPromptAndCredentialValues(t *testing.T) {
	if got := Redact("a visitor prompt"); got != RedactedValue {
		t.Fatalf("Redact() = %q, want %q", got, RedactedValue)
	}
	if got := redactText("provider password=secret token=abc"); strings.Contains(got, "secret") || strings.Contains(got, "abc") {
		t.Fatalf("credential values were not redacted: %s", got)
	}
}
