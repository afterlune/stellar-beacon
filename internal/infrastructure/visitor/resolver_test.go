package visitor

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestClientIPPrefersRealIPHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Real-IP", "198.51.100.20")
	req.Header.Set("X-Forwarded-For", "198.51.100.30")

	if got := ClientIP(context.Background(), req); got != "198.51.100.20" {
		t.Fatalf("ClientIP() = %q, want X-Real-IP value", got)
	}
}

func TestClientIPSkipsUnknownHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test/", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	req.Header.Set("X-Real-IP", "unknown")
	req.Header.Set("X-Forwarded-For", "198.51.100.30")

	if got := ClientIP(context.Background(), req); got != "198.51.100.30" {
		t.Fatalf("ClientIP() = %q, want X-Forwarded-For value", got)
	}
}
