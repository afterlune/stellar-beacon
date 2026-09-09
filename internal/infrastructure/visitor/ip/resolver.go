package visitor

import (
	"benetnasch/internal/domain/port"
	"benetnasch/internal/infrastructure/config"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/mssola/user_agent"
)

type Resolver struct{}

func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) Resolve(ctx context.Context, req *http.Request) (port.VisitorIdentity, error) {
	if req == nil {
		return port.VisitorIdentity{}, fmt.Errorf("request is nil")
	}
	ip := clientIP(ctx, req)
	ua := user_agent.New(req.Header.Get("User-Agent"))
	browser, version := ua.Browser()
	os := ua.OS()
	identity := port.VisitorIdentity{
		IP:             ip,
		Browser:        browser,
		BrowserVersion: version,
		OS:             os,
		Fingerprint:    fingerprint(ip, browser, version, os),
		IsBot:          ua.Bot(),
	}
	region, err := regionForIP(ip)
	if err != nil {
		slog.WarnContext(ctx, "resolve visitor region failed", "error", err)
	} else {
		identity.Region = region
	}
	return identity, nil
}

func clientIP(ctx context.Context, req *http.Request) string {
	for _, header := range []string{
		"X-Real-IP",
		"X-Forwarded-For",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
		"HTTP_CLIENT_IP",
		"HTTP_X_FORWARDED_FOR",
	} {
		value := req.Header.Get(header)
		if value != "" && !strings.EqualFold(value, "unknown") {
			return value
		}
	}
	remote := req.RemoteAddr
	if remote != "127.0.0.1" && remote != "0:0:0:0:0:0:0:1" {
		return remote
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", "8.8.8.8:53")
	if err != nil {
		return remote
	}
	defer conn.Close()
	if address, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return address.IP.String()
	}
	return remote
}

func regionForIP(ip string) (string, error) {
	path, err := config.ResourcePath("ip/ip2region.xdb")
	if err != nil {
		return "", err
	}
	content, err := xdb.LoadContentFromFile(path)
	if err != nil {
		return "", err
	}
	searcher, err := xdb.NewWithBuffer(content)
	if err != nil {
		return "", err
	}
	defer searcher.Close()
	region, err := searcher.SearchByStr(ip)
	if err != nil && err != io.EOF {
		return "", err
	}
	return region, nil
}

func fingerprint(ip, browser, version, os string) string {
	sum := md5.Sum([]byte(ip + browser + version + os))
	return hex.EncodeToString(sum[:])
}

var _ port.VisitorResolver = (*Resolver)(nil)
