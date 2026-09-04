package cache

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"
)

func TestRedisAgentEventStoreReplaysOnlyLatestTurnAfterSequence(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStoreWithTTL(cache, time.Minute)
	ctx := context.Background()
	appendEvent := func(turn string, seq int64, text string) {
		t.Helper()
		err := store.Append(ctx, "ip:one", port.AgentChatEvent{
			EventID:   turn + "-event-" + text,
			Seq:       seq,
			Kind:      port.StreamEventDelta,
			SessionID: "session-1",
			TurnID:    turn,
			Text:      text,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	appendEvent("turn-1", 1, "old")
	appendEvent("turn-1", 2, "old-2")
	appendEvent("turn-2", 1, "new")
	appendEvent("turn-2", 2, "new-2")

	events, err := store.Replay(ctx, "ip:one", "session-1", "", 1)
	if err != nil || len(events) != 1 || events[0].TurnID != "turn-2" || events[0].Seq != 2 || !events[0].Replay {
		t.Fatalf("latest replay = %#v, error=%v", events, err)
	}
	old, err := store.Replay(ctx, "ip:one", "session-1", "turn-1", 0)
	if err != nil || len(old) != 0 {
		t.Fatalf("old turn replay = %#v, error=%v", old, err)
	}
	crossOwner, err := store.Replay(ctx, "ip:two", "session-1", "", 0)
	if err != nil || len(crossOwner) != 0 {
		t.Fatalf("cross-owner replay = %#v, error=%v", crossOwner, err)
	}

	keys := server.Keys()
	if len(keys) != 4 {
		t.Fatalf("event keys = %v, want two turn hashes, turn index, and latest pointer", keys)
	}
	for _, key := range keys {
		if server.TTL(key) <= 0 || server.TTL(key) > time.Minute {
			t.Fatalf("key %q TTL = %s, want bounded TTL", key, server.TTL(key))
		}
	}
}

func TestRedisAgentEventStoreDeleteRemovesAllTurnData(t *testing.T) {
	cache, server, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStoreWithTTL(cache, time.Minute)
	ctx := context.Background()
	for _, turnID := range []string{"turn-1", "turn-2"} {
		if err := store.Append(ctx, "ip:one", port.AgentChatEvent{
			Seq: 1, Kind: port.StreamEventDelta, SessionID: "session-1", TurnID: turnID, Text: turnID,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(ctx, "ip:one", "session-1"); err != nil {
		t.Fatal(err)
	}
	if keys := server.Keys(); len(keys) != 0 {
		t.Fatalf("deleted session keys = %v, want none", keys)
	}
	if events, err := store.Replay(ctx, "ip:one", "session-1", "", 0); err != nil || len(events) != 0 {
		t.Fatalf("replay after deletion = %#v, error=%v", events, err)
	}
}

func TestRedisAgentEventStoreRejectsInvalidEvents(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStore(cache)
	if err := store.Append(context.Background(), "owner", port.AgentChatEvent{SessionID: "session", TurnID: "turn", Kind: port.StreamEventDelta}); err == nil {
		t.Fatal("zero sequence was accepted")
	}
	if err := store.Append(context.Background(), "owner", port.AgentChatEvent{SessionID: "../other", TurnID: "turn", Seq: 1, Kind: port.StreamEventDelta}); err == nil {
		t.Fatal("unsafe session ID was accepted")
	}
	if _, err := store.Replay(context.Background(), "owner", "session", "", -1); err == nil {
		t.Fatal("negative after sequence was accepted")
	}
	if _, err := store.Replay(context.Background(), "owner", "../other", "", 0); err == nil {
		t.Fatal("unsafe replay session ID was accepted")
	}
	if err := store.Delete(context.Background(), "owner", "../other"); err == nil {
		t.Fatal("unsafe delete session ID was accepted")
	}
}

func TestRedisAgentEventStoreSkipsPayloadFromAnotherSession(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStore(cache)
	ctx := context.Background()
	raw, err := json.Marshal(port.AgentChatEvent{
		SessionID: "session-other",
		TurnID:    "turn-1",
		Seq:       1,
		Kind:      port.StreamEventDelta,
		Text:      "不应回放",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.HSet(ctx, agentEventKey("owner", "session-1", "turn-1"), "1", string(raw), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := cache.Set(ctx, agentLatestTurnKey("owner", "session-1"), "turn-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	events, err := store.Replay(ctx, "owner", "session-1", "", 0)
	if err != nil || len(events) != 0 {
		t.Fatalf("mismatched session replay = %#v, error=%v", events, err)
	}
}

func TestRedisAgentEventStorePreservesCanceledContext(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStore(cache)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Append(ctx, "owner", port.AgentChatEvent{
		SessionID: "session-1", TurnID: "turn-1", Seq: 1, Kind: port.StreamEventDelta, Text: "hello",
	}); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Append() error = %v, want context canceled", err)
	}
	if _, err := store.Replay(ctx, "owner", "session-1", "", 0); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Replay() error = %v, want context canceled", err)
	}
	if err := store.Delete(ctx, "owner", "session-1"); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Delete() error = %v, want context canceled", err)
	}
}

func TestRedisAgentEventStoreClassifiesNonContextCacheFailure(t *testing.T) {
	cache, _, cleanup := newTestCache(t)
	defer cleanup()
	store := NewAgentEventStore(cache)
	if err := cache.client.Close(); err != nil {
		t.Fatal(err)
	}
	err := store.Append(context.Background(), "owner", port.AgentChatEvent{
		SessionID: "session-1", TurnID: "turn-1", Seq: 1, Kind: port.StreamEventDelta, Text: "hello",
	})
	if !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("non-context cache error kind = %q, want unavailable", apperrors.KindOf(err))
	}
}
