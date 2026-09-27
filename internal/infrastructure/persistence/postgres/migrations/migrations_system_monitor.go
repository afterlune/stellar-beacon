package migrations

import (
	"context"
	"fmt"

	"xorm.io/xorm"
)

// applySystemMonitorSchema stores per-instance minute aggregates for the
// admin health dashboard. The retention policy is enforced by the collector.
func applySystemMonitorSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 32)").Get(&applied); err != nil {
		return fmt.Errorf("check system monitor migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin system monitor migration: %w", err)
	}
	defer session.Rollback()
	if _, err := session.Exec(`
		CREATE TABLE IF NOT EXISTS t_system_monitor_sample (
			instance_id VARCHAR(120) NOT NULL,
			captured_at TIMESTAMPTZ NOT NULL,
			overall_status VARCHAR(24) NOT NULL,
			components_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			resources_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			workers_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			http_requests BIGINT NOT NULL DEFAULT 0,
			http_2xx BIGINT NOT NULL DEFAULT 0,
			http_3xx BIGINT NOT NULL DEFAULT 0,
			http_4xx BIGINT NOT NULL DEFAULT 0,
			http_5xx BIGINT NOT NULL DEFAULT 0,
			http_avg_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			http_p95_latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			http_sample_seconds DOUBLE PRECISION NOT NULL DEFAULT 0,
			http_latency_buckets JSONB NOT NULL DEFAULT '[0,0,0,0,0,0,0]'::jsonb,
			PRIMARY KEY (instance_id, captured_at)
		)`); err != nil {
		return fmt.Errorf("create system monitor sample table: %w", err)
	}
	if _, err := session.Exec(`CREATE INDEX IF NOT EXISTS idx_system_monitor_sample_captured_at ON t_system_monitor_sample (captured_at DESC)`); err != nil {
		return fmt.Errorf("create system monitor sample index: %w", err)
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 32, "system-monitor-samples"); err != nil {
		return fmt.Errorf("record system monitor migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit system monitor migration: %w", err)
	}
	return nil
}

// applySystemMonitorHistorySchema adds compact long-retention status samples
// and operator-authored incident updates without extending metric retention.
func applySystemMonitorHistorySchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 33)").Get(&applied); err != nil {
		return fmt.Errorf("check system monitor history migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin system monitor history migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`ALTER TABLE t_system_monitor_sample ADD COLUMN IF NOT EXISTS replica_id VARCHAR(120)`,
		`UPDATE t_system_monitor_sample SET replica_id = regexp_replace(instance_id, '-[a-z0-9]+$', '') WHERE replica_id IS NULL`,
		`CREATE TABLE IF NOT EXISTS t_system_monitor_status_sample (
			replica_id VARCHAR(120) NOT NULL,
			instance_id VARCHAR(120) NOT NULL,
			captured_at TIMESTAMPTZ NOT NULL,
			states_json JSONB NOT NULL DEFAULT '[]'::jsonb,
			PRIMARY KEY (instance_id, captured_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_system_monitor_status_replica_time ON t_system_monitor_status_sample (replica_id, captured_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_system_monitor_status_captured_at ON t_system_monitor_status_sample (captured_at DESC)`,
		`CREATE TABLE IF NOT EXISTS t_system_monitor_incident (
			id BIGSERIAL PRIMARY KEY,
			replica_id VARCHAR(120) NOT NULL,
			instance_id VARCHAR(120) NOT NULL,
			scope VARCHAR(120) NOT NULL,
			severity VARCHAR(16) NOT NULL,
			state VARCHAR(16) NOT NULL DEFAULT 'open',
			started_at TIMESTAMPTZ NOT NULL,
			resolved_at TIMESTAMPTZ NULL,
			last_seen_at TIMESTAMPTZ NOT NULL,
			message TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_system_monitor_incident_open ON t_system_monitor_incident (replica_id, scope) WHERE state = 'open'`,
		`CREATE INDEX IF NOT EXISTS idx_system_monitor_incident_started_at ON t_system_monitor_incident (started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS t_system_monitor_incident_update (
			id BIGSERIAL PRIMARY KEY,
			incident_id BIGINT NOT NULL REFERENCES t_system_monitor_incident(id) ON DELETE CASCADE,
			author_id INTEGER NOT NULL,
			author_name VARCHAR(120) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_system_monitor_incident_update_incident ON t_system_monitor_incident_update (incident_id, created_at ASC)`,
		`INSERT INTO t_system_monitor_status_sample (replica_id, instance_id, captured_at, states_json)
		 SELECT COALESCE(replica_id, regexp_replace(instance_id, '-[a-z0-9]+$', '')), instance_id, captured_at,
			jsonb_build_array(jsonb_build_object('scope', 'system', 'status', overall_status, 'message', '', 'latencyMs', 0))
			|| COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'scope', component.value->>'name',
					'status', component.value->>'status',
					'message', COALESCE(component.value->>'message', ''),
					'latencyMs', COALESCE(NULLIF(component.value->>'latencyMs', '')::DOUBLE PRECISION, 0)
				))
				FROM jsonb_array_elements(components_json) AS component(value)
			), '[]'::jsonb)
			|| COALESCE((
				SELECT jsonb_agg(jsonb_build_object(
					'scope', 'worker:' || (worker.value->>'name'),
					'status', worker.value->>'status',
					'message', '',
					'latencyMs', 0
				))
				FROM jsonb_array_elements(workers_json) AS worker(value)
			), '[]'::jsonb)
		 FROM t_system_monitor_sample
		 ON CONFLICT (instance_id, captured_at) DO NOTHING`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply system monitor history schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 33, "system-monitor-status-history"); err != nil {
		return fmt.Errorf("record system monitor history migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit system monitor history migration: %w", err)
	}
	return nil
}
