package cache

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const agentQuotaKeyPrefix = "agent:quota:v1:"

type AgentTurnQuota struct {
	cache         port.Cache
	guestDailyMax int64
	adminDailyMax int64
}

func NewAgentTurnQuota(cache port.Cache, guestDailyMax, adminDailyMax int) *AgentTurnQuota {
	if guestDailyMax <= 0 {
		guestDailyMax = 20
	}
	if adminDailyMax <= 0 {
		adminDailyMax = 200
	}
	return &AgentTurnQuota{
		cache:         cache,
		guestDailyMax: int64(guestDailyMax),
		adminDailyMax: int64(adminDailyMax),
	}
}

func (q *AgentTurnQuota) Allow(ctx context.Context, ownerKey string, privileged bool) (port.AgentQuotaDecision, error) {
	if q == nil || q.cache == nil {
		return port.AgentQuotaDecision{}, errors.Unavailable("agent.quota.allow", nil)
	}
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return port.AgentQuotaDecision{}, errors.Invalid("agent.quota.allow", "owner is required")
	}
	now := time.Now().UTC()
	limit := q.guestDailyMax
	scope := "guest"
	if privileged {
		limit = q.adminDailyMax
		scope = "admin"
	}
	resetAt := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	ttl := time.Until(resetAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	hash := sha256.Sum256([]byte(ownerKey))
	key := agentQuotaKeyPrefix + scope + ":" + now.Format("20060102") + ":" + hex.EncodeToString(hash[:])
	count, err := q.cache.IncrementWithExpiry(ctx, key, ttl)
	if err != nil {
		return port.AgentQuotaDecision{}, errors.WrapUnavailable("agent.quota.allow", err)
	}
	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}
	return port.AgentQuotaDecision{Allowed: count <= limit, Remaining: remaining, ResetAt: resetAt}, nil
}

var _ port.AgentTurnQuota = (*AgentTurnQuota)(nil)
