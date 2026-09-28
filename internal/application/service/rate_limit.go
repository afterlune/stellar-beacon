package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

// allowRateLimit hashes request-derived identifiers before they reach the
// cache. Redis keys are intentionally short-lived and contain no raw email or
// network address.
func allowRateLimit(ctx context.Context, limiter port.RateLimiter, prefix, value string, limit int64, window time.Duration) (bool, error) {
	if limiter == nil {
		return true, nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = "unknown"
	}
	digest := sha256.Sum256([]byte(value))
	return limiter.Allow(ctx, prefix+hex.EncodeToString(digest[:]), limit, window)
}
