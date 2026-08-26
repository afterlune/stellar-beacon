package middlewares

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestIsAdminPathUsesSegmentBoundary(t *testing.T) {
	tests := []struct {
		path  string
		admin bool
	}{
		{path: "/admin", admin: true},
		{path: "/admin/roles", admin: true},
		{path: "/administrator", admin: false},
		{path: "/administer", admin: false},
	}
	for _, tt := range tests {
		if got := isAdminPath(tt.path); got != tt.admin {
			t.Errorf("isAdminPath(%q) = %v, want %v", tt.path, got, tt.admin)
		}
	}
}

func TestRequestLogPayloadOmitsMultipartBytes(t *testing.T) {
	body := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")

	got := requestLogPayload(req, body)
	if !utf8.ValidString(got) {
		t.Fatal("request log payload must be valid UTF-8")
	}
	if !strings.Contains(got, "request body omitted") || !strings.Contains(got, "multipart/form-data") {
		t.Fatalf("unexpected sanitized payload: %q", got)
	}
}

func TestTruncateLogTextPreservesUTF8(t *testing.T) {
	got := truncateLogText(strings.Repeat("中", 1000), 2000)
	if !utf8.ValidString(got) {
		t.Fatal("truncated log payload must be valid UTF-8")
	}
	if len(got) > 2000 || !strings.HasSuffix(got, "...[truncated]") {
		t.Fatalf("unexpected truncated payload length or suffix: %d %q", len(got), got[len(got)-minInt(len(got), 20):])
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
