package shared

import (
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"io"
	"log/slog"
	"net"
	"net/http"
)

func GetIpAddress(req *http.Request) (s string) {
	ipAddress := req.Header.Get("X-Real-IP")
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.Header.Get("x-forwarded-for")
	}
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.Header.Get("Proxy-Client-IP")
	}
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.Header.Get("WL-Proxy-Client-IP")
	}
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.Header.Get("HTTP_CLIENT_IP")
	}
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.Header.Get("HTTP_X_FORWARDED_FOR")
	}
	if ipAddress == "" || len(ipAddress) == 0 || ipAddress == "unknown" {
		ipAddress = req.RemoteAddr
		if ipAddress == "127.0.0.1" || ipAddress == "0:0:0:0:0:0:0:1" {
			//根据网卡取本机配置的IP
			conn, err := net.Dial("udp", "8.8.8.8:53")
			if err != nil {
				slog.Error("resolve local IP failed", "error", err)
			} else {
				defer conn.Close()
				ipAddress = conn.LocalAddr().(*net.UDPAddr).IP.String()
			}
		}
	}
	return ipAddress
}

func GetIpSource(ip string) string {
	dbpath := "resources/ip/ip2region.xdb"
	file, err := xdb.LoadContentFromFile(dbpath)
	if err != nil {
		slog.Error("load IP region database failed", "error", err)
		return ""
	}
	searcher, err := xdb.NewWithBuffer(file)
	if err != nil {
		slog.Error("initialize IP region searcher failed", "error", err)
		return ""
	}
	defer searcher.Close()
	region, err := searcher.SearchByStr(ip)
	if err != nil && err != io.EOF {
		slog.Error("search IP region failed", "error", err)
		return ""
	}
	return region
}
