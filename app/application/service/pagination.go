package service

import (
	"strconv"
	"strings"
)

// parsePageQuery keeps the established repository defaults for omitted
// pagination values, but does not silently turn malformed input into a valid
// query. Numeric values are left for pgsql.Page to normalize as before.
func parsePageQuery(currentValue, sizeValue string, defaultCurrent, defaultSize int) (int, int, bool) {
	current, ok := parsePageValue(currentValue, defaultCurrent)
	if !ok {
		return 0, 0, false
	}
	size, ok := parsePageValue(sizeValue, defaultSize)
	if !ok {
		return 0, 0, false
	}
	return current, size, true
}

func parsePageValue(value string, fallback int) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed, true
}
