package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

type MySystemMonitorRepository struct{ engine *xorm.Engine }

func NewSystemMonitorRepository(engine *xorm.Engine) *MySystemMonitorRepository {
	return &MySystemMonitorRepository{engine: engine}
}

type monitorSampleRow struct {
	InstanceID         string    `xorm:"instance_id"`
	ReplicaID          string    `xorm:"replica_id"`
	CapturedAt         time.Time `xorm:"captured_at"`
	OverallStatus      string    `xorm:"overall_status"`
	ComponentsJSON     string    `xorm:"components_json"`
	ResourcesJSON      string    `xorm:"resources_json"`
	WorkersJSON        string    `xorm:"workers_json"`
	Requests           uint64    `xorm:"http_requests"`
	Status2xx          uint64    `xorm:"http_2xx"`
	Status3xx          uint64    `xorm:"http_3xx"`
	Status4xx          uint64    `xorm:"http_4xx"`
	Status5xx          uint64    `xorm:"http_5xx"`
	AvgLatencyMs       float64   `xorm:"http_avg_latency_ms"`
	P95LatencyMs       float64   `xorm:"http_p95_latency_ms"`
	SampleSeconds      float64   `xorm:"http_sample_seconds"`
	LatencyBucketsJSON string    `xorm:"http_latency_buckets"`
}

type monitorStatusSampleRow struct {
	ReplicaID  string    `xorm:"replica_id"`
	InstanceID string    `xorm:"instance_id"`
	CapturedAt time.Time `xorm:"captured_at"`
	StatesJSON string    `xorm:"states_json"`
}

type monitorIncidentRow struct {
	ID         int64      `xorm:"id"`
	ReplicaID  string     `xorm:"replica_id"`
	InstanceID string     `xorm:"instance_id"`
	Scope      string     `xorm:"scope"`
	Severity   string     `xorm:"severity"`
	State      string     `xorm:"state"`
	StartedAt  time.Time  `xorm:"started_at"`
	ResolvedAt *time.Time `xorm:"resolved_at"`
	LastSeenAt time.Time  `xorm:"last_seen_at"`
	Message    string     `xorm:"message"`
}

type monitorIncidentUpdateRow struct {
	ID         int64     `xorm:"id"`
	IncidentID int64     `xorm:"incident_id"`
	AuthorID   int       `xorm:"author_id"`
	AuthorName string    `xorm:"author_name"`
	Content    string    `xorm:"content"`
	CreatedAt  time.Time `xorm:"created_at"`
}

func (r *MySystemMonitorRepository) Save(ctx context.Context, sample port.MonitorSample) error {
	if r == nil || r.engine == nil {
		return errors.Unavailable("system_monitor.save.database", nil)
	}
	components, err := json.Marshal(sample.Components)
	if err != nil {
		return fmt.Errorf("encode system monitor components: %w", err)
	}
	resources, err := json.Marshal(sample.Resources)
	if err != nil {
		return fmt.Errorf("encode system monitor resources: %w", err)
	}
	workers, err := json.Marshal(sample.Workers)
	if err != nil {
		return fmt.Errorf("encode system monitor workers: %w", err)
	}
	buckets, err := json.Marshal(sample.HTTP.LatencyBuckets)
	if err != nil {
		return fmt.Errorf("encode system monitor latency histogram: %w", err)
	}
	session, err := repoTransactionSession(r.engine, ctx, "system_monitor.save")
	if err != nil {
		return err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return repoCtxErr(ctx, "system_monitor.save.begin", errors.Unavailable("system_monitor.save", err))
	}
	defer session.Rollback()
	_, err = session.Exec(`
		INSERT INTO t_system_monitor_sample (
			instance_id, replica_id, captured_at, overall_status, components_json, resources_json, workers_json,
			http_requests, http_2xx, http_3xx, http_4xx, http_5xx, http_avg_latency_ms,
			http_p95_latency_ms, http_sample_seconds, http_latency_buckets
		) VALUES (?, ?, ?, ?, ?::jsonb, ?::jsonb, ?::jsonb, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb)
		ON CONFLICT (instance_id, captured_at) DO UPDATE SET
			replica_id = EXCLUDED.replica_id,
			overall_status = EXCLUDED.overall_status,
			components_json = EXCLUDED.components_json,
			resources_json = EXCLUDED.resources_json,
			workers_json = EXCLUDED.workers_json,
			http_requests = EXCLUDED.http_requests,
			http_2xx = EXCLUDED.http_2xx,
			http_3xx = EXCLUDED.http_3xx,
			http_4xx = EXCLUDED.http_4xx,
			http_5xx = EXCLUDED.http_5xx,
			http_avg_latency_ms = EXCLUDED.http_avg_latency_ms,
			http_p95_latency_ms = EXCLUDED.http_p95_latency_ms,
			http_sample_seconds = EXCLUDED.http_sample_seconds,
			http_latency_buckets = EXCLUDED.http_latency_buckets
	`, sample.InstanceID, monitorReplicaID(sample), sample.CapturedAt, sample.Status, string(components), string(resources), string(workers),
		sample.HTTP.Requests, sample.HTTP.Status2xx, sample.HTTP.Status3xx, sample.HTTP.Status4xx, sample.HTTP.Status5xx,
		sample.HTTP.AvgLatencyMs, sample.HTTP.P95LatencyMs, sample.HTTP.SampleSeconds, string(buckets))
	if err != nil {
		return repoCtxErr(ctx, "system_monitor.save", errors.Unavailable("system_monitor.save", err))
	}
	states := monitorStatusEntries(sample.MonitorInstance)
	statesJSON, err := json.Marshal(states)
	if err != nil {
		return fmt.Errorf("encode system monitor status states: %w", err)
	}
	replicaID := monitorReplicaID(sample)
	if err := reconcileMonitorIncidents(session, sample, states); err != nil {
		return repoCtxErr(ctx, "system_monitor.save.incidents", errors.Unavailable("system_monitor.save", err))
	}
	if _, err := session.Exec(`
		INSERT INTO t_system_monitor_status_sample (replica_id, instance_id, captured_at, states_json)
		VALUES (?, ?, ?, ?::jsonb)
		ON CONFLICT (instance_id, captured_at) DO UPDATE SET
			replica_id = EXCLUDED.replica_id,
			states_json = EXCLUDED.states_json
	`, replicaID, sample.InstanceID, sample.CapturedAt, string(statesJSON)); err != nil {
		return repoCtxErr(ctx, "system_monitor.save.status", errors.Unavailable("system_monitor.save", err))
	}
	if err := session.Commit(); err != nil {
		return repoCtxErr(ctx, "system_monitor.save.commit", errors.Unavailable("system_monitor.save", err))
	}
	return nil
}

func (r *MySystemMonitorRepository) Latest(ctx context.Context) ([]port.MonitorSample, error) {
	if r == nil || r.engine == nil {
		return nil, errors.Unavailable("system_monitor.latest.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.latest")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []monitorSampleRow
	err = session.SQL(`
		SELECT DISTINCT ON (instance_id)
			instance_id, replica_id, captured_at, overall_status,
			components_json::text AS components_json, resources_json::text AS resources_json,
			workers_json::text AS workers_json, http_requests, http_2xx, http_3xx, http_4xx, http_5xx,
			http_avg_latency_ms, http_p95_latency_ms, http_sample_seconds,
			http_latency_buckets::text AS http_latency_buckets
		FROM t_system_monitor_sample
		WHERE captured_at >= CURRENT_TIMESTAMP - INTERVAL '10 minutes'
		ORDER BY instance_id, captured_at DESC
	`).Find(&rows)
	if err != nil {
		return nil, repoCtxErr(ctx, "system_monitor.latest", errors.Unavailable("system_monitor.latest", err))
	}
	return decodeMonitorRows(rows)
}

func (r *MySystemMonitorRepository) ListSince(ctx context.Context, since time.Time) ([]port.MonitorSample, error) {
	if r == nil || r.engine == nil {
		return nil, errors.Unavailable("system_monitor.list_since.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.list_since")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []monitorSampleRow
	err = session.SQL(`
		SELECT
			instance_id, replica_id, captured_at, overall_status,
			components_json::text AS components_json, resources_json::text AS resources_json,
			workers_json::text AS workers_json, http_requests, http_2xx, http_3xx, http_4xx, http_5xx,
			http_avg_latency_ms, http_p95_latency_ms, http_sample_seconds,
			http_latency_buckets::text AS http_latency_buckets
		FROM t_system_monitor_sample
		WHERE captured_at >= ?
		ORDER BY captured_at ASC, instance_id ASC
	`, since).Find(&rows)
	if err != nil {
		return nil, repoCtxErr(ctx, "system_monitor.list_since", errors.Unavailable("system_monitor.list_since", err))
	}
	return decodeMonitorRows(rows)
}

func (r *MySystemMonitorRepository) ListStatusSamples(ctx context.Context, from, to time.Time) ([]port.MonitorStatusSample, error) {
	if r == nil || r.engine == nil {
		return nil, errors.Unavailable("system_monitor.status_samples.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.status_samples")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var priorRows []monitorStatusSampleRow
	err = session.SQL(`
		SELECT DISTINCT ON (replica_id)
			replica_id, instance_id, captured_at, states_json::text AS states_json
		FROM t_system_monitor_status_sample
		WHERE captured_at < ?
		ORDER BY replica_id, captured_at DESC
	`, from).Find(&priorRows)
	if err != nil {
		return nil, repoCtxErr(ctx, "system_monitor.status_samples.prior", errors.Unavailable("system_monitor.status_samples", err))
	}
	var rangeRows []monitorStatusSampleRow
	err = session.SQL(`
		SELECT replica_id, instance_id, captured_at, states_json::text AS states_json
		FROM t_system_monitor_status_sample
		WHERE captured_at >= ? AND captured_at <= ?
		ORDER BY replica_id, captured_at ASC
	`, from, to).Find(&rangeRows)
	if err != nil {
		return nil, repoCtxErr(ctx, "system_monitor.status_samples.range", errors.Unavailable("system_monitor.status_samples", err))
	}
	rows := append(priorRows, rangeRows...)
	result := make([]port.MonitorStatusSample, 0, len(rows))
	for _, row := range rows {
		sample := port.MonitorStatusSample{
			ReplicaID: row.ReplicaID, InstanceID: row.InstanceID, CapturedAt: row.CapturedAt,
			States: []port.MonitorStatusEntry{},
		}
		if err := json.Unmarshal([]byte(row.StatesJSON), &sample.States); err != nil {
			return nil, fmt.Errorf("decode system monitor status states: %w", err)
		}
		result = append(result, sample)
	}
	return result, nil
}

func (r *MySystemMonitorRepository) ListIncidents(ctx context.Context, since time.Time, limit int) ([]port.MonitorIncident, error) {
	if r == nil || r.engine == nil {
		return nil, errors.Unavailable("system_monitor.incidents.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.incidents")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []monitorIncidentRow
	err = session.SQL(`
		SELECT id, replica_id, instance_id, scope, severity, state,
			started_at, resolved_at, last_seen_at, message
		FROM t_system_monitor_incident
		WHERE started_at >= ? OR state = 'open'
		ORDER BY started_at DESC
		LIMIT ?
	`, since, limit).Find(&rows)
	if err != nil {
		return nil, repoCtxErr(ctx, "system_monitor.incidents", errors.Unavailable("system_monitor.incidents", err))
	}
	result := make([]port.MonitorIncident, 0, len(rows))
	for _, row := range rows {
		result = append(result, monitorIncidentFromRow(row))
	}
	return result, nil
}

func (r *MySystemMonitorRepository) GetIncident(ctx context.Context, id int64) (port.MonitorIncident, error) {
	if r == nil || r.engine == nil {
		return port.MonitorIncident{}, errors.Unavailable("system_monitor.incident.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.incident")
	if err != nil {
		return port.MonitorIncident{}, err
	}
	defer session.Close()
	var row monitorIncidentRow
	found, err := session.SQL(`
		SELECT id, replica_id, instance_id, scope, severity, state,
			started_at, resolved_at, last_seen_at, message
		FROM t_system_monitor_incident WHERE id = ?
	`, id).Get(&row)
	if err != nil {
		return port.MonitorIncident{}, repoCtxErr(ctx, "system_monitor.incident", errors.Unavailable("system_monitor.incident", err))
	}
	if !found {
		return port.MonitorIncident{}, errors.NotFound("system_monitor.incident")
	}
	incident := monitorIncidentFromRow(row)
	var updates []monitorIncidentUpdateRow
	if err := session.SQL(`
		SELECT id, incident_id, author_id, author_name, content, created_at
		FROM t_system_monitor_incident_update
		WHERE incident_id = ? ORDER BY created_at ASC, id ASC
	`, id).Find(&updates); err != nil {
		return port.MonitorIncident{}, repoCtxErr(ctx, "system_monitor.incident.updates", errors.Unavailable("system_monitor.incident", err))
	}
	incident.Updates = make([]port.MonitorIncidentUpdate, 0, len(updates))
	for _, update := range updates {
		incident.Updates = append(incident.Updates, port.MonitorIncidentUpdate{
			ID: update.ID, IncidentID: update.IncidentID, AuthorID: update.AuthorID,
			AuthorName: update.AuthorName, Content: update.Content, CreatedAt: update.CreatedAt,
		})
	}
	return incident, nil
}

func (r *MySystemMonitorRepository) AddIncidentUpdate(ctx context.Context, incidentID int64, authorID int, authorName, content string) (port.MonitorIncidentUpdate, error) {
	if r == nil || r.engine == nil {
		return port.MonitorIncidentUpdate{}, errors.Unavailable("system_monitor.incident_update.database", nil)
	}
	session, err := repoTransactionSession(r.engine, ctx, "system_monitor.incident_update")
	if err != nil {
		return port.MonitorIncidentUpdate{}, err
	}
	defer session.Close()
	if err := session.Begin(); err != nil {
		return port.MonitorIncidentUpdate{}, repoCtxErr(ctx, "system_monitor.incident_update.begin", errors.Unavailable("system_monitor.incident_update", err))
	}
	defer session.Rollback()
	var incidentIDFound int64
	found, err := session.SQL("SELECT id FROM t_system_monitor_incident WHERE id = ? FOR UPDATE", incidentID).Get(&incidentIDFound)
	if err != nil {
		return port.MonitorIncidentUpdate{}, repoCtxErr(ctx, "system_monitor.incident_update.incident", errors.Unavailable("system_monitor.incident_update", err))
	}
	if !found {
		return port.MonitorIncidentUpdate{}, errors.NotFound("system_monitor.incident_update")
	}
	var row monitorIncidentUpdateRow
	found, err = session.SQL(`
		INSERT INTO t_system_monitor_incident_update (incident_id, author_id, author_name, content)
		VALUES (?, ?, ?, ?) RETURNING id, incident_id, author_id, author_name, content, created_at
	`, incidentID, authorID, authorName, content).Get(&row)
	if err != nil {
		return port.MonitorIncidentUpdate{}, repoCtxErr(ctx, "system_monitor.incident_update.insert", errors.Unavailable("system_monitor.incident_update", err))
	}
	if !found {
		return port.MonitorIncidentUpdate{}, errors.Unavailable("system_monitor.incident_update.insert", nil)
	}
	if err := session.Commit(); err != nil {
		return port.MonitorIncidentUpdate{}, repoCtxErr(ctx, "system_monitor.incident_update.commit", errors.Unavailable("system_monitor.incident_update", err))
	}
	return port.MonitorIncidentUpdate{
		ID: row.ID, IncidentID: row.IncidentID, AuthorID: row.AuthorID,
		AuthorName: row.AuthorName, Content: row.Content, CreatedAt: row.CreatedAt,
	}, nil
}

func (r *MySystemMonitorRepository) DeleteBefore(ctx context.Context, cutoff time.Time) error {
	if r == nil || r.engine == nil {
		return errors.Unavailable("system_monitor.delete_before.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.delete_before")
	if err != nil {
		return err
	}
	defer session.Close()
	if _, err := session.Exec(`DELETE FROM t_system_monitor_sample WHERE captured_at < ?`, cutoff); err != nil {
		return repoCtxErr(ctx, "system_monitor.delete_before", errors.Unavailable("system_monitor.delete_before", err))
	}
	return nil
}

func (r *MySystemMonitorRepository) DeleteStatusBefore(ctx context.Context, cutoff time.Time) error {
	if r == nil || r.engine == nil {
		return errors.Unavailable("system_monitor.delete_status_before.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.delete_status_before")
	if err != nil {
		return err
	}
	defer session.Close()
	if _, err := session.Exec(`DELETE FROM t_system_monitor_status_sample WHERE captured_at < ?`, cutoff); err != nil {
		return repoCtxErr(ctx, "system_monitor.delete_status_before", errors.Unavailable("system_monitor.delete_status_before", err))
	}
	return nil
}

func (r *MySystemMonitorRepository) DeleteIncidentsBefore(ctx context.Context, cutoff time.Time) error {
	if r == nil || r.engine == nil {
		return errors.Unavailable("system_monitor.delete_incidents_before.database", nil)
	}
	session, err := repoSession(r.engine, ctx, "system_monitor.delete_incidents_before")
	if err != nil {
		return err
	}
	defer session.Close()
	if _, err := session.Exec(`DELETE FROM t_system_monitor_incident WHERE state = 'resolved' AND started_at < ?`, cutoff); err != nil {
		return repoCtxErr(ctx, "system_monitor.delete_incidents_before", errors.Unavailable("system_monitor.delete_incidents_before", err))
	}
	return nil
}

func monitorReplicaID(sample port.MonitorSample) string {
	if sample.ReplicaID != "" {
		return sample.ReplicaID
	}
	return sample.InstanceID
}

func monitorStatusEntries(instance port.MonitorInstance) []port.MonitorStatusEntry {
	entries := make([]port.MonitorStatusEntry, 0, len(instance.Components)+len(instance.Workers)+1)
	entries = append(entries, port.MonitorStatusEntry{Scope: "system", Status: instance.Status})
	for _, component := range instance.Components {
		entries = append(entries, port.MonitorStatusEntry{
			Scope: component.Name, Status: component.Status,
			Message: component.Message, LatencyMs: component.LatencyMs,
		})
	}
	for _, worker := range instance.Workers {
		entries = append(entries, port.MonitorStatusEntry{
			Scope: "worker:" + worker.Name, Status: worker.Status,
		})
	}
	return entries
}

func reconcileMonitorIncidents(session *xorm.Session, sample port.MonitorSample, states []port.MonitorStatusEntry) error {
	replicaID := monitorReplicaID(sample)
	for _, state := range states {
		if state.Scope == "" {
			continue
		}
		severity := ""
		switch state.Status {
		case port.MonitorStatusDegraded:
			severity = "warning"
		case port.MonitorStatusUnhealthy:
			severity = "outage"
		}
		var active struct {
			ID       int64  `xorm:"id"`
			Severity string `xorm:"severity"`
		}
		found, err := session.SQL(`
			SELECT id, severity FROM t_system_monitor_incident
			WHERE replica_id = ? AND scope = ? AND state = 'open'
			ORDER BY started_at DESC LIMIT 1 FOR UPDATE
		`, replicaID, state.Scope).Get(&active)
		if err != nil {
			return err
		}
		if severity != "" {
			if found {
				if severity == "outage" && active.Severity != "outage" {
					if _, err := session.Exec(`
						UPDATE t_system_monitor_incident SET severity = 'outage', instance_id = ?, last_seen_at = ?, message = ?
						WHERE id = ?
					`, sample.InstanceID, sample.CapturedAt, state.Message, active.ID); err != nil {
						return err
					}
				} else if _, err := session.Exec(`
					UPDATE t_system_monitor_incident SET instance_id = ?, last_seen_at = ?, message = ?
					WHERE id = ?
				`, sample.InstanceID, sample.CapturedAt, state.Message, active.ID); err != nil {
					return err
				}
			} else if _, err := session.Exec(`
				INSERT INTO t_system_monitor_incident (
					replica_id, instance_id, scope, severity, state, started_at, last_seen_at, message
				) VALUES (?, ?, ?, ?, 'open', ?, ?, ?)
			`, replicaID, sample.InstanceID, state.Scope, severity, sample.CapturedAt, sample.CapturedAt, state.Message); err != nil {
				return err
			}
			continue
		}
		if found && (state.Status == port.MonitorStatusHealthy || state.Status == port.MonitorStatusNotConfigured) {
			if _, err := session.Exec(`
				UPDATE t_system_monitor_incident
				SET state = 'resolved', resolved_at = ?, last_seen_at = ?, instance_id = ?
				WHERE id = ?
			`, sample.CapturedAt, sample.CapturedAt, sample.InstanceID, active.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func monitorIncidentFromRow(row monitorIncidentRow) port.MonitorIncident {
	return port.MonitorIncident{
		ID: row.ID, ReplicaID: row.ReplicaID, InstanceID: row.InstanceID, Scope: row.Scope,
		Severity: row.Severity, State: row.State, StartedAt: row.StartedAt,
		ResolvedAt: row.ResolvedAt, LastSeenAt: row.LastSeenAt, Message: row.Message,
		Updates: []port.MonitorIncidentUpdate{},
	}
}

func decodeMonitorRows(rows []monitorSampleRow) ([]port.MonitorSample, error) {
	result := make([]port.MonitorSample, 0, len(rows))
	for _, row := range rows {
		sample := port.MonitorSample{MonitorInstance: port.MonitorInstance{
			InstanceID: row.InstanceID, ReplicaID: row.ReplicaID, Status: row.OverallStatus, CapturedAt: row.CapturedAt,
			HTTP: port.MonitorHTTPMetrics{
				Requests: row.Requests, Status2xx: row.Status2xx, Status3xx: row.Status3xx, Status4xx: row.Status4xx, Status5xx: row.Status5xx,
				AvgLatencyMs: row.AvgLatencyMs, P95LatencyMs: row.P95LatencyMs, SampleSeconds: row.SampleSeconds,
			},
		}}
		if err := json.Unmarshal([]byte(row.ComponentsJSON), &sample.Components); err != nil {
			return nil, fmt.Errorf("decode system monitor components: %w", err)
		}
		if err := json.Unmarshal([]byte(row.ResourcesJSON), &sample.Resources); err != nil {
			return nil, fmt.Errorf("decode system monitor resources: %w", err)
		}
		if err := json.Unmarshal([]byte(row.WorkersJSON), &sample.Workers); err != nil {
			return nil, fmt.Errorf("decode system monitor workers: %w", err)
		}
		if err := json.Unmarshal([]byte(row.LatencyBucketsJSON), &sample.HTTP.LatencyBuckets); err != nil {
			return nil, fmt.Errorf("decode system monitor latency histogram: %w", err)
		}
		result = append(result, sample)
	}
	return result, nil
}

var _ port.SystemMonitorRepository = (*MySystemMonitorRepository)(nil)
