package port

import (
	"context"
	"time"
)

const (
	MonitorStatusHealthy       = "healthy"
	MonitorStatusDegraded      = "degraded"
	MonitorStatusUnhealthy     = "unhealthy"
	MonitorStatusUnknown       = "unknown"
	MonitorStatusStale         = "stale"
	MonitorStatusNotConfigured = "not_configured"
)

type MonitorComponent struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Required  bool      `json:"required"`
	LatencyMs float64   `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
	Message   string    `json:"message"`
}

type MonitorResources struct {
	UptimeSeconds  float64 `json:"uptimeSeconds"`
	CPUPercent     float64 `json:"cpuPercent"`
	HeapAllocBytes uint64  `json:"heapAllocBytes"`
	HeapSysBytes   uint64  `json:"heapSysBytes"`
	Goroutines     int     `json:"goroutines"`
	GCTotal        uint32  `json:"gcTotal"`
}

type MonitorHTTPMetrics struct {
	Requests       uint64   `json:"requests"`
	Status2xx      uint64   `json:"status2xx"`
	Status3xx      uint64   `json:"status3xx"`
	Status4xx      uint64   `json:"status4xx"`
	Status5xx      uint64   `json:"status5xx"`
	AvgLatencyMs   float64  `json:"avgLatencyMs"`
	P95LatencyMs   float64  `json:"p95LatencyMs"`
	SampleSeconds  float64  `json:"sampleSeconds"`
	LatencyBuckets []uint64 `json:"-"`
}

type MonitorWorker struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Queued    int64     `json:"queued"`
	Capacity  int64     `json:"capacity"`
	Failed    int64     `json:"failed"`
	Running   int       `json:"running"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type MonitorInstance struct {
	InstanceID string             `json:"instanceId"`
	ReplicaID  string             `json:"replicaId"`
	Status     string             `json:"status"`
	CapturedAt time.Time          `json:"capturedAt"`
	Components []MonitorComponent `json:"components"`
	Resources  MonitorResources   `json:"resources"`
	HTTP       MonitorHTTPMetrics `json:"http"`
	Workers    []MonitorWorker    `json:"workers"`
}

type MonitorSnapshot struct {
	Status           string            `json:"status"`
	GeneratedAt      time.Time         `json:"generatedAt"`
	HistoryAvailable bool              `json:"historyAvailable"`
	HistoryMessage   string            `json:"historyMessage,omitempty"`
	Instances        []MonitorInstance `json:"instances"`
}

type MonitorSample struct {
	MonitorInstance
}

type MonitorStatusEntry struct {
	Scope     string  `json:"scope"`
	Status    string  `json:"status"`
	Message   string  `json:"message"`
	LatencyMs float64 `json:"latencyMs"`
}

type MonitorStatusSample struct {
	ReplicaID  string               `json:"replicaId"`
	InstanceID string               `json:"instanceId"`
	CapturedAt time.Time            `json:"capturedAt"`
	States     []MonitorStatusEntry `json:"states"`
}

type MonitorStatusPeriod struct {
	ReplicaID  string     `json:"replicaId"`
	InstanceID string     `json:"instanceId"`
	Scope      string     `json:"scope"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
	Message    string     `json:"message"`
	LatencyMs  float64    `json:"latencyMs"`
	IncidentID int64      `json:"incidentId,omitempty"`
}

type MonitorTimeline struct {
	Range     string                `json:"range"`
	From      time.Time             `json:"from"`
	To        time.Time             `json:"to"`
	Available bool                  `json:"available"`
	Message   string                `json:"message,omitempty"`
	Periods   []MonitorStatusPeriod `json:"periods"`
}

type MonitorIncidentUpdate struct {
	ID         int64     `json:"id"`
	IncidentID int64     `json:"incidentId"`
	AuthorID   int       `json:"authorId"`
	AuthorName string    `json:"authorName"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
}

type MonitorIncident struct {
	ID         int64                   `json:"id"`
	ReplicaID  string                  `json:"replicaId"`
	InstanceID string                  `json:"instanceId"`
	Scope      string                  `json:"scope"`
	Severity   string                  `json:"severity"`
	State      string                  `json:"state"`
	StartedAt  time.Time               `json:"startedAt"`
	ResolvedAt *time.Time              `json:"resolvedAt,omitempty"`
	LastSeenAt time.Time               `json:"lastSeenAt"`
	Message    string                  `json:"message"`
	Updates    []MonitorIncidentUpdate `json:"updates"`
}

type MonitorTrendPoint struct {
	Period         time.Time `json:"period"`
	Requests       uint64    `json:"requests"`
	Redirects      uint64    `json:"redirects"`
	ClientErrors   uint64    `json:"clientErrors"`
	ServerErrors   uint64    `json:"serverErrors"`
	AvgLatencyMs   float64   `json:"avgLatencyMs"`
	P95LatencyMs   float64   `json:"p95LatencyMs"`
	CPUPercent     float64   `json:"cpuPercent"`
	HeapAllocBytes uint64    `json:"heapAllocBytes"`
}

type MonitorTrends struct {
	Range     string              `json:"range"`
	Available bool                `json:"available"`
	Message   string              `json:"message,omitempty"`
	Points    []MonitorTrendPoint `json:"points"`
}

type SystemMonitorRepository interface {
	Save(context.Context, MonitorSample) error
	Latest(context.Context) ([]MonitorSample, error)
	ListSince(context.Context, time.Time) ([]MonitorSample, error)
	DeleteBefore(context.Context, time.Time) error
	ListStatusSamples(context.Context, time.Time, time.Time) ([]MonitorStatusSample, error)
	ListIncidents(context.Context, time.Time, int) ([]MonitorIncident, error)
	GetIncident(context.Context, int64) (MonitorIncident, error)
	AddIncidentUpdate(context.Context, int64, int, string, string) (MonitorIncidentUpdate, error)
	DeleteStatusBefore(context.Context, time.Time) error
	DeleteIncidentsBefore(context.Context, time.Time) error
}

type SystemMonitor interface {
	Snapshot(context.Context) MonitorSnapshot
	Trends(context.Context, string) (MonitorTrends, error)
	Timeline(context.Context, string) (MonitorTimeline, error)
	Incidents(context.Context, string, int) ([]MonitorIncident, error)
	Incident(context.Context, int64) (MonitorIncident, error)
	AddIncidentUpdate(context.Context, int64, int, string, string) (MonitorIncidentUpdate, error)
}

// ObjectStorageHealthChecker is an optional read-only connectivity probe.
// Storage adapters must not create buckets or write objects during a check.
type ObjectStorageHealthChecker interface {
	CheckHealth(context.Context) error
}
