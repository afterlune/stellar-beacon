package shared

import (
	"github.com/mssola/user_agent"
	"net/http"
)

func GetRedisId(req *http.Request) string {
	ipAdress := GetIpAddress(req)
	ua := user_agent.New(req.Header.Get("User-Agent"))
	name, version := ua.Browser()
	uuid := ipAdress + name + version + ua.OS()
	return uuid
}

func GetBrowser(req *http.Request) (name, version string) {
	ua := user_agent.New(req.Header.Get("User-Agent"))
	return ua.Browser()
}

func GetOS(req *http.Request) (os string) {
	ua := user_agent.New(req.Header.Get("User-Agent"))
	return ua.OS()
}

func IsBot(req *http.Request) bool {
	ua := user_agent.New(req.Header.Get("User-Agent"))
	return ua.Bot()
}
