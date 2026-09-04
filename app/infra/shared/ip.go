package shared

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"io"
	"log/slog"
	"net/http"
)

func GetIpAddress(req *http.Request) (s string) {
	return config.ResolveClientIP(req)
}

func GetIpSource(ip string) string {
	dbpath := "resource/ip/ip2region.xdb"
	file, err := xdb.LoadContentFromFile(dbpath)
	if err != nil {
		slog.Error("load IP region database failed", "error_code", apperrors.SafeCode(err))
		return ""
	}
	searcher, err := xdb.NewWithBuffer(file)
	if err != nil {
		slog.Error("initialize IP region searcher failed", "error_code", apperrors.SafeCode(err))
		return ""
	}
	defer searcher.Close()
	region, err := searcher.SearchByStr(ip)
	if err != nil && err != io.EOF {
		slog.Error("search IP region failed", "error_code", apperrors.SafeCode(err))
		return ""
	}
	return region
}
