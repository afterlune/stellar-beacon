package cache

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"errors"
	"testing"
	"time"
)

func TestRedisAgentSessionCoordinatorSerializesAndReleasesOwnership(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinatorWithTTL(cache, time.Minute)

	first, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	blockedCtx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err := coordinator.Acquire(blockedCtx, "ip:one", "session-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Acquire() error = %v, want deadline exceeded", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}

	second, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
	if keys := server.Keys(); len(keys) != 0 {
		t.Fatalf("released session lock keys = %v, want none", keys)
	}
}

func TestRedisAgentSessionCoordinatorAcquirePreservesCanceledContext(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinatorWithTTL(cache, time.Minute)

	first, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := coordinator.Acquire(ctx, "ip:one", "session-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Acquire() with canceled context error = %v, want context canceled", err)
	}
}

func TestRedisAgentSessionCoordinatorStaleLeaseCannotDeleteNewOwner(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinatorWithTTL(cache, time.Minute)

	first, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	server.FastForward(time.Minute)
	second, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}

	blockedCtx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err := coordinator.Acquire(blockedCtx, "ip:one", "session-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() after stale release error = %v, want deadline exceeded", err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisAgentSessionLeaseKeepAliveRenewsBeforeExpiry(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinatorWithTTL(cache, 300*time.Millisecond)

	lease, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	keepAliveCtx, cancelKeepAlive := context.WithCancel(context.Background())
	keepAliveDone := make(chan error, 1)
	go func() {
		keepAliveDone <- lease.KeepAlive(keepAliveCtx)
	}()

	blockedCtx, cancelBlocked := context.WithTimeout(context.Background(), 650*time.Millisecond)
	defer cancelBlocked()
	if _, err := coordinator.Acquire(blockedCtx, "ip:one", "session-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Acquire() error = %v, want deadline exceeded while keepalive is active", err)
	}

	cancelKeepAlive()
	if err := <-keepAliveDone; err != nil {
		t.Fatalf("KeepAlive() after cancellation error = %v, want nil", err)
	}
	if err := lease.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisAgentSessionLeaseKeepAliveStopsWhenOwnershipIsLost(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinatorWithTTL(cache, 300*time.Millisecond)

	first, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	server.FastForward(300 * time.Millisecond)
	second, err := coordinator.Acquire(context.Background(), "ip:one", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Release()

	keepAliveCtx, cancelKeepAlive := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancelKeepAlive()
	if err := first.KeepAlive(keepAliveCtx); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("stale KeepAlive() error = %v, want conflict", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestRedisAgentSessionCoordinatorValidatesIdentity(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	coordinator := NewAgentSessionCoordinator(cache)

	for _, test := range []struct {
		name      string
		ownerKey  string
		sessionID string
	}{
		{name: "missing owner", sessionID: "session-1"},
		{name: "missing session", ownerKey: "ip:one"},
		{name: "unsafe session", ownerKey: "ip:one", sessionID: "../other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := coordinator.Acquire(context.Background(), test.ownerKey, test.sessionID); err == nil {
				t.Fatal("invalid identity was accepted")
			}
		})
	}
}

var _ port.AgentSessionCoordinator = (*RedisAgentSessionCoordinator)(nil)
