package einoadapter

import (
	"sync"
	"time"
)

type circuitState uint8

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

// circuitBreaker is deliberately local to one ChatAdapter. Since the router
// creates one adapter per capability route, a broken writing model cannot
// take the public chat route down with it.
type circuitBreaker struct {
	mu sync.Mutex

	failureThreshold int
	resetTimeout     time.Duration
	state            circuitState
	failures         int
	openedAt         time.Time
	probeInFlight    bool
}

func newCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *circuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if resetTimeout <= 0 {
		resetTimeout = 10 * time.Second
	}
	return &circuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

func (b *circuitBreaker) allow(now time.Time) bool {
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if now.Before(b.openedAt.Add(b.resetTimeout)) {
			return false
		}
		b.state = circuitHalfOpen
		b.probeInFlight = true
		return true
	case circuitHalfOpen:
		// Only one request may probe a recovered provider. Other callers fail
		// fast until that request records success or failure.
		return false
	default:
		b.state = circuitOpen
		b.openedAt = now
		return false
	}
}

func (b *circuitBreaker) success() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.state = circuitClosed
	b.failures = 0
	b.probeInFlight = false
	b.mu.Unlock()
}

func (b *circuitBreaker) failure(retryable bool, now time.Time) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == circuitHalfOpen {
		b.state = circuitOpen
		b.openedAt = now
		b.probeInFlight = false
		b.failures = b.failureThreshold
		return
	}
	if b.state != circuitClosed || !retryable {
		return
	}
	b.failures++
	if b.failures >= b.failureThreshold {
		b.state = circuitOpen
		b.openedAt = now
	}
}

func (b *circuitBreaker) abandon() {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.state == circuitHalfOpen {
		// Cancellation or a disconnected consumer is not provider evidence.
		// Return to closed so a later request can probe normally.
		b.state = circuitClosed
		b.failures = 0
		b.probeInFlight = false
	}
	b.mu.Unlock()
}

func (b *circuitBreaker) stateName() string {
	if b == nil {
		return "disabled"
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitOpen:
		return "open"
	case circuitHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}
