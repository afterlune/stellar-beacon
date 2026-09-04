package visitor

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"

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
	ip := config.ResolveClientIP(req)
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
		slog.WarnContext(ctx, "resolve visitor region failed", "error_code", apperrors.SafeCode(err))
	} else {
		identity.Region = region
	}
	return identity, nil
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
