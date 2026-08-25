package pgsql

import "strings"

const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// ContainsPattern builds a safe, case-sensitive contains pattern for a
// parameterized LIKE expression. User supplied LIKE wildcards are treated as
// literal characters to avoid accidental full-table scans.
func ContainsPattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\`+`\`)
	value = strings.ReplaceAll(value, "%", `\%`)
	value = strings.ReplaceAll(value, "_", `\_`)
	return "%" + value + "%"
}

// Page normalizes untrusted pagination values before they reach the database.
func Page(current, size int) (limit, offset int) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return size, (current - 1) * size
}
