package config

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizeTrustedProxies(t *testing.T) {
	got, err := normalizeTrustedProxies([]string{
		"10.0.0.1",
		"10.0.0.0/8",
		"10.0.0.0/8",
		" 192.0.2.1, 2001:db8::/32 ",
	})
	if err != nil {
		t.Fatalf("normalizeTrustedProxies() error = %v", err)
	}
	want := []string{"10.0.0.1", "10.0.0.0/8", "192.0.2.1", "2001:db8::/32"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeTrustedProxies() = %#v, want %#v", got, want)
	}
}

func TestNormalizeTrustedProxiesRejectsInvalidAndGlobalNetworks(t *testing.T) {
	for _, value := range []string{"not-an-ip", "0.0.0.0/0", "::/0"} {
		t.Run(value, func(t *testing.T) {
			if _, err := normalizeTrustedProxies([]string{value}); err == nil {
				t.Fatalf("normalizeTrustedProxies(%q) error = nil", value)
			}
		})
	}
}

func TestTrustedProxiesEnvironmentOverridesConfig(t *testing.T) {
	t.Setenv(TrustedProxiesEnv, "192.0.2.1, 198.51.100.0/24")
	got, err := TrustedProxies()
	if err != nil {
		t.Fatalf("TrustedProxies() error = %v", err)
	}
	want := []string{"192.0.2.1", "198.51.100.0/24"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TrustedProxies() = %#v, want %#v", got, want)
	}
}

func TestResolveClientIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "198.51.100.20:443"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	req.Header.Set("X-Real-IP", "203.0.113.8")

	got := resolveClientIP(req, trustedProxyNetworks(nil))
	if got != "198.51.100.20" {
		t.Fatalf("resolveClientIP() = %q, want socket peer", got)
	}
}

func TestResolveClientIPUsesForwardedHeadersOnlyFromTrustedPeer(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 10.0.0.3")

	got := resolveClientIP(req, trustedProxyNetworks([]string{"10.0.0.0/8"}))
	if got != "198.51.100.9" {
		t.Fatalf("resolveClientIP() = %q, want original client IP", got)
	}
}

func TestResolveClientIPStopsAtUntrustedForwardedHop(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.9, 203.0.113.4")

	got := resolveClientIP(req, trustedProxyNetworks([]string{"10.0.0.0/8"}))
	if got != "203.0.113.4" {
		t.Fatalf("resolveClientIP() = %q, want nearest untrusted hop", got)
	}
}

func TestResolveClientIPFallsBackToValidRealIPAfterMalformedForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "10.0.0.2:443"
	req.Header.Set("X-Forwarded-For", "not-an-ip")
	req.Header.Set("X-Real-IP", "198.51.100.9")

	got := resolveClientIP(req, trustedProxyNetworks([]string{"10.0.0.0/8"}))
	if got != "198.51.100.9" {
		t.Fatalf("resolveClientIP() = %q, want X-Real-IP", got)
	}
}

func TestResolveClientIPFailsClosedWhenConfiguredProxyIsInvalid(t *testing.T) {
	t.Setenv(TrustedProxiesEnv, "not-an-ip")
	req := httptest.NewRequest("GET", "http://example.test", nil)
	req.RemoteAddr = "198.51.100.20:443"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")

	got := ResolveClientIP(req)
	if got != "198.51.100.20" {
		t.Fatalf("ResolveClientIP() = %q, want socket peer", got)
	}
}
