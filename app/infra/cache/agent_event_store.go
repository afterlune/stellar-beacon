package cache

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAgentEventTTL   = 30 * time.Minute
	maxAgentReplayEvents   = 128
	maxAgentEventPayload   = 64 << 10
	maxAgentEventTurnBytes = 128
)

// RedisAgentEventStore stores public SSE frames in a per-turn hash. The
// latest-turn pointer makes a reconnect without an explicit turn safe: an
// old turn is never mixed into the current replay stream.
type RedisAgentEventStore struct {
	cache port.Cache
	ttl   time.Duration
}

func NewAgentEventStore(cache port.Cache) *RedisAgentEventStore {
	return NewAgentEventStoreWithTTL(cache, defaultAgentEventTTL)
}

func NewAgentEventStoreWithTTL(cache port.Cache, ttl time.Duration) *RedisAgentEventStore {
	if ttl <= 0 {
		ttl = defaultAgentEventTTL
	}
	return &RedisAgentEventStore{cache: cache, ttl: ttl}
}

func (s *RedisAgentEventStore) Append(ctx context.Context, ownerKey string, event port.AgentChatEvent) error {
	if s == nil || s.cache == nil {
		return errors.Unavailable("agent.events.append", nil)
	}
	ownerKey = strings.TrimSpace(ownerKey)
	var err error
	event.SessionID, err = normalizeAgentEventID(event.SessionID, "session")
	if err != nil {
		return errors.Invalid("agent.events.append", err.Error())
	}
	event.TurnID, err = normalizeAgentEventID(event.TurnID, "turn")
	if err != nil {
		return errors.Invalid("agent.events.append", err.Error())
	}
	if ownerKey == "" {
		return errors.Invalid("agent.events.append", "owner, session, and turn are required")
	}
	if len(event.TurnID) > maxAgentEventTurnBytes || event.Seq <= 0 {
		return errors.Invalid("agent.events.append", "event turn or sequence is invalid")
	}
	if !isReplayableAgentEvent(event.Kind) {
		return errors.Invalid("agent.events.append", "event type is not replayable")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(errors.KindInternal, "agent.events.encode", err)
	}
	if len(payload) > maxAgentEventPayload {
		return errors.Invalid("agent.events.append", "event payload is too large")
	}
	key := agentEventKey(ownerKey, event.SessionID, event.TurnID)
	if err := s.cache.HSet(ctx, key, strconv.FormatInt(event.Seq, 10), string(payload), s.ttl); err != nil {
		return errors.WrapUnavailable("agent.events.append", err)
	}
	if err := s.cache.HSet(ctx, agentEventTurnIndexKey(ownerKey, event.SessionID), event.TurnID, "1", s.ttl); err != nil {
		return errors.WrapUnavailable("agent.events.turn_index", err)
	}
	if err := s.cache.Set(ctx, agentLatestTurnKey(ownerKey, event.SessionID), event.TurnID, s.ttl); err != nil {
		return errors.WrapUnavailable("agent.events.latest", err)
	}
	if err := s.trim(ctx, key); err != nil {
		return err
	}
	return nil
}

func (s *RedisAgentEventStore) Replay(ctx context.Context, ownerKey, sessionID, turnID string, afterSeq int64) ([]port.AgentChatEvent, error) {
	if s == nil || s.cache == nil {
		return nil, errors.Unavailable("agent.events.replay", nil)
	}
	ownerKey = strings.TrimSpace(ownerKey)
	sessionID, err := normalizeAgentEventID(sessionID, "session")
	if err != nil {
		return nil, errors.Invalid("agent.events.replay", err.Error())
	}
	turnID = strings.TrimSpace(turnID)
	if turnID != "" {
		turnID, err = normalizeAgentEventID(turnID, "turn")
		if err != nil {
			return nil, errors.Invalid("agent.events.replay", err.Error())
		}
	}
	if ownerKey == "" {
		return nil, errors.Invalid("agent.events.replay", "owner and session are required")
	}
	if afterSeq < 0 {
		return nil, errors.Invalid("agent.events.replay", "after sequence cannot be negative")
	}
	latest, err := s.cache.Get(ctx, agentLatestTurnKey(ownerKey, sessionID))
	if err != nil {
		if stderrors.Is(err, port.ErrCacheMiss) {
			return []port.AgentChatEvent{}, nil
		}
		return nil, errors.WrapUnavailable("agent.events.latest", err)
	}
	latest = strings.TrimSpace(latest)
	if latest != "" {
		latest, err = normalizeAgentEventID(latest, "turn")
		if err != nil {
			// A corrupted latest pointer must fail closed rather than selecting a
			// key outside the event identity contract.
			return []port.AgentChatEvent{}, nil
		}
	}
	if latest == "" || (turnID != "" && turnID != latest) {
		// Explicitly filter stale turns, even if their individual Redis hash has
		// not expired yet.
		return []port.AgentChatEvent{}, nil
	}
	turnID = latest
	fields, err := s.cache.HGetAll(ctx, agentEventKey(ownerKey, sessionID, turnID))
	if err != nil && !stderrors.Is(err, port.ErrCacheMiss) {
		return nil, errors.WrapUnavailable("agent.events.replay", err)
	}
	result := make([]port.AgentChatEvent, 0, len(fields))
	for field, raw := range fields {
		seq, parseErr := strconv.ParseInt(field, 10, 64)
		if parseErr != nil || seq <= afterSeq {
			continue
		}
		var event port.AgentChatEvent
		if json.Unmarshal([]byte(raw), &event) != nil || event.SessionID != sessionID || event.TurnID != turnID || !isReplayableAgentEvent(event.Kind) {
			continue
		}
		event.Seq = seq
		event.Replay = true
		result = append(result, event)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Seq < result[j].Seq })
	return result, nil
}

func (s *RedisAgentEventStore) Delete(ctx context.Context, ownerKey, sessionID string) error {
	if s == nil || s.cache == nil {
		return errors.Unavailable("agent.events.delete", nil)
	}
	ownerKey = strings.TrimSpace(ownerKey)
	sessionID, err := normalizeAgentEventID(sessionID, "session")
	if err != nil {
		return errors.Invalid("agent.events.delete", err.Error())
	}
	if ownerKey == "" {
		return errors.Invalid("agent.events.delete", "owner and session are required")
	}
	latest, err := s.cache.Get(ctx, agentLatestTurnKey(ownerKey, sessionID))
	if err != nil && !stderrors.Is(err, port.ErrCacheMiss) {
		return errors.WrapUnavailable("agent.events.delete.latest", err)
	}
	turns, err := s.cache.HGetAll(ctx, agentEventTurnIndexKey(ownerKey, sessionID))
	if err != nil && !stderrors.Is(err, port.ErrCacheMiss) {
		return errors.WrapUnavailable("agent.events.delete.turn_index", err)
	}
	turnIDs := make(map[string]struct{}, len(turns)+1)
	for turnID := range turns {
		turnID, normalizeErr := normalizeAgentEventID(turnID, "turn")
		if normalizeErr == nil {
			turnIDs[turnID] = struct{}{}
		}
	}
	if latest = strings.TrimSpace(latest); latest != "" {
		// The index was introduced after the original event format. Including
		// latest keeps deletion complete for sessions written during a partial
		// rollout where the index may not exist yet.
		if normalizedLatest, normalizeErr := normalizeAgentEventID(latest, "turn"); normalizeErr == nil {
			turnIDs[normalizedLatest] = struct{}{}
		}
	}
	for turnID := range turnIDs {
		if err := s.cache.Delete(ctx, agentEventKey(ownerKey, sessionID, turnID)); err != nil {
			return errors.WrapUnavailable("agent.events.delete.turn", err)
		}
	}
	if err := s.cache.Delete(ctx, agentEventTurnIndexKey(ownerKey, sessionID)); err != nil {
		return errors.WrapUnavailable("agent.events.delete.turn_index", err)
	}
	if err := s.cache.Delete(ctx, agentLatestTurnKey(ownerKey, sessionID)); err != nil {
		return errors.WrapUnavailable("agent.events.delete.latest", err)
	}
	return nil
}

func (s *RedisAgentEventStore) trim(ctx context.Context, key string) error {
	fields, err := s.cache.HGetAll(ctx, key)
	if err != nil {
		return errors.WrapUnavailable("agent.events.trim", err)
	}
	if len(fields) <= maxAgentReplayEvents {
		return nil
	}
	sequences := make([]int64, 0, len(fields))
	for field := range fields {
		seq, parseErr := strconv.ParseInt(field, 10, 64)
		if parseErr == nil && seq > 0 {
			sequences = append(sequences, seq)
		}
	}
	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })
	remove := len(sequences) - maxAgentReplayEvents
	for _, seq := range sequences[:remove] {
		if err := s.cache.HDel(ctx, key, strconv.FormatInt(seq, 10)); err != nil {
			return errors.WrapUnavailable("agent.events.trim", err)
		}
	}
	return nil
}

func isReplayableAgentEvent(kind port.StreamEventKind) bool {
	switch kind {
	case port.StreamEventMeta, port.StreamEventDelta, port.StreamEventCitation, port.StreamEventState, port.StreamEventDone, port.StreamEventError:
		return true
	default:
		return false
	}
}

func agentEventKey(ownerKey, sessionID, turnID string) string {
	return fmt.Sprintf("agent:events:v1:%s:%s:%s", hashAgentEventPart(ownerKey), hashAgentEventPart(sessionID), hashAgentEventPart(turnID))
}

func agentLatestTurnKey(ownerKey, sessionID string) string {
	return fmt.Sprintf("agent:events:v1:%s:%s:latest", hashAgentEventPart(ownerKey), hashAgentEventPart(sessionID))
}

func agentEventTurnIndexKey(ownerKey, sessionID string) string {
	return fmt.Sprintf("agent:events:v1:%s:%s:turns", hashAgentEventPart(ownerKey), hashAgentEventPart(sessionID))
}

func normalizeAgentEventID(value, name string) (string, error) {
	normalized, err := port.NormalizeAgentSessionID(value)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return normalized, nil
}

func hashAgentEventPart(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

var _ port.AgentEventStore = (*RedisAgentEventStore)(nil)
