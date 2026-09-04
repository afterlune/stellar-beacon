package cache

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultAgentSessionLockTTL = 2 * time.Minute
	agentSessionLockRetry      = 50 * time.Millisecond
)

// RedisAgentSessionCoordinator serializes a complete public Agent turn for
// one owner/session pair. The lease is deliberately longer than the normal
// upstream request budget and long enough for the stream plus one final tool
// round; the provider adapters still own their per-request timeout.
type RedisAgentSessionCoordinator struct {
	cache         *RedisCache
	ttl           time.Duration
	retryInterval time.Duration
}

func NewAgentSessionCoordinator(cache *RedisCache) *RedisAgentSessionCoordinator {
	return NewAgentSessionCoordinatorWithTTL(cache, defaultAgentSessionLockTTL)
}

func NewAgentSessionCoordinatorWithTTL(cache *RedisCache, ttl time.Duration) *RedisAgentSessionCoordinator {
	if ttl <= 0 {
		ttl = defaultAgentSessionLockTTL
	}
	return &RedisAgentSessionCoordinator{
		cache:         cache,
		ttl:           ttl,
		retryInterval: agentSessionLockRetry,
	}
}

func (c *RedisAgentSessionCoordinator) Acquire(ctx context.Context, ownerKey, sessionID string) (port.AgentSessionLease, error) {
	if c == nil || c.cache == nil || c.cache.client == nil {
		return nil, apperrors.Unavailable("agent.session.lock", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ownerKey = strings.TrimSpace(ownerKey)
	sessionID, err := port.NormalizeAgentSessionID(sessionID)
	if err != nil || sessionID == "" {
		if err == nil {
			err = errors.New("session id is required")
		}
		return nil, apperrors.Invalid("agent.session.lock", err.Error())
	}
	if ownerKey == "" {
		return nil, apperrors.Invalid("agent.session.lock", "session owner is required")
	}

	key := agentSessionLockKey(ownerKey, sessionID)
	token := uuid.NewString()
	for {
		acquired, err := c.cache.client.SetNX(cacheContext(ctx), key, token, c.ttl).Result()
		if err != nil {
			return nil, apperrors.WrapUnavailable("agent.session.lock", err)
		}
		if acquired {
			return &redisAgentSessionLease{
				cache:         c.cache,
				key:           key,
				token:         token,
				ttl:           c.ttl,
				renewInterval: leaseRenewInterval(c.ttl),
			}, nil
		}

		timer := time.NewTimer(c.retryInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func agentSessionLockKey(ownerKey, sessionID string) string {
	hash := sha256.Sum256([]byte(ownerKey))
	return fmt.Sprintf("agent:session-lock:v1:%s:%s", hex.EncodeToString(hash[:]), sessionID)
}

type redisAgentSessionLease struct {
	cache         *RedisCache
	key           string
	token         string
	ttl           time.Duration
	renewInterval time.Duration
}

// KeepAlive renews the lease until the caller cancels ctx. A lease loss is a
// hard error: continuing the turn after another request has acquired the key
// would reintroduce the read/modify/write race this coordinator prevents.
func (l *redisAgentSessionLease) KeepAlive(ctx context.Context) error {
	if l == nil || l.cache == nil || l.cache.client == nil {
		return apperrors.Unavailable("agent.session.keepalive", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	interval := l.renewInterval
	if interval <= 0 {
		interval = leaseRenewInterval(l.ttl)
	}

	// Verify ownership immediately instead of waiting for the first renewal
	// interval. This closes the window in which a caller could start work with
	// an already expired lease and also makes lease loss deterministic under
	// scheduler or Redis load.
	renewed, err := l.renew(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}
	if !renewed {
		return apperrors.Conflict("agent.session.keepalive", "session lease was lost")
	}

	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			renewed, err := l.renew(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return apperrors.WrapUnavailable("agent.session.keepalive", err)
			}
			if !renewed {
				return apperrors.Conflict("agent.session.keepalive", "session lease was lost")
			}
			timer.Reset(interval)
		}
	}
}

func (l *redisAgentSessionLease) renew(ctx context.Context) (bool, error) {
	renewCtx, cancel := context.WithTimeout(ctx, leaseRenewTimeout(l.ttl))
	defer cancel()
	return l.cache.CompareAndExpire(renewCtx, l.key, l.token, l.ttl)
}

func (l *redisAgentSessionLease) Release() error {
	if l == nil || l.cache == nil || l.cache.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := l.cache.CompareAndDelete(ctx, l.key, l.token)
	if err != nil {
		return apperrors.WrapUnavailable("agent.session.unlock", err)
	}
	return nil
}

var _ port.AgentSessionCoordinator = (*RedisAgentSessionCoordinator)(nil)
var _ port.AgentSessionLease = (*redisAgentSessionLease)(nil)

func leaseRenewInterval(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		ttl = defaultAgentSessionLockTTL
	}
	interval := ttl / 3
	if interval <= 0 {
		return time.Millisecond
	}
	return interval
}

func leaseRenewTimeout(ttl time.Duration) time.Duration {
	const maxTimeout = 2 * time.Second
	if ttl <= 0 || ttl >= maxTimeout {
		return maxTimeout
	}
	timeout := ttl / 2
	if timeout <= 0 {
		return time.Millisecond
	}
	return timeout
}
