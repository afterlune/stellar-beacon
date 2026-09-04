package config

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// TrustedProxiesEnv is a comma-separated list of IP addresses or CIDRs for
// the reverse proxies that are allowed to provide client-IP headers. It is
// intentionally opt-in: an empty value makes the application use the socket
// peer address and ignore X-Forwarded-For/X-Real-IP.
const TrustedProxiesEnv = "BENETNASCH_TRUSTED_PROXIES"

// TrustedProxies returns the validated proxy allowlist used by both Gin and
// request helpers outside Gin. The environment variable wins over YAML so a
// deployment can bind the list to its actual ingress network without baking
// it into the image.
func TrustedProxies() ([]string, error) {
	values := viper.GetStringSlice("server.trusted_proxies")
	if raw, ok := os.LookupEnv(TrustedProxiesEnv); ok {
		values = strings.Split(raw, ",")
	}
	return normalizeTrustedProxies(values)
}

func normalizeTrustedProxies(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}

			if ip := net.ParseIP(item); ip != nil {
				item = ip.String()
			} else {
				_, network, err := net.ParseCIDR(item)
				if err != nil {
					return nil, fmt.Errorf("trusted proxy %q is not an IP address or CIDR", item)
				}
				ones, _ := network.Mask.Size()
				if ones == 0 {
					return nil, fmt.Errorf("trusted proxy %q would trust every address", item)
				}
				item = network.String()
			}

			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result, nil
}

// ResolveClientIP returns the client address only when the socket peer is in
// the configured trusted-proxy list. This keeps non-Gin consumers (visitor
// reporting, login throttling and operation logs) from having a second,
// weaker interpretation of forwarding headers.
func ResolveClientIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	proxies, err := TrustedProxies()
	if err != nil {
		// Startup validation reports this configuration error. Runtime callers
		// fail closed as a defense in depth measure if they are used in tests or
		// by a separately assembled process.
		return requestRemoteIP(req.RemoteAddr)
	}
	return resolveClientIP(req, trustedProxyNetworks(proxies))
}

func resolveClientIP(req *http.Request, trustedProxies []*net.IPNet) string {
	remoteIP := requestRemoteIP(req.RemoteAddr)
	if remoteIP == "" {
		return strings.TrimSpace(req.RemoteAddr)
	}
	parsedRemote := net.ParseIP(remoteIP)
	if parsedRemote == nil || !isTrustedProxy(parsedRemote, trustedProxies) {
		return remoteIP
	}

	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.Join(req.Header.Values(header), ",")
		if header == "X-Forwarded-For" {
			if clientIP := clientIPFromForwardedFor(value, trustedProxies); clientIP != "" {
				return clientIP
			}
			continue
		}
		if clientIP := parseForwardedIP(value); clientIP != "" {
			return clientIP
		}
	}
	return remoteIP
}

func requestRemoteIP(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	if ip := net.ParseIP(remoteAddr); ip != nil {
		return ip.String()
	}
	return remoteAddr
}

func trustedProxyNetworks(proxies []string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(proxies))
	for _, proxy := range proxies {
		if ip := net.ParseIP(proxy); ip != nil {
			bits := 128
			if ipv4 := ip.To4(); ipv4 != nil {
				ip = ipv4
				bits = 32
			}
			networks = append(networks, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(proxy)
		if err == nil {
			networks = append(networks, network)
		}
	}
	return networks
}

func isTrustedProxy(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func clientIPFromForwardedFor(value string, trustedProxies []*net.IPNet) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	items := strings.Split(value, ",")
	for index := len(items) - 1; index >= 0; index-- {
		ip := net.ParseIP(strings.TrimSpace(items[index]))
		if ip == nil {
			return ""
		}
		if index == 0 || !isTrustedProxy(ip, trustedProxies) {
			return ip.String()
		}
	}
	return ""
}

func parseForwardedIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}
