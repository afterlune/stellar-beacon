package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

type recordingRateLimiter struct {
	key   string
	limit int64
}

func (r *recordingRateLimiter) Allow(_ context.Context, key string, limit int64, _ time.Duration) (bool, error) {
	r.key = key
	r.limit = limit
	return true, nil
}

func TestAllowRateLimitHashesIdentifier(t *testing.T) {
	limiter := &recordingRateLimiter{}
	if allowed, err := allowRateLimit(context.Background(), limiter, "test:", "person@example.test", 3, time.Minute); err != nil || !allowed {
		t.Fatalf("allowRateLimit() = %v, %v", allowed, err)
	}
	if strings.Contains(limiter.key, "person@example.test") || !strings.HasPrefix(limiter.key, "test:") {
		t.Fatalf("rate-limit key leaked identifier: %q", limiter.key)
	}
	if limiter.limit != 3 {
		t.Fatalf("limit = %d, want 3", limiter.limit)
	}
}
