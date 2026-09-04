package cache

import (
	"benetnasch/app/domain/port"
	"context"
	"errors"
	"testing"
	"time"
)

func TestRedisAgentSessionStoreScopesSessionsAndRefreshesTTL(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentSessionStoreWithTTL(cache, time.Minute)
	session := port.AgentSession{
		ID:            "session-1",
		PromptVersion: "v1",
		Messages:      []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
	}
	if err := store.Save(context.Background(), session, "ip:one"); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := store.Load(context.Background(), "session-1", "ip:one")
	if err != nil || !found || loaded.Messages[0].Content != "hello" {
		t.Fatalf("Load() = (%#v, %v, %v)", loaded, found, err)
	}
	if _, found, err := store.Load(context.Background(), "session-1", "ip:two"); err != nil || found {
		t.Fatalf("cross-owner Load() = (%v, %v), want miss", found, err)
	}
	keys := server.Keys()
	if len(keys) != 1 || server.TTL(keys[0]) <= 0 || server.TTL(keys[0]) > time.Minute {
		t.Fatalf("session TTL is not bounded: keys=%v ttl=%s", keys, server.TTL(keys[0]))
	}
	if err := store.Delete(context.Background(), "session-1", "ip:one"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Load(context.Background(), "session-1", "ip:one"); err != nil || found {
		t.Fatalf("Load() after Delete() = (%v, %v), want miss", found, err)
	}
}

func TestRedisAgentSessionStoreRejectsUnsafeIdentity(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentSessionStore(cache)
	if err := store.Save(context.Background(), port.AgentSession{ID: "../other"}, "owner"); err == nil {
		t.Fatal("unsafe session id was accepted")
	}
	if _, _, err := store.Load(context.Background(), "session-1", ""); err == nil {
		t.Fatal("empty owner was accepted")
	}
	if !errors.Is(context.Canceled, context.Canceled) {
		t.Fatal("test setup is invalid")
	}
}

func TestRedisAgentSessionStorePreservesContextErrors(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentSessionStore(cache)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	session := port.AgentSession{ID: "session-1", Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}}}

	if err := store.Save(ctx, session, "owner"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save() error = %v, want context canceled", err)
	}
	if _, _, err := store.Load(ctx, "session-1", "owner"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context canceled", err)
	}
	if err := store.Delete(ctx, "session-1", "owner"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() error = %v, want context canceled", err)
	}
}
