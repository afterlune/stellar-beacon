package shared

import (
	"log/slog"
	"regexp"
)

func CheckEmail(check string) bool {
	rule := "\\w[-\\w.+]*@([A-Za-z0-9][-A-Za-z0-9]+\\.)+[A-Za-z]{2,14}"
	match, err := regexp.MatchString(rule, check)
	if err != nil {
		slog.Error("validate email pattern failed", "error", err)
	}
	return match
}
