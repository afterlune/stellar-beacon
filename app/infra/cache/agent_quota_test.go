package cache

import (
	"benetnasch/app/domain/errors"
	"context"
	stderrors "errors"
	"testing"
)

func TestAgentTurnQuotaEnforcesSeparateGuestAndAdminLimits(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	quota := NewAgentTurnQuota(cache, 2, 3)
	ctx := context.Background()
	for index := 0; index < 2; index++ {
		decision, err := quota.Allow(ctx, "ip:one", false)
		if err != nil || !decision.Allowed {
			t.Fatalf("guest decision %d = %#v, error=%v", index, decision, err)
		}
	}
	decision, err := quota.Allow(ctx, "ip:one", false)
	if err != nil || decision.Allowed || decision.Remaining != 0 {
		t.Fatalf("guest over-limit decision = %#v, error=%v", decision, err)
	}
	admin, err := quota.Allow(ctx, "ip:one", true)
	if err != nil || !admin.Allowed {
		t.Fatalf("admin quota should be independent: %#v, error=%v", admin, err)
	}
}

func TestAgentTurnQuotaRequiresOwner(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	_, err := NewAgentTurnQuota(cache, 1, 1).Allow(context.Background(), "", false)
	if err == nil || errors.KindOf(err) != errors.KindValidation {
		t.Fatal("empty owner unexpectedly accepted")
	}
}

func TestAgentTurnQuotaPreservesContextErrors(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	quota := NewAgentTurnQuota(cache, 1, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := quota.Allow(ctx, "owner", false); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Allow() error = %v, want context canceled", err)
	}
}
