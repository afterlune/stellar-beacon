package cache

import (
	"benetnasch/internal/domain/port"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*RedisCache, *miniredis.Miniredis, func()) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cache := NewRedisCacheWithClient(client)
	return cache, server, func() {
		_ = cache.Close()
		server.Close()
	}
}

func TestRedisCacheHSetOverwritesAndRefreshesTTL(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()
	if err := cache.HSet(ctx, "login_user", "1", "first", time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := cache.HSet(ctx, "login_user", "1", "second", time.Hour); err != nil {
		t.Fatal(err)
	}
	got, err := cache.HGet(ctx, "login_user", "1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "second" {
		t.Fatalf("HGet() = %q, want %q", got, "second")
	}
	if ttl := server.TTL("login_user"); ttl <= 0 || ttl > time.Hour {
		t.Fatalf("TTL = %s, want a positive value no greater than one hour", ttl)
	}
}

func TestRedisCacheMissIsTyped(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()

	_, err := cache.Get(context.Background(), "missing")
	if !errors.Is(err, port.ErrCacheMiss) {
		t.Fatalf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestRedisCacheIncrementWithExpiry(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()

	ctx := context.Background()
	first, err := cache.IncrementWithExpiry(ctx, "captcha:test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.IncrementWithExpiry(ctx, "captcha:test", time.Minute)
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
