package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestIsUniqueViolationUsesPostgresSQLState(t *testing.T) {
	wrapped := fmt.Errorf("insert failed: %w", &pq.Error{Code: postgresUniqueViolation, Message: "duplicate key"})
	if !isUniqueViolation(wrapped) {
		t.Fatal("wrapped PostgreSQL unique violation was not classified")
	}

	if isUniqueViolation(&pq.Error{Code: "23514", Message: "duplicate key"}) {
		t.Fatal("non-unique PostgreSQL error was classified as a unique violation")
	}
	if isUniqueViolation(errors.New("duplicate key value violates unique constraint")) {
		t.Fatal("plain error text was classified as a unique violation")
	}
}

func TestPostgresSQLStateUnwrapsDriverErrorWithoutExposingDetails(t *testing.T) {
	err := fmt.Errorf("database failure: %w", &pq.Error{Code: "42601", Message: "private SQL detail"})
	if got := postgresSQLState(err); got != "42601" {
		t.Fatalf("postgresSQLState() = %q, want 42601", got)
	}
	if got := postgresSQLState(fmt.Errorf("plain failure")); got != "" {
		t.Fatalf("postgresSQLState(plain error) = %q, want empty", got)
	}
}
