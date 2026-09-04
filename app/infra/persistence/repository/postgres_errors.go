package repository

import (
	"errors"

	"github.com/lib/pq"
)

const postgresUniqueViolation = "23505"

func postgresSQLState(err error) string {
	if err == nil {
		return ""
	}
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr == nil {
		return ""
	}
	return string(pqErr.Code)
}

// isUniqueViolation classifies PostgreSQL uniqueness failures by SQLSTATE,
// not by the localized/driver-specific error text. Repositories can then
// expose a stable conflict without misclassifying an unrelated database
// failure that happens to mention a duplicate value.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr != nil && string(pqErr.Code) == postgresUniqueViolation
}
