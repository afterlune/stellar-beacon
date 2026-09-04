//go:build integration

package migration

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"xorm.io/xorm"
)

func TestRunnerFailureRollsBackMigrationAndRecoveryApplies(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dsn := strings.TrimSpace(os.Getenv("MIGRATION_INTEGRATION_DSN"))
	if dsn == "" {
		if os.Getenv("TESTCONTAINERS_ENABLED") != "1" {
			t.Skip("set MIGRATION_INTEGRATION_DSN or TESTCONTAINERS_ENABLED=1 to run the isolated PostgreSQL migration recovery test")
		}
		dsn = startMigrationPostgresContainer(t, ctx)
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open isolated PostgreSQL: %v", err)
	}
	defer database.Close()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("ping isolated PostgreSQL: %v", err)
	}

	schema := fmt.Sprintf("migration_probe_%d", time.Now().UnixNano())
	if _, err := database.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create probe schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	})

	scopedDSN, err := withMigrationSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("scope migration DSN: %v", err)
	}
	engine, err := xorm.NewEngine("postgres", scopedDSN)
	if err != nil {
		t.Fatalf("open scoped migration engine: %v", err)
	}
	defer engine.Close()

	brokenMigrations := fstest.MapFS{
		"migrations/0001_probe.sql": &fstest.MapFile{Data: []byte(`
CREATE TABLE migration_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL);
`)},
		"migrations/0002_failure_probe.sql": &fstest.MapFile{Data: []byte(`
CREATE TABLE migration_failure_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO migration_failure_probe (id, value) VALUES (1, 'must roll back');
SELECT 1 / 0;
`)},
	}

	if err := NewRunnerWithFS(engine, brokenMigrations).Up(ctx); err == nil {
		t.Fatal("broken migration unexpectedly succeeded")
	}

	failedPlan, err := NewRunnerWithFS(engine, brokenMigrations).Status(ctx)
	if err != nil {
		t.Fatalf("inspect failed migration state: %v", err)
	}
	assertMigrationPlan(t, failedPlan, 1, 1, 0)
	assertMigrationStatus(t, failedPlan, 1, MigrationPlanApplied)
	assertMigrationStatus(t, failedPlan, 2, MigrationPlanPending)

	scopedDatabase, err := sql.Open("postgres", scopedDSN)
	if err != nil {
		t.Fatalf("open scoped verification database: %v", err)
	}
	defer scopedDatabase.Close()
	if err := scopedDatabase.PingContext(ctx); err != nil {
		t.Fatalf("ping scoped verification database: %v", err)
	}
	var failedTable *string
	if err := scopedDatabase.QueryRowContext(ctx, "SELECT to_regclass('migration_failure_probe')").Scan(&failedTable); err != nil {
		t.Fatalf("inspect rolled-back table: %v", err)
	}
	if failedTable != nil {
		t.Fatalf("failed migration table still exists: %q", *failedTable)
	}

	recoveredMigrations := fstest.MapFS{
		"migrations/0001_probe.sql": &fstest.MapFile{Data: []byte(`
CREATE TABLE migration_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL);
`)},
		"migrations/0002_failure_probe.sql": &fstest.MapFile{Data: []byte(`
CREATE TABLE migration_failure_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO migration_failure_probe (id, value) VALUES (1, 'recovered');
`)},
	}
	if err := NewRunnerWithFS(engine, recoveredMigrations).Up(ctx); err != nil {
		t.Fatalf("apply repaired migration: %v", err)
	}

	recoveredPlan, err := NewRunnerWithFS(engine, recoveredMigrations).Status(ctx)
	if err != nil {
		t.Fatalf("inspect recovered migration state: %v", err)
	}
	assertMigrationPlan(t, recoveredPlan, 2, 0, 0)
	assertMigrationStatus(t, recoveredPlan, 1, MigrationPlanApplied)
	assertMigrationStatus(t, recoveredPlan, 2, MigrationPlanApplied)

	var recoveredValue string
	if err := scopedDatabase.QueryRowContext(ctx, "SELECT value FROM migration_failure_probe WHERE id = 1").Scan(&recoveredValue); err != nil {
		t.Fatalf("read recovered migration row: %v", err)
	}
	if recoveredValue != "recovered" {
		t.Fatalf("recovered migration value = %q, want recovered", recoveredValue)
	}
}

func startMigrationPostgresContainer(t *testing.T, ctx context.Context) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16.15-bookworm@sha256:bb3e1a57e5407e0a5280b4211980a5e537f4abd234a87014ac979849a78dd825",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "integration",
				"POSTGRES_DB":       "integration",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start isolated migration PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		terminateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(terminateCtx); err != nil {
			t.Errorf("terminate isolated migration PostgreSQL: %v", err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("resolve isolated migration PostgreSQL host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("resolve isolated migration PostgreSQL port: %v", err)
	}
	return fmt.Sprintf("postgres://postgres:integration@%s:%s/integration?sslmode=disable", host, port.Port())
}

func withMigrationSearchPath(dsn, schema string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("options", "-c search_path="+schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func assertMigrationPlan(t *testing.T, plan MigrationPlan, applied, pending, drifted int) {
	t.Helper()
	if !plan.SchemaMigrationsExists || plan.Applied != applied || plan.Pending != pending || plan.Drifted != drifted {
		t.Fatalf("migration plan = %+v, want exists=true applied=%d pending=%d drifted=%d", plan, applied, pending, drifted)
	}
}

func assertMigrationStatus(t *testing.T, plan MigrationPlan, version int64, want MigrationPlanEntryStatus) {
	t.Helper()
	for _, entry := range plan.Entries {
		if entry.Version == version {
			if entry.Status != want {
				t.Fatalf("migration %d status = %q, want %q", version, entry.Status, want)
			}
			return
		}
	}
	t.Fatalf("migration %d missing from plan: %+v", version, plan)
}
