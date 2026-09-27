package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"runtime"
	runtimemetrics "runtime/metrics"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

const (
	probeInterval         = 30 * time.Second
	probeTimeout          = 3 * time.Second
	metricsPersistEvery   = time.Minute
	metricsRetention      = 7 * 24 * time.Hour
	statusRetention       = 90 * 24 * time.Hour
	instanceStaleAfter    = 2 * time.Minute
	monitorAPITimeout     = 3 * time.Second
	metricCleanupInterval = time.Hour
	processCPUMetric      = "/cpu/classes/total:cpu-seconds"
)

var processStartedAt = time.Now()

type Probe struct {
	Name     string
	Required bool
	Check    func(context.Context) (status, message string)
}

type WorkerSnapshot func(context.Context) []port.MonitorWorker

type Collector struct {
	repository port.SystemMonitorRepository
	requests   *RequestMetrics
	probes     []Probe
	workers    WorkerSnapshot
	replicaID  string
	instanceID string
	startedAt  time.Time

	mu          sync.RWMutex
	current     port.MonitorInstance
	lastCPU     float64
	lastCPUAt   time.Time
	lastCleanup time.Time
}

func NewCollector(repository port.SystemMonitorRepository, requests *RequestMetrics, probes []Probe, workers WorkerSnapshot) *Collector {
	if requests == nil {
		requests = DefaultRequestMetrics
	}
	startedAt := processStartedAt
	instanceID, _ := os.Hostname()
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		instanceID = "backend"
	}
	replicaID := instanceID + "-" + strconv.Itoa(os.Getpid())
	if len(replicaID) > 120 {
		replicaID = replicaID[len(replicaID)-120:]
	}
	instanceID = replicaID + "-" + strconv.FormatInt(startedAt.UnixNano(), 36)
	if len(instanceID) > 120 {
		instanceID = instanceID[len(instanceID)-120:]
	}
	return &Collector{
		repository: repository,
		requests:   requests,
		probes:     append([]Probe(nil), probes...),
		workers:    workers,
		replicaID:  replicaID,
		instanceID: instanceID,
		startedAt:  startedAt,
		current: port.MonitorInstance{
			InstanceID: instanceID,
			ReplicaID:  replicaID,
			Status:     port.MonitorStatusUnknown,
			CapturedAt: startedAt,
			Components: []port.MonitorComponent{},
			Workers:    []port.MonitorWorker{},
		},
	}
}

func (c *Collector) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.refresh(ctx)
	probeTicker := time.NewTicker(probeInterval)
	defer probeTicker.Stop()
	nextPersist := time.Now().Truncate(metricsPersistEvery).Add(metricsPersistEvery)
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-probeTicker.C:
			c.refresh(ctx)
			if !now.Before(nextPersist) {
				c.persist(ctx, now)
				nextPersist = now.Truncate(metricsPersistEvery).Add(metricsPersistEvery)
			}
		}
	}
}

func (c *Collector) Snapshot(ctx context.Context) port.MonitorSnapshot {
	if ctx == nil {
		ctx = context.Background()
	}
	c.mu.RLock()
	local := cloneMonitorInstance(c.current)
	c.mu.RUnlock()
	response := port.MonitorSnapshot{
		Status: local.Status, GeneratedAt: time.Now(), HistoryAvailable: false,
		Instances: []port.MonitorInstance{local},
	}
	if c.repository == nil {
		response.HistoryMessage = "historyStorageNotConfigured"
		return response
	}
	queryCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	latest, err := c.repository.Latest(queryCtx)
	if err != nil {
		response.HistoryMessage = "historyUnavailable"
		return response
	}
	instances := make(map[string]port.MonitorInstance, len(latest)+1)
	for _, sample := range latest {
		instance := sample.MonitorInstance
		if instance.InstanceID != c.instanceID {
			if time.Since(instance.CapturedAt) > instanceStaleAfter {
				instance.Status = port.MonitorStatusStale
				for index := range instance.Components {
					instance.Components[index].Status = port.MonitorStatusStale
					instance.Components[index].Message = "stale"
				}
				for index := range instance.Workers {
					instance.Workers[index].Status = port.MonitorStatusStale
				}
			}
			instances[instance.InstanceID] = instance
		}
	}
	instances[c.instanceID] = local
	response.Instances = response.Instances[:0]
	for _, instance := range instances {
		response.Instances = append(response.Instances, instance)
	}
	sort.Slice(response.Instances, func(i, j int) bool {
		return response.Instances[i].InstanceID < response.Instances[j].InstanceID
	})
	response.Status = aggregateStatus(response.Instances)
	response.HistoryAvailable = true
	return response
}

func (c *Collector) Trends(ctx context.Context, rangeValue string) (port.MonitorTrends, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	var span, bucket time.Duration
	switch rangeValue {
	case "1h":
		span, bucket = time.Hour, time.Minute
	case "24h":
		span, bucket = 24*time.Hour, 10*time.Minute
	case "7d":
		span, bucket = metricsRetention, time.Hour
	default:
		return port.MonitorTrends{}, fmt.Errorf("unsupported system monitor range")
	}
	result := port.MonitorTrends{Range: rangeValue, Available: false, Points: []port.MonitorTrendPoint{}}
	if c.repository == nil {
		result.Message = "historyStorageNotConfigured"
		return result, nil
	}
	queryCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	samples, err := c.repository.ListSince(queryCtx, now.Add(-span))
	if err != nil {
		result.Message = "historyUnavailable"
		return result, nil
	}
	result.Points = aggregateTrends(samples, now, span, bucket)
	result.Available = len(samples) > 0
	return result, nil
}

func (c *Collector) Timeline(ctx context.Context, rangeValue string) (port.MonitorTimeline, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var span time.Duration
	switch rangeValue {
	case "24h":
		span = 24 * time.Hour
	case "7d":
		span = 7 * 24 * time.Hour
	case "30d":
		span = 30 * 24 * time.Hour
	case "90d":
		span = statusRetention
	default:
		return port.MonitorTimeline{}, fmt.Errorf("unsupported system monitor timeline range")
	}
	now := time.Now()
	result := port.MonitorTimeline{
		Range: rangeValue, From: now.Add(-span), To: now, Periods: []port.MonitorStatusPeriod{},
	}
	if c.repository == nil {
		result.Message = "historyStorageNotConfigured"
		return result, nil
	}
	queryCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	samples, err := c.repository.ListStatusSamples(queryCtx, result.From, result.To)
	if err != nil {
		slog.Warn("system monitor status history could not be loaded", "error", err)
		result.Message = "historyUnavailable"
		return result, nil
	}
	result.Available = len(samples) > 0
	result.Periods = statusPeriods(samples, result.From, result.To)
	if incidents, incidentErr := c.repository.ListIncidents(queryCtx, result.From, 200); incidentErr == nil {
		attachIncidentIDs(result.Periods, incidents)
	} else {
		slog.Warn("system monitor incident links could not be loaded", "error", incidentErr)
	}
	return result, nil
}

func attachIncidentIDs(periods []port.MonitorStatusPeriod, incidents []port.MonitorIncident) {
	for periodIndex := range periods {
		period := &periods[periodIndex]
		if period.Status != port.MonitorStatusDegraded && period.Status != port.MonitorStatusUnhealthy {
			continue
		}
		periodEnd := time.Time{}
		if period.EndedAt != nil {
			periodEnd = *period.EndedAt
		}
		for _, incident := range incidents {
			if incident.ReplicaID != period.ReplicaID || incident.Scope != period.Scope {
				continue
			}
			incidentEnd := incident.LastSeenAt
			if incident.ResolvedAt != nil {
				incidentEnd = *incident.ResolvedAt
			}
			if incident.StartedAt.Before(periodEnd) && incidentEnd.After(period.StartedAt) {
				period.IncidentID = incident.ID
				break
			}
		}
	}
}

func (c *Collector) Incidents(ctx context.Context, rangeValue string, limit int) ([]port.MonitorIncident, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	span, err := monitorRange(rangeValue)
	if err != nil {
		return nil, err
	}
	if c.repository == nil {
		return []port.MonitorIncident{}, nil
	}
	if limit < 1 || limit > 200 {
		limit = 100
	}
	queryCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	return c.repository.ListIncidents(queryCtx, time.Now().Add(-span), limit)
}

func (c *Collector) Incident(ctx context.Context, id int64) (port.MonitorIncident, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.repository == nil {
		return port.MonitorIncident{}, fmt.Errorf("system monitor history is unavailable")
	}
	queryCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	return c.repository.GetIncident(queryCtx, id)
}

func (c *Collector) AddIncidentUpdate(ctx context.Context, id int64, authorID int, authorName, content string) (port.MonitorIncidentUpdate, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c.repository == nil {
		return port.MonitorIncidentUpdate{}, fmt.Errorf("system monitor history is unavailable")
	}
	writeCtx, cancel := context.WithTimeout(ctx, monitorAPITimeout)
	defer cancel()
	return c.repository.AddIncidentUpdate(writeCtx, id, authorID, authorName, content)
}

func (c *Collector) refresh(ctx context.Context) {
	results := make([]port.MonitorComponent, len(c.probes))
	var wg sync.WaitGroup
	for index, probe := range c.probes {
		index, probe := index, probe
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkedAt := time.Now()
			component := port.MonitorComponent{
				Name: probe.Name, Required: probe.Required, Status: port.MonitorStatusUnknown,
				CheckedAt: checkedAt, Message: "checkNotCompleted",
			}
			if probe.Check == nil {
				component.Message = "healthCheckNotConfigured"
				results[index] = component
				return
			}
			probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			started := time.Now()
			status, message := probe.Check(probeCtx)
			component.LatencyMs = float64(time.Since(started)) / float64(time.Millisecond)
			component.CheckedAt = time.Now()
			if status == "" {
				status = port.MonitorStatusUnknown
			}
			component.Status = status
			component.Message = message
			results[index] = component
		}()
	}
	wg.Wait()
	resources := c.sampleResources(time.Now())
	workers := []port.MonitorWorker{}
	if c.workers != nil {
		workers = c.workers(ctx)
	}
	status := overallStatus(results, workers)
	c.mu.Lock()
	c.current.Status = status
	c.current.CapturedAt = time.Now()
	c.current.Components = results
	c.current.Resources = resources
	c.current.Workers = workers
	c.mu.Unlock()
}

func (c *Collector) persist(ctx context.Context, now time.Time) {
	if c.repository == nil {
		return
	}
	httpMetrics := c.requests.Take(now)
	c.mu.Lock()
	c.current.HTTP = httpMetrics
	c.current.CapturedAt = now.Truncate(time.Minute)
	c.current.ReplicaID = c.replicaID
	sample := port.MonitorSample{MonitorInstance: cloneMonitorInstance(c.current)}
	c.mu.Unlock()
	saveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	if err := c.repository.Save(saveCtx, sample); err != nil {
		slog.Warn("system monitor sample could not be persisted", "error", err)
	}
	cancel()
	if c.lastCleanup.IsZero() || now.Sub(c.lastCleanup) >= metricCleanupInterval {
		cleanupCtx, cleanupCancel := context.WithTimeout(ctx, 5*time.Second)
		if err := c.repository.DeleteBefore(cleanupCtx, now.Add(-metricsRetention)); err != nil {
			slog.Warn("system monitor retention cleanup failed")
		} else if err := c.repository.DeleteStatusBefore(cleanupCtx, now.Add(-statusRetention)); err != nil {
			slog.Warn("system monitor status retention cleanup failed")
		} else if err := c.repository.DeleteIncidentsBefore(cleanupCtx, now.Add(-statusRetention)); err != nil {
			slog.Warn("system monitor incident retention cleanup failed")
		} else {
			c.lastCleanup = now
		}
		cleanupCancel()
	}
}

func (c *Collector) sampleResources(now time.Time) port.MonitorResources {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	cpuSeconds := readProcessCPU()
	cpuPercent := 0.0
	c.mu.Lock()
	if c.lastCPUAt.IsZero() {
		c.lastCPUAt = now
		c.lastCPU = cpuSeconds
	} else {
		elapsed := now.Sub(c.lastCPUAt).Seconds()
		if elapsed > 0 && runtime.NumCPU() > 0 {
			cpuPercent = math.Max(0, (cpuSeconds-c.lastCPU)/elapsed/float64(runtime.NumCPU())*100)
		}
		c.lastCPUAt = now
		c.lastCPU = cpuSeconds
	}
	c.mu.Unlock()
	return port.MonitorResources{
		UptimeSeconds: now.Sub(c.startedAt).Seconds(), CPUPercent: cpuPercent,
		HeapAllocBytes: memory.HeapAlloc, HeapSysBytes: memory.HeapSys,
		Goroutines: runtime.NumGoroutine(), GCTotal: memory.NumGC,
	}
}

func readProcessCPU() float64 {
	samples := []runtimemetrics.Sample{{Name: processCPUMetric}}
	runtimemetrics.Read(samples)
	if len(samples) == 0 || samples[0].Value.Kind() != runtimemetrics.KindFloat64 {
		return 0
	}
	return samples[0].Value.Float64()
}

func overallStatus(components []port.MonitorComponent, workers []port.MonitorWorker) string {
	status := port.MonitorStatusHealthy
	for _, component := range components {
		if component.Status == port.MonitorStatusHealthy || component.Status == port.MonitorStatusNotConfigured {
			continue
		}
		if component.Required {
			return port.MonitorStatusUnhealthy
		}
		status = port.MonitorStatusDegraded
	}
	for _, worker := range workers {
		if worker.Status != port.MonitorStatusHealthy {
			status = port.MonitorStatusDegraded
		}
	}
	return status
}

func aggregateStatus(instances []port.MonitorInstance) string {
	status := port.MonitorStatusUnknown
	for _, instance := range instances {
		switch instance.Status {
		case port.MonitorStatusUnhealthy:
			return port.MonitorStatusUnhealthy
		case port.MonitorStatusDegraded:
			status = port.MonitorStatusDegraded
		case port.MonitorStatusHealthy:
			if status == port.MonitorStatusUnknown {
				status = port.MonitorStatusHealthy
			}
		case port.MonitorStatusStale, port.MonitorStatusUnknown:
			// A stale instance is a monitoring gap, not evidence of a service failure.
			continue
		}
	}
	return status
}

func cloneMonitorInstance(instance port.MonitorInstance) port.MonitorInstance {
	clone := instance
	clone.Components = append([]port.MonitorComponent(nil), instance.Components...)
	clone.Workers = append([]port.MonitorWorker(nil), instance.Workers...)
	clone.HTTP.LatencyBuckets = append([]uint64(nil), instance.HTTP.LatencyBuckets...)
	return clone
}

func monitorRange(rangeValue string) (time.Duration, error) {
	switch rangeValue {
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	case "90d":
		return statusRetention, nil
	default:
		return 0, fmt.Errorf("unsupported system monitor range")
	}
}

type observedStatus struct {
	replicaID  string
	instanceID string
	scope      string
	status     string
	message    string
	latencyMs  float64
	at         time.Time
}

func statusPeriods(samples []port.MonitorStatusSample, from, to time.Time) []port.MonitorStatusPeriod {
	streams := make(map[string][]observedStatus)
	for _, sample := range samples {
		for _, state := range sample.States {
			if state.Scope == "" || state.Status == port.MonitorStatusNotConfigured {
				continue
			}
			status := state.Status
			message := state.Message
			if status == port.MonitorStatusUnknown || status == port.MonitorStatusStale {
				status = "monitoring_gap"
				if message == "" {
					message = "monitoringGap"
				}
			}
			observation := observedStatus{
				replicaID: sample.ReplicaID, instanceID: sample.InstanceID,
				scope: state.Scope, status: status, message: message,
				latencyMs: state.LatencyMs, at: sample.CapturedAt,
			}
			key := sample.ReplicaID + "\x00" + state.Scope
			streams[key] = append(streams[key], observation)
		}
	}
	result := make([]port.MonitorStatusPeriod, 0)
	for _, observations := range streams {
		sort.Slice(observations, func(i, j int) bool {
			if observations[i].at.Equal(observations[j].at) {
				return observations[i].instanceID < observations[j].instanceID
			}
			return observations[i].at.Before(observations[j].at)
		})
		result = append(result, periodsForStream(observations, from, to)...)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Scope != result[j].Scope {
			return result[i].Scope < result[j].Scope
		}
		if result[i].ReplicaID != result[j].ReplicaID {
			return result[i].ReplicaID < result[j].ReplicaID
		}
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result
}

func periodsForStream(observations []observedStatus, from, to time.Time) []port.MonitorStatusPeriod {
	result := make([]port.MonitorStatusPeriod, 0, len(observations)/2+2)
	appendPeriod := func(observation observedStatus, status string, message string, start, end time.Time) {
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		if !end.After(start) {
			return
		}
		endCopy := end
		result = append(result, port.MonitorStatusPeriod{
			ReplicaID: observation.replicaID, InstanceID: observation.instanceID,
			Scope: observation.scope, Status: status, StartedAt: start, EndedAt: &endCopy,
			Message: message, LatencyMs: observation.latencyMs,
		})
	}
	first := observations[0]
	active := first
	activeStatus := first.status
	activeStart := first.at
	last := first
	for _, observation := range observations[1:] {
		if observation.at.Sub(last.at) > instanceStaleAfter {
			gapStart := last.at.Add(instanceStaleAfter)
			appendPeriod(last, activeStatus, active.message, activeStart, gapStart)
			appendPeriod(observation, "monitoring_gap", "monitoringGap", gapStart, observation.at)
			active, activeStatus, activeStart = observation, observation.status, observation.at
		} else if observation.status != activeStatus {
			appendPeriod(active, activeStatus, active.message, activeStart, observation.at)
			active, activeStatus, activeStart = observation, observation.status, observation.at
		} else {
			active = observation
		}
		last = observation
	}
	if to.Sub(last.at) > instanceStaleAfter {
		gapStart := last.at.Add(instanceStaleAfter)
		appendPeriod(last, activeStatus, active.message, activeStart, gapStart)
		appendPeriod(last, "monitoring_gap", "monitoringGap", gapStart, to)
	} else {
		appendPeriod(active, activeStatus, active.message, activeStart, to)
	}
	return result
}

type trendAccumulator struct {
	point          port.MonitorTrendPoint
	latencyBuckets []uint64
	latencyTotal   float64
	resourceCount  uint64
}

func aggregateTrends(samples []port.MonitorSample, now time.Time, span, bucket time.Duration) []port.MonitorTrendPoint {
	start := now.Add(-span).Truncate(bucket)
	count := int(span / bucket)
	accumulators := make([]trendAccumulator, count+1)
	for index := range accumulators {
		accumulators[index].point.Period = start.Add(time.Duration(index) * bucket)
		accumulators[index].latencyBuckets = make([]uint64, len(latencyBucketUpperMs)+1)
	}
	for _, sample := range samples {
		index := int(sample.CapturedAt.Sub(start) / bucket)
		if index < 0 || index >= len(accumulators) {
			continue
		}
		accumulator := &accumulators[index]
		http := sample.HTTP
		accumulator.point.Requests += http.Requests
		accumulator.point.Redirects += http.Status3xx
		accumulator.point.ClientErrors += http.Status4xx
		accumulator.point.ServerErrors += http.Status5xx
		accumulator.latencyTotal += http.AvgLatencyMs * float64(http.Requests)
		for bucketIndex, count := range http.LatencyBuckets {
			if bucketIndex < len(accumulator.latencyBuckets) {
				accumulator.latencyBuckets[bucketIndex] += count
			}
		}
		accumulator.point.CPUPercent += sample.Resources.CPUPercent
		accumulator.point.HeapAllocBytes += sample.Resources.HeapAllocBytes
		accumulator.resourceCount++
	}
	result := make([]port.MonitorTrendPoint, len(accumulators))
	for index, accumulator := range accumulators {
		point := accumulator.point
		if point.Requests > 0 {
			point.AvgLatencyMs = accumulator.latencyTotal / float64(point.Requests)
			point.P95LatencyMs = histogramPercentile(accumulator.latencyBuckets, 0.95)
		}
		if accumulator.resourceCount > 0 {
			point.CPUPercent /= float64(accumulator.resourceCount)
			point.HeapAllocBytes /= accumulator.resourceCount
		}
		result[index] = point
	}
	return result
}

var _ port.SystemMonitor = (*Collector)(nil)
