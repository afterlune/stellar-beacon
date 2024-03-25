package shared

import (
	"benetnasch/app/infrastructure/zlog"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"io"
	"net"
	"net/http"
	"strings"
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
				zlog.Error(err.Error())
			}
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			ipAddress = strings.Split(localAddr.String(), ":")[0]
		}
	}
	return ipAddress
}

func GetIpSource(ip string) string {
	dbpath := "resource/ip/ip2region.xdb"
	file, err := xdb.LoadContentFromFile(dbpath)
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}
		return ""
	}
	searcher, err := xdb.NewWithBuffer(file)
	if err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}
		return ""
	}
	defer searcher.Close()
	region, err := searcher.SearchByStr(ip)
	if err != nil && err != io.EOF {
		if err != nil {
			zlog.Error(err.Error())
		}
		return ""
	}
	return region
}
