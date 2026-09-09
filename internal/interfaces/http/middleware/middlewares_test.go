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
		{path: "/v1/admin", admin: true},
		{path: "/v1/admin/roles", admin: true},
		{path: "/admin", admin: false},
		{path: "/v1/administrator", admin: false},
		{path: "/v1/administer", admin: false},
	}
	for _, tt := range tests {
		if got := isAdminPath(tt.path); got != tt.admin {
			t.Errorf("isAdminPath(%q) = %v, want %v", tt.path, got, tt.admin)
		}
	}
}

func TestRequiresAuthentication(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/v1/admin/articles", want: true},
		{path: "/v1/auth/logout", want: true},
		{path: "/v1/auth/me", want: true},
		{path: "/v1/auth/me/avatar", want: true},
		{path: "/v1/auth/login", want: false},
		{path: "/v1/auth/password", want: false},
		{path: "/v1/public/articles", want: false},
	}
	for _, tt := range tests {
		if got := requiresAuthentication(tt.path); got != tt.want {
			t.Errorf("requiresAuthentication(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSwaggerPathMatchesParameters(t *testing.T) {
	tests := []struct {
		template string
		request  string
		want     bool
	}{
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/21", want: true},
		{template: "/v1/admin/albums/{albumId}/photos", request: "/v1/admin/albums/21/photos", want: true},
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/21/photos", want: false},
		{template: "/v1/admin/albums/{albumId}", request: "/v1/admin/albums/", want: false},
		{template: "/v1/admin/albums", request: "/v1/admin/albums/21", want: false},
	}
	for _, tt := range tests {
		if got := swaggerPathMatches(tt.template, tt.request); got != tt.want {
			t.Errorf("swaggerPathMatches(%q, %q) = %v, want %v", tt.template, tt.request, got, tt.want)
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
