package cmd

import "testing"

func TestRequireMigrationWriteAuthorization(t *testing.T) {
	if err := requireMigrationWriteAuthorization(false); err == nil {
		t.Fatal("expected migration write authorization error")
	}
	if err := requireMigrationWriteAuthorization(true); err != nil {
		t.Fatalf("authorized migration returned error: %v", err)
	}
}
