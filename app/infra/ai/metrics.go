package ai

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const maxMetricRoutes = 64

// BudgetAlert is a sanitized operational signal. It contains no prompt,
// completion text, credentials or visitor identity.
type BudgetAlert struct {
	Date      string
	Budget    int64
	Used      int64
	Remaining int64
	Exceeded  bool
}

// RunMetricsObserverConfig keeps the observer usable in tests and in a
// future metrics exporter without coupling the AI adapter to a monitoring
// vendor. A zero DailyTokenBudget disables budget alerts.
type RunMetricsObserverConfig struct {
	Capacity         int
	DailyTokenBudget int64
	Now              func() time.Time
	OnBudgetAlert    func(BudgetAlert)
}

// RouteMetricsSnapshot contains bounded provider/use-case aggregates. The
// counters cover the observer lifetime; percentile latency is calculated from
// the bounded Recent window to keep memory usage predictable.
type RouteMetricsSnapshot struct {
	Provider      string         `json:"provider"`
	Model         string         `json:"model"`
	UseCase       port.AIUseCase `json:"useCase"`
	Total         uint64         `json:"total"`
	Succeeded     uint64         `json:"succeeded"`
	Failed        uint64         `json:"failed"`
	Canceled      uint64         `json:"canceled"`
	RateLimited   uint64         `json:"rateLimited"`
	ErrorRate     float64        `json:"errorRate"`
	DurationP95   time.Duration  `json:"durationP95"`
	FirstTokenP95 time.Duration  `json:"firstTokenP95"`
	InputTokens   uint64         `json:"inputTokens"`
	OutputTokens  uint64         `json:"outputTokens"`
	TotalTokens   uint64         `json:"totalTokens"`
}

// RunMetricsSnapshot is intentionally bounded. It is suitable for health or
// admin diagnostics and contains no prompt, completion text, or credentials.
// Percentiles describe the bounded Recent window; counters and token totals
// cover the observer lifetime (and DailyTokensUsed covers the current UTC
// day).
type RunMetricsSnapshot struct {
	Total                uint64                 `json:"total"`
	Succeeded            uint64                 `json:"succeeded"`
	Failed               uint64                 `json:"failed"`
	Canceled             uint64                 `json:"canceled"`
	RateLimited          uint64                 `json:"rateLimited"`
	ErrorRate            float64                `json:"errorRate"`
	SuccessRate          float64                `json:"successRate"`
	InputTokens          uint64                 `json:"inputTokens"`
	OutputTokens         uint64                 `json:"outputTokens"`
	TotalTokens          uint64                 `json:"totalTokens"`
	DurationP95          time.Duration          `json:"durationP95"`
	FirstTokenP95        time.Duration          `json:"firstTokenP95"`
	DailyTokenBudget     int64                  `json:"dailyTokenBudget"`
	DailyTokensUsed      int64                  `json:"dailyTokensUsed"`
	DailyTokensRemaining int64                  `json:"dailyTokensRemaining"`
	DailyBudgetAlert     bool                   `json:"dailyBudgetAlert"`
	DailyBudgetExceeded  bool                   `json:"dailyBudgetExceeded"`
	Routes               []RouteMetricsSnapshot `json:"routes"`
	Recent               []port.AgentRun        `json:"recent"`
}

type routeMetrics struct {
	Provider     string
	Model        string
	UseCase      port.AIUseCase
	Total        uint64
	Succeeded    uint64
	Failed       uint64
	Canceled     uint64
	RateLimited  uint64
	InputTokens  uint64
	OutputTokens uint64
	TotalTokens  uint64
}

// RunMetricsObserver records a bounded recent window plus counters. A later
// persistence-backed observer can implement the same domain port without
// changing the Eino adapter or application layer.
type RunMetricsObserver struct {
	mu          sync.RWMutex
	capacity    int
	recent      []port.AgentRun
	total       uint64
	success     uint64
	failed      uint64
	canceled    uint64
	rateLimited uint64

	inputTokens  uint64
	outputTokens uint64
	totalTokens  uint64
	routes       map[string]*routeMetrics

	dailyTokenBudget  int64
	dailyDate         string
	dailyTokensUsed   int64
	alertedDate       string
	exceededAlertDate string
	now               func() time.Time
	onBudgetAlert     func(BudgetAlert)
}

var _ port.AIRunObserver = (*RunMetricsObserver)(nil)

func NewRunMetricsObserver(capacity int) *RunMetricsObserver {
	return NewRunMetricsObserverWithConfig(RunMetricsObserverConfig{Capacity: capacity})
}

func NewRunMetricsObserverWithConfig(config RunMetricsObserverConfig) *RunMetricsObserver {
	if config.Capacity <= 0 {
		config.Capacity = 128
	}
	if config.DailyTokenBudget < 0 {
		config.DailyTokenBudget = 0
	}
	return &RunMetricsObserver{
		capacity:         config.Capacity,
		recent:           make([]port.AgentRun, 0, config.Capacity),
		routes:           make(map[string]*routeMetrics),
		dailyTokenBudget: config.DailyTokenBudget,
		now:              config.Now,
		onBudgetAlert:    config.OnBudgetAlert,
	}
}

func (o *RunMetricsObserver) ObserveAIRun(_ context.Context, run port.AgentRun) {
	if o == nil {
		return
	}
	now := o.currentTime()
	var alerts []BudgetAlert
	o.mu.Lock()
	o.resetDailyIfNeeded(now)
	o.total++
	switch run.Status {
	case port.AIRunSucceeded:
		o.success++
	case port.AIRunCanceled:
		o.canceled++
	default:
		o.failed++
	}
	if strings.TrimSpace(run.ErrorCode) == string(apperrors.AICodeRateLimited) {
		o.rateLimited++
	}
	inputTokens := nonNegativeTokenCount(run.Usage.InputTokens)
	outputTokens := nonNegativeTokenCount(run.Usage.OutputTokens)
	totalTokens := nonNegativeTokenCount(run.Usage.TotalTokens)
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	o.inputTokens += uint64(inputTokens)
	o.outputTokens += uint64(outputTokens)
	o.totalTokens += uint64(totalTokens)
	o.dailyTokensUsed += totalTokens
	o.observeRoute(run, inputTokens, outputTokens, totalTokens)
	if o.capacity > 0 {
		if len(o.recent) == o.capacity {
			copy(o.recent, o.recent[1:])
			o.recent[len(o.recent)-1] = run
		} else {
			o.recent = append(o.recent, run)
		}
	}
	if o.dailyTokenBudget > 0 {
		threshold := o.dailyTokenBudget - o.dailyTokenBudget/5
		if o.dailyTokensUsed >= threshold && o.alertedDate != o.dailyDate {
			o.alertedDate = o.dailyDate
			alerts = append(alerts, o.newBudgetAlert(false))
		}
		if o.dailyTokensUsed >= o.dailyTokenBudget && o.exceededAlertDate != o.dailyDate {
			o.exceededAlertDate = o.dailyDate
			alerts = append(alerts, o.newBudgetAlert(true))
		}
	}
	callback := o.onBudgetAlert
	o.mu.Unlock()
	if callback != nil {
		for _, alert := range alerts {
			callback(alert)
		}
	}
}

func (o *RunMetricsObserver) Snapshot() RunMetricsSnapshot {
	if o == nil {
		return RunMetricsSnapshot{}
	}
	now := o.currentTime()
	o.mu.Lock()
	o.resetDailyIfNeeded(now)
	defer o.mu.Unlock()

	errorCount := o.failed + o.canceled
	return RunMetricsSnapshot{
		Total:                o.total,
		Succeeded:            o.success,
		Failed:               o.failed,
		Canceled:             o.canceled,
		RateLimited:          o.rateLimited,
		ErrorRate:            ratio(errorCount, o.total),
		SuccessRate:          ratio(o.success, o.total),
		InputTokens:          o.inputTokens,
		OutputTokens:         o.outputTokens,
		TotalTokens:          o.totalTokens,
		DailyTokenBudget:     o.dailyTokenBudget,
		DailyTokensUsed:      o.dailyTokensUsed,
		DailyTokensRemaining: dailyTokensRemaining(o.dailyTokenBudget, o.dailyTokensUsed),
		DailyBudgetAlert:     o.dailyTokenBudget > 0 && o.dailyTokensUsed >= o.dailyTokenBudget-o.dailyTokenBudget/5,
		DailyBudgetExceeded:  o.dailyTokenBudget > 0 && o.dailyTokensUsed >= o.dailyTokenBudget,
		DurationP95:          percentile95(runDurations(o.recent, false)),
		FirstTokenP95:        percentile95(runDurations(o.recent, true)),
		Recent:               append([]port.AgentRun(nil), o.recent...),
		Routes:               o.routeSnapshots(),
	}
}

func (o *RunMetricsObserver) observeRoute(run port.AgentRun, inputTokens, outputTokens, totalTokens int64) {
	key := metricRouteKey(run)
	metrics, ok := o.routes[key]
	if !ok {
		if len(o.routes) >= maxMetricRoutes {
			return
		}
		metrics = &routeMetrics{Provider: strings.TrimSpace(run.Provider), Model: strings.TrimSpace(run.Model), UseCase: run.UseCase}
		o.routes[key] = metrics
	}
	metrics.Total++
	switch run.Status {
	case port.AIRunSucceeded:
		metrics.Succeeded++
	case port.AIRunCanceled:
		metrics.Canceled++
	default:
		metrics.Failed++
	}
	if strings.TrimSpace(run.ErrorCode) == string(apperrors.AICodeRateLimited) {
		metrics.RateLimited++
	}
	metrics.InputTokens += uint64(inputTokens)
	metrics.OutputTokens += uint64(outputTokens)
	metrics.TotalTokens += uint64(totalTokens)
}

func (o *RunMetricsObserver) routeSnapshots() []RouteMetricsSnapshot {
	if len(o.routes) == 0 {
		return []RouteMetricsSnapshot{}
	}
	keys := make([]string, 0, len(o.routes))
	for key := range o.routes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]RouteMetricsSnapshot, 0, len(keys))
	for _, key := range keys {
		metrics := o.routes[key]
		result = append(result, RouteMetricsSnapshot{
			Provider:      metrics.Provider,
			Model:         metrics.Model,
			UseCase:       metrics.UseCase,
			Total:         metrics.Total,
			Succeeded:     metrics.Succeeded,
			Failed:        metrics.Failed,
			Canceled:      metrics.Canceled,
			RateLimited:   metrics.RateLimited,
			ErrorRate:     ratio(metrics.Failed+metrics.Canceled, metrics.Total),
			DurationP95:   percentile95(routeDurations(o.recent, key, false)),
			FirstTokenP95: percentile95(routeDurations(o.recent, key, true)),
			InputTokens:   metrics.InputTokens,
			OutputTokens:  metrics.OutputTokens,
			TotalTokens:   metrics.TotalTokens,
		})
	}
	return result
}

func (o *RunMetricsObserver) newBudgetAlert(exceeded bool) BudgetAlert {
	return BudgetAlert{
		Date:      o.dailyDate,
		Budget:    o.dailyTokenBudget,
		Used:      o.dailyTokensUsed,
		Remaining: dailyTokensRemaining(o.dailyTokenBudget, o.dailyTokensUsed),
		Exceeded:  exceeded,
	}
}

func (o *RunMetricsObserver) resetDailyIfNeeded(now time.Time) {
	date := now.UTC().Format("2006-01-02")
	if date == o.dailyDate {
		return
	}
	o.dailyDate = date
	o.dailyTokensUsed = 0
	o.alertedDate = ""
	o.exceededAlertDate = ""
}

func (o *RunMetricsObserver) currentTime() time.Time {
	if o.now != nil {
		if now := o.now(); !now.IsZero() {
			return now.UTC()
		}
	}
	return time.Now().UTC()
}

func metricRouteKey(run port.AgentRun) string {
	return strings.Join([]string{string(run.UseCase), strings.TrimSpace(run.Provider), strings.TrimSpace(run.Model)}, "\x00")
}

func routeDurations(runs []port.AgentRun, key string, firstToken bool) []time.Duration {
	result := make([]time.Duration, 0)
	for _, run := range runs {
		if metricRouteKey(run) != key {
			continue
		}
		if firstToken {
			if run.FirstTokenAt.IsZero() || run.FirstTokenLatency < 0 {
				continue
			}
			result = append(result, run.FirstTokenLatency)
			continue
		}
		if run.Duration >= 0 {
			result = append(result, run.Duration)
		}
	}
	return result
}

func runDurations(runs []port.AgentRun, firstToken bool) []time.Duration {
	result := make([]time.Duration, 0, len(runs))
	for _, run := range runs {
		if firstToken {
			if run.FirstTokenAt.IsZero() || run.FirstTokenLatency < 0 {
				continue
			}
			result = append(result, run.FirstTokenLatency)
			continue
		}
		if run.Duration >= 0 {
			result = append(result, run.Duration)
		}
	}
	return result
}

func percentile95(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	index := (len(values)*95+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func ratio(numerator, denominator uint64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func nonNegativeTokenCount(value int) int64 {
	if value <= 0 {
		return 0
	}
	return int64(value)
}

func dailyTokensRemaining(budget, used int64) int64 {
	if budget <= 0 {
		return 0
	}
	if used >= budget {
		return 0
	}
	return budget - used
}
