// Package migration owns explicit, versioned database migrations. The server
// does not call this package during normal startup; operators invoke the
// migrate command deliberately.
package migration

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"xorm.io/xorm"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

var migrationNamePattern = regexp.MustCompile(`^([0-9]{4,})_([a-z0-9][a-z0-9_]*)\.sql$`)

const createMigrationTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
)`

// migrationTableExistsSQL deliberately resolves the table in PostgreSQL's
// current schema. Up creates the unqualified schema_migrations table through
// the connection's search_path, so a read-only status check must inspect the
// same namespace instead of assuming public (the isolated integration stack
// uses a per-test schema).
const migrationTableExistsSQL = `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = current_schema() AND table_name = 'schema_migrations'
) AS exists`

// migrationAdvisoryLockID serializes migration runners that target the same
// PostgreSQL database. The lock is held on the session for the complete
// inspection/apply cycle and is released explicitly (and automatically by
// PostgreSQL when the session closes).
const migrationAdvisoryLockID int64 = 0x42454e45544e4153

type Migration struct {
	Version  int64
	Name     string
	SQL      string
	Checksum string
}

type MigrationPlanEntryStatus string

const (
	MigrationPlanPending MigrationPlanEntryStatus = "pending"
	MigrationPlanApplied MigrationPlanEntryStatus = "applied"
	MigrationPlanDrifted MigrationPlanEntryStatus = "drifted"
)

// MigrationPlan is a read-only snapshot used by release preflight tooling.
// Status never creates schema_migrations, acquires an advisory lock, or runs
// migration SQL; it can therefore be used while preparing a separately
// approved migration window.
type MigrationPlan struct {
	SchemaMigrationsExists bool                 `json:"schemaMigrationsExists"`
	Applied                int                  `json:"applied"`
	Pending                int                  `json:"pending"`
	Drifted                int                  `json:"drifted"`
	Entries                []MigrationPlanEntry `json:"entries"`
}

type MigrationPlanEntry struct {
	Version         int64                    `json:"version"`
	Name            string                   `json:"name"`
	Checksum        string                   `json:"checksum"`
	AppliedChecksum string                   `json:"appliedChecksum,omitempty"`
	Status          MigrationPlanEntryStatus `json:"status"`
}

type Runner struct {
	engine     *xorm.Engine
	migrations fs.FS
}

func NewRunner(engine *xorm.Engine) *Runner {
	return &Runner{engine: engine, migrations: embeddedMigrations}
}

// NewRunnerWithFS is useful for parser tests and isolated migration checks.
// Production callers should use NewRunner so the compiled migration set is
// immutable and travels with the application binary.
func NewRunnerWithFS(engine *xorm.Engine, migrations fs.FS) *Runner {
	return &Runner{engine: engine, migrations: migrations}
}

func (r *Runner) Up(ctx context.Context) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.engine == nil {
		return errors.New("database engine is not initialized")
	}
	if r.migrations == nil {
		return errors.New("migration filesystem is not initialized")
	}

	migrations, err := loadMigrations(r.migrations)
	if err != nil {
		return err
	}

	session := r.engine.NewSession().Context(ctx)
	defer session.Close()
	if err := acquireMigrationLock(session); err != nil {
		return err
	}
	defer func() {
		if _, unlockErr := session.QueryString("SELECT pg_advisory_unlock(?)", migrationAdvisoryLockID); unlockErr != nil && err == nil {
			err = fmt.Errorf("release migration lock: %w", unlockErr)
		}
	}()
	if _, err := session.Exec(createMigrationTableSQL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := readApplied(session)
	if err != nil {
		return err
	}
	if err := validateAppliedMigrations(applied, migrations); err != nil {
		return err
	}
	for _, migration := range migrations {
		previous, ok := applied[migration.Version]
		if ok {
			if previous.name != migration.Name || previous.checksum != migration.Checksum {
				return fmt.Errorf("migration %d changed after it was applied", migration.Version)
			}
			continue
		}
		if err := applyMigration(session, migration); err != nil {
			return err
		}
	}
	return nil
}

// Status reads the current migration table without creating it or taking the
// migration advisory lock. The result is intentionally a snapshot: an
// operator must still re-run it immediately before an approved Up operation.
func (r *Runner) Status(ctx context.Context) (plan MigrationPlan, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.engine == nil {
		return plan, errors.New("database engine is not initialized")
	}
	if r.migrations == nil {
		return plan, errors.New("migration filesystem is not initialized")
	}

	migrations, err := loadMigrations(r.migrations)
	if err != nil {
		return plan, err
	}
	session := r.engine.NewSession().Context(ctx)
	defer func() {
		if closeErr := session.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close migration status session: %w", closeErr)
		}
	}()

	tableExists, err := migrationTableExists(session)
	if err != nil {
		return plan, err
	}
	var applied map[int64]appliedMigration
	if tableExists {
		applied, err = readApplied(session)
		if err != nil {
			return plan, err
		}
	}
	return buildMigrationPlan(tableExists, migrations, applied)
}

func migrationTableExists(session *xorm.Session) (bool, error) {
	rows, err := session.QueryString(migrationTableExistsSQL)
	if err != nil {
		return false, fmt.Errorf("inspect schema_migrations: %w", err)
	}
	if len(rows) != 1 {
		return false, errors.New("inspect schema_migrations: unexpected result")
	}
	value := strings.ToLower(strings.TrimSpace(firstRowValue(rows[0], "exists", "EXISTS")))
	switch value {
	case "true", "t", "1":
		return true, nil
	case "false", "f", "0":
		return false, nil
	default:
		return false, fmt.Errorf("inspect schema_migrations: invalid existence value %q", value)
	}
}

func acquireMigrationLock(session *xorm.Session) error {
	if _, err := session.QueryString("SELECT pg_advisory_lock(?)", migrationAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	return nil
}

type appliedMigration struct {
	name     string
	checksum string
}

func readApplied(session *xorm.Session) (map[int64]appliedMigration, error) {
	rows, err := session.QueryString("SELECT version, name, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	applied := make(map[int64]appliedMigration, len(rows))
	for _, row := range rows {
		versionText := row["version"]
		if versionText == "" {
			versionText = row["VERSION"]
		}
		version, err := strconv.ParseInt(versionText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse applied migration version %q: %w", versionText, err)
		}
		applied[version] = appliedMigration{
			name:     firstRowValue(row, "name", "NAME"),
			checksum: firstRowValue(row, "checksum", "CHECKSUM"),
		}
	}
	return applied, nil
}

func firstRowValue(row map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := row[key]; value != "" {
			return value
		}
	}
	return ""
}

func validateAppliedMigrations(applied map[int64]appliedMigration, migrations []Migration) error {
	known := make(map[int64]struct{}, len(migrations))
	for _, migration := range migrations {
		known[migration.Version] = struct{}{}
	}
	for version := range applied {
		if _, ok := known[version]; !ok {
			return fmt.Errorf("database migration %d is not present in this binary", version)
		}
	}
	return nil
}

func buildMigrationPlan(tableExists bool, migrations []Migration, applied map[int64]appliedMigration) (MigrationPlan, error) {
	if applied == nil {
		applied = map[int64]appliedMigration{}
	}
	if err := validateAppliedMigrations(applied, migrations); err != nil {
		return MigrationPlan{}, err
	}

	plan := MigrationPlan{
		SchemaMigrationsExists: tableExists,
		Entries:                make([]MigrationPlanEntry, 0, len(migrations)),
	}
	for _, migration := range migrations {
		entry := MigrationPlanEntry{
			Version:  migration.Version,
			Name:     migration.Name,
			Checksum: migration.Checksum,
			Status:   MigrationPlanPending,
		}
		if tableExists {
			if previous, ok := applied[migration.Version]; ok {
				entry.AppliedChecksum = previous.checksum
				if previous.name != migration.Name || previous.checksum != migration.Checksum {
					entry.Status = MigrationPlanDrifted
					plan.Drifted++
				} else {
					entry.Status = MigrationPlanApplied
					plan.Applied++
				}
			} else {
				plan.Pending++
			}
		} else {
			plan.Pending++
		}
		plan.Entries = append(plan.Entries, entry)
	}
	return plan, nil
}

func applyMigration(session *xorm.Session, migration Migration) (err error) {
	if err = session.Begin(); err != nil {
		return fmt.Errorf("begin migration %d: %w", migration.Version, err)
	}
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := session.Rollback(); rollbackErr != nil && err == nil {
				err = fmt.Errorf("rollback migration %d: %w", migration.Version, rollbackErr)
			}
		}
	}()

	if strings.TrimSpace(migration.SQL) != "" {
		if _, err = session.Exec(migration.SQL); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}
	if _, err = session.Exec(
		"INSERT INTO schema_migrations (version, name, checksum) VALUES (?, ?, ?)",
		migration.Version, migration.Name, migration.Checksum,
	); err != nil {
		return fmt.Errorf("record migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	if err = session.Commit(); err != nil {
		return fmt.Errorf("commit migration %d (%s): %w", migration.Version, migration.Name, err)
	}
	committed = true
	return nil
}

func loadMigrations(migrations fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	loaded := make([]Migration, 0, len(entries))
	seenVersions := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationNamePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename %q", entry.Name())
		}
		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", entry.Name(), err)
		}
		if _, exists := seenVersions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %d", version)
		}
		data, err := fs.ReadFile(migrations, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}
		checksum := sha256.Sum256(data)
		loaded = append(loaded, Migration{
			Version:  version,
			Name:     matches[2],
			SQL:      strings.TrimSpace(string(data)),
			Checksum: hex.EncodeToString(checksum[:]),
		})
		seenVersions[version] = struct{}{}
	}
	sort.Slice(loaded, func(i, j int) bool { return loaded[i].Version < loaded[j].Version })
	return loaded, nil
}
