package shared

import (
	"benetnasch/app/infra/config"
	"net/http/httptest"
	"testing"
)

func TestGetIpAddressIgnoresForwardedHeadersWithoutTrustedProxy(t *testing.T) {
	t.Setenv(config.TrustedProxiesEnv, "")
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "198.51.100.20:443"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	req.Header.Set("X-Forwarded-For", "203.0.113.8")

	if got := GetIpAddress(req); got != "198.51.100.20" {
		t.Fatalf("GetIpAddress() = %q, want socket peer", got)
	}
}

func TestGetIpAddressUsesForwardedHeadersFromTrustedProxy(t *testing.T) {
	t.Setenv(config.TrustedProxiesEnv, "10.0.0.0/8")
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")

	if got := GetIpAddress(req); got != "198.51.100.20" {
		t.Fatalf("GetIpAddress() = %q, want forwarded client IP", got)
	}
}
