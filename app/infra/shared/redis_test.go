package shared

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*miniredis.Miniredis, func()) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	restore := setRedisClientForTest(client)
	return server, func() {
		restore()
		_ = client.Close()
		server.Close()
	}
}

func TestHSetCtxOverwritesAndRefreshesTTL(t *testing.T) {
	server, cleanup := newTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	if err := HSetCtx(ctx, "login_user", "1", "first", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := HSetCtx(ctx, "login_user", "1", "second", time.Hour); err != nil {
		t.Fatal(err)
	}
	got, err := HGetCtx(ctx, "login_user", "1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "second" {
		t.Fatalf("HGetCtx() = %q, want %q", got, "second")
	}
	if ttl := server.TTL("login_user"); ttl <= 0 || ttl > time.Hour {
		t.Fatalf("TTL = %s, want a positive value no greater than one hour", ttl)
	}
}

func TestIncrExpireCtxSetsTTLOnlyOnFirstIncrement(t *testing.T) {
	server, cleanup := newTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	first, err := IncrExpireCtx(ctx, "captcha:test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := IncrExpireCtx(ctx, "captcha:test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 2 {
		t.Fatalf("increments = (%d, %d), want (1, 2)", first, second)
	}
	if ttl := server.TTL("captcha:test"); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("TTL = %s, want a positive value no greater than one minute", ttl)
	}
}
