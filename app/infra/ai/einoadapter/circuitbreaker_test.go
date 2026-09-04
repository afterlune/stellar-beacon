package einoadapter

import (
	"testing"
	"time"
)

func TestCircuitBreakerOpensAllowsOneProbeAndClosesOnSuccess(t *testing.T) {
	base := time.Unix(100, 0)
	breaker := newCircuitBreaker(2, time.Minute)

	if !breaker.allow(base) {
		t.Fatal("closed circuit rejected the first request")
	}
	breaker.failure(true, base)
	if !breaker.allow(base.Add(time.Second)) {
		t.Fatal("closed circuit rejected a request before threshold")
	}
	breaker.failure(true, base.Add(time.Second))
	if breaker.allow(base.Add(2 * time.Second)) {
		t.Fatal("open circuit allowed a request before reset timeout")
	}

	probeAt := base.Add(time.Minute + time.Second)
	if !breaker.allow(probeAt) {
		t.Fatal("open circuit did not allow a half-open probe")
	}
	if breaker.allow(probeAt) {
		t.Fatal("circuit allowed more than one half-open probe")
	}
	breaker.success()
	if !breaker.allow(probeAt) {
		t.Fatal("successful probe did not close the circuit")
	}
}

func TestCircuitBreakerAbandonsCanceledHalfOpenProbe(t *testing.T) {
	base := time.Unix(200, 0)
	breaker := newCircuitBreaker(1, time.Minute)
	if !breaker.allow(base) {
		t.Fatal("closed circuit rejected request")
	}
	breaker.failure(true, base)
	probeAt := base.Add(time.Minute)
	if !breaker.allow(probeAt) {
		t.Fatal("circuit did not enter half-open state")
	}
	breaker.abandon()
	if !breaker.allow(probeAt) {
		t.Fatal("abandoned probe did not return circuit to closed state")
	}
}
