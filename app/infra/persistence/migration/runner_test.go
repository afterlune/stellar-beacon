package migration

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadMigrationsOrdersFilesAndCalculatesChecksums(t *testing.T) {
	migrations, err := loadMigrations(fstest.MapFS{
		"migrations/0002_second.sql": &fstest.MapFile{Data: []byte("SELECT 2;")},
		"migrations/0001_first.sql":  &fstest.MapFile{Data: []byte("SELECT 1;")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != 2 || migrations[0].Version != 1 || migrations[1].Version != 2 {
		t.Fatalf("migrations were not ordered: %+v", migrations)
	}
	if migrations[0].Checksum == "" || migrations[0].Checksum == migrations[1].Checksum {
		t.Fatalf("migration checksums are not distinct: %+v", migrations)
	}
	if migrations[0].SQL != "SELECT 1;" {
		t.Fatalf("migration SQL was not normalized: %q", migrations[0].SQL)
	}
}

func TestMigrationStatusLooksUpCurrentSchema(t *testing.T) {
	if !strings.Contains(migrationTableExistsSQL, "table_schema = current_schema()") {
		t.Fatalf("migration status must follow the connection search_path: %s", migrationTableExistsSQL)
	}
	if strings.Contains(migrationTableExistsSQL, "table_schema = 'public'") {
		t.Fatalf("migration status must not be hard-coded to public: %s", migrationTableExistsSQL)
	}
}

func TestEmbeddedMigrationsIncludeDurableAIJobSchema(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, migration := range migrations {
		if migration.Version == 1 {
			found = true
			if migration.Name != "ai_job" || !strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_ai_job") || !strings.Contains(migration.SQL, "idempotency_key") {
				t.Fatalf("unexpected AI job migration: %+v", migration)
			}
		}
	}
	if !found {
		t.Fatal("embedded AI job migration was not found")
	}
}

func TestEmbeddedMigrationsIncludeBackfillControlSchema(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version == 2 {
			if migration.Name != "ai_index_backfill" ||
				!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_ai_index_backfill") ||
				!strings.Contains(migration.SQL, "pause_requested") ||
				!strings.Contains(migration.SQL, "processed_articles") {
				t.Fatalf("unexpected backfill migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("embedded backfill migration was not found")
}

func TestEmbeddedMigrationsIncludeAIReviewSchema(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version == 3 {
			if migration.Name != "ai_review" ||
				!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_ai_review") ||
				!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_ai_review_action") ||
				!strings.Contains(migration.SQL, "partially_accepted") {
				t.Fatalf("unexpected AI review migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("embedded AI review migration was not found")
}

func TestEmbeddedMigrationsIncludeAgentBehaviorMetadata(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version == 4 {
			if migration.Name != "agent_behavior" ||
				!strings.Contains(migration.SQL, "ADD COLUMN IF NOT EXISTS agent_id") ||
				!strings.Contains(migration.SQL, "idx_t_ai_review_idempotency") {
				t.Fatalf("unexpected agent behavior migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("embedded agent behavior migration was not found")
}

func TestEmbeddedMigrationsIncludeAgentReviewPublication(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, migration := range migrations {
		if migration.Version != 5 {
			continue
		}
		found = true
		if migration.Name != "agent_review_publication" ||
			!strings.Contains(migration.SQL, "publish_status") ||
			!strings.Contains(migration.SQL, "publish_failed") ||
			!strings.Contains(migration.SQL, "published_at") {
			t.Fatalf("unexpected publication migration: %+v", migration)
		}
	}
	if !found {
		t.Fatal("agent review publication migration is missing")
	}
}

func TestEmbeddedMigrationsIncludeAgentActivity(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 6 {
			continue
		}
		if migration.Name != "agent_activity" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_activity") ||
			!strings.Contains(migration.SQL, "emotion") {
			t.Fatalf("unexpected agent activity migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent activity migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentTaskState(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 7 {
			continue
		}
		if migration.Name != "agent_task" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_task_run") ||
			!strings.Contains(migration.SQL, "t_agent_task_effect") ||
			!strings.Contains(migration.SQL, "waiting_approval") {
			t.Fatalf("unexpected agent task migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent task migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentReviewBinding(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 8 {
			continue
		}
		if migration.Name != "agent_review_binding" ||
			!strings.Contains(migration.SQL, "ADD COLUMN IF NOT EXISTS session_id") ||
			!strings.Contains(migration.SQL, "publish_status IN ('not_attempted', 'processing'") {
			t.Fatalf("unexpected agent review binding migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent review binding migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentMemory(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 9 {
			continue
		}
		if migration.Name != "agent_memory" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_memory_assertion") ||
			!strings.Contains(migration.SQL, "subject_key LIKE 'user:%'") ||
			!strings.Contains(migration.SQL, "conflicted") {
			t.Fatalf("unexpected agent memory migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent memory migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentMemoryHistoryAndConflicts(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Name != "agent_memory_history_conflicts" {
			continue
		}
		for _, fragment := range []string{
			"CREATE TABLE IF NOT EXISTS t_agent_memory_assertion_revision",
			"CREATE TABLE IF NOT EXISTS t_agent_memory_conflict",
			"CREATE TABLE IF NOT EXISTS t_agent_memory_conflict_member",
			"uq_t_agent_memory_open_conflict",
		} {
			if !strings.Contains(migration.SQL, fragment) {
				t.Fatalf("memory history/conflict migration is missing %q: %+v", fragment, migration)
			}
		}
		return
	}
	t.Fatal("embedded agent memory history/conflict migration is missing")
}

func TestEmbeddedMigrationsIncludeContentProjection(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 10 {
			continue
		}
		if migration.Name != "content_projection" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_content_projection") ||
			!strings.Contains(migration.SQL, "pca_input") ||
			!strings.Contains(migration.SQL, "is_deleted = TRUE") {
			t.Fatalf("unexpected content projection migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded content projection migration is missing")
}

func TestEmbeddedMigrationsIncludeTimeCapsule(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 12 {
			continue
		}
		if migration.Name != "time_capsule" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_time_capsule") ||
			!strings.Contains(migration.SQL, "owner_user_id INTEGER NOT NULL REFERENCES t_user_info") ||
			!strings.Contains(migration.SQL, "status IN ('draft', 'sealed', 'due', 'delivered', 'delivery_failed')") {
			t.Fatalf("unexpected time capsule migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded time capsule migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentVideo(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 13 {
			continue
		}
		if migration.Name != "agent_video" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_video") ||
			!strings.Contains(migration.SQL, "size_bytes") ||
			!strings.Contains(migration.SQL, "source IN ('local', 'external')") {
			t.Fatalf("unexpected agent video migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent video migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentProfile(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 14 {
			continue
		}
		if migration.Name != "agent_profile" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_profile") ||
			!strings.Contains(migration.SQL, "system_prompt_ref") ||
			!strings.Contains(migration.SQL, "behavior_enabled") {
			t.Fatalf("unexpected agent profile migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent profile migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentAdminRBAC(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 15 {
			continue
		}
		if migration.Name != "agent_admin_rbac" ||
			!strings.Contains(migration.SQL, "'/ai-studio'") ||
			!strings.Contains(migration.SQL, "'/admin/ai/profile'") ||
			!strings.Contains(migration.SQL, "t_role_resource") {
			t.Fatalf("unexpected agent admin RBAC migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent admin RBAC migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentReviewPolicy(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 16 {
			continue
		}
		if migration.Name != "agent_review_policy" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_review_policy") ||
			!strings.Contains(migration.SQL, "review_required = TRUE") ||
			!strings.Contains(migration.SQL, "'/admin/ai/review-policy'") {
			t.Fatalf("unexpected agent review policy migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent review policy migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentMemoryAdminRBAC(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 18 {
			continue
		}
		if migration.Name != "agent_memory_admin_rbac" ||
			!strings.Contains(migration.SQL, "读取 Agent 记忆断言") ||
			!strings.Contains(migration.SQL, "'/admin/ai/memory/assertions/*/history'") ||
			!strings.Contains(migration.SQL, "'/admin/ai/memory/conflicts/*/resolve'") ||
			!strings.Contains(migration.SQL, "t_role_resource") {
			t.Fatalf("unexpected agent memory admin RBAC migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent memory admin RBAC migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentMemoryAdminMenu(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 19 {
			continue
		}
		if migration.Name != "agent_memory_admin_menu" ||
			!strings.Contains(migration.SQL, "'/ai-memory'") ||
			!strings.Contains(migration.SQL, "'/ai/Memory.vue'") ||
			!strings.Contains(migration.SQL, "t_role_menu") {
			t.Fatalf("unexpected agent memory admin menu migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent memory admin menu migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentProviderProbeRBAC(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 20 {
			continue
		}
		if migration.Name != "agent_provider_probe_rbac" ||
			!strings.Contains(migration.SQL, "'/admin/ai/providers/test'") ||
			!strings.Contains(migration.SQL, "t_role_resource") {
			t.Fatalf("unexpected provider probe RBAC migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded provider probe RBAC migration is missing")
}

func TestEmbeddedMigrationsIncludeAgentVisionRBAC(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 21 {
			continue
		}
		if migration.Name != "agent_vision_rbac" ||
			!strings.Contains(migration.SQL, "'/admin/ai/vision/preview'") ||
			!strings.Contains(migration.SQL, "t_role_resource") {
			t.Fatalf("unexpected agent vision RBAC migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded agent vision RBAC migration is missing")
}

func TestEmbeddedMigrationsIncludeAIObservabilityRBAC(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 22 {
			continue
		}
		if migration.Name != "ai_observability_rbac" ||
			!strings.Contains(migration.SQL, "'/admin/ai/observability'") ||
			!strings.Contains(migration.SQL, "t_role_resource") {
			t.Fatalf("unexpected AI observability RBAC migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded AI observability RBAC migration is missing")
}

func TestEmbeddedMigrationsIncludeSpaceCompanionBoundary(t *testing.T) {
	migrations, err := loadMigrations(embeddedMigrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.Version != 23 {
			continue
		}
		if migration.Name != "space_companion" ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_agent_principal") ||
			!strings.Contains(migration.SQL, "VALUES ('moonfei', 'agent'") ||
			!strings.Contains(migration.SQL, "CREATE TABLE IF NOT EXISTS t_space_publication") ||
			!strings.Contains(migration.SQL, "content_type IN ('status', 'dream', 'radio')") {
			t.Fatalf("unexpected space Companion migration: %+v", migration)
		}
		return
	}
	t.Fatal("embedded space Companion migration is missing")
}

func TestLoadMigrationsRejectsInvalidAndDuplicateVersions(t *testing.T) {
	_, err := loadMigrations(fstest.MapFS{
		"migrations/not-a-migration.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	})
	if err == nil {
		t.Fatal("expected invalid filename error")
	}

	_, err = loadMigrations(fstest.MapFS{
		"migrations/0001_first.sql":  &fstest.MapFile{Data: []byte("SELECT 1;")},
		"migrations/0001_second.sql": &fstest.MapFile{Data: []byte("SELECT 2;")},
	})
	if err == nil {
		t.Fatal("expected duplicate version error")
	}
}

func TestValidateAppliedMigrationsRejectsUnknownDatabaseVersion(t *testing.T) {
	migrations := []Migration{{Version: 1, Name: "first"}}
	if err := validateAppliedMigrations(map[int64]appliedMigration{
		1: {name: "first"},
		2: {name: "future"},
	}, migrations); err == nil || !strings.Contains(err.Error(), "migration 2 is not present") {
		t.Fatalf("validateAppliedMigrations() error = %v, want unknown version error", err)
	}
}

func TestValidateAppliedMigrationsAcceptsVersionsInBinary(t *testing.T) {
	if err := validateAppliedMigrations(map[int64]appliedMigration{
		1: {name: "first"},
	}, []Migration{{Version: 1, Name: "first"}}); err != nil {
		t.Fatalf("validateAppliedMigrations() error = %v", err)
	}
}

func TestBuildMigrationPlanReportsPendingAppliedAndDrift(t *testing.T) {
	migrations := []Migration{
		{Version: 1, Name: "first", Checksum: "checksum-1"},
		{Version: 2, Name: "second", Checksum: "checksum-2"},
		{Version: 3, Name: "third", Checksum: "checksum-3"},
	}
	plan, err := buildMigrationPlan(true, migrations, map[int64]appliedMigration{
		1: {name: "first", checksum: "checksum-1"},
		2: {name: "second", checksum: "old-checksum"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.SchemaMigrationsExists || plan.Applied != 1 || plan.Pending != 1 || plan.Drifted != 1 || len(plan.Entries) != 3 {
		t.Fatalf("unexpected migration plan: %+v", plan)
	}
	if plan.Entries[0].Status != MigrationPlanApplied || plan.Entries[1].Status != MigrationPlanDrifted || plan.Entries[2].Status != MigrationPlanPending {
		t.Fatalf("unexpected migration entry statuses: %+v", plan.Entries)
	}
	if plan.Entries[1].AppliedChecksum != "old-checksum" {
		t.Fatalf("drifted checksum was not retained: %+v", plan.Entries[1])
	}
}

func TestBuildMigrationPlanTreatsMissingTableAsAllPending(t *testing.T) {
	plan, err := buildMigrationPlan(false, []Migration{{Version: 1, Name: "first", Checksum: "checksum-1"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SchemaMigrationsExists || plan.Applied != 0 || plan.Pending != 1 || plan.Drifted != 0 || plan.Entries[0].Status != MigrationPlanPending {
		t.Fatalf("unexpected missing-table migration plan: %+v", plan)
	}
}
