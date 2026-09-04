package cache

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"strings"
	"time"
)

const (
	agentSessionKeyPrefix = "agent:session:v1:"
	maxAgentSessionBytes  = 64 << 10
	maxAgentSessionTurns  = 6
)

// RedisAgentSessionStore stores only a bounded, short-lived conversation
// context. The owner hash is part of the Redis key so a known session ID is
// not sufficient to read or delete another visitor's session.
type RedisAgentSessionStore struct {
	cache port.Cache
	ttl   time.Duration
}

func NewAgentSessionStore(cache port.Cache) *RedisAgentSessionStore {
	return &RedisAgentSessionStore{cache: cache, ttl: port.DefaultAgentSessionTTL}
}

func NewAgentSessionStoreWithTTL(cache port.Cache, ttl time.Duration) *RedisAgentSessionStore {
	if ttl <= 0 {
		ttl = port.DefaultAgentSessionTTL
	}
	return &RedisAgentSessionStore{cache: cache, ttl: ttl}
}

func (s *RedisAgentSessionStore) Load(ctx context.Context, sessionID, ownerKey string) (port.AgentSession, bool, error) {
	key, err := s.key(sessionID, ownerKey)
	if err != nil {
		return port.AgentSession{}, false, err
	}
	if s == nil || s.cache == nil {
		return port.AgentSession{}, false, apperrors.Unavailable("agent.session.load", nil)
	}
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		if stderrors.Is(err, port.ErrCacheMiss) {
			return port.AgentSession{}, false, nil
		}
		return port.AgentSession{}, false, apperrors.WrapUnavailable("agent.session.load", err)
	}
	var session port.AgentSession
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return port.AgentSession{}, false, apperrors.Wrap(apperrors.KindUnavailable, "agent.session.decode", err)
	}
	if session.ID != strings.TrimSpace(sessionID) {
		return port.AgentSession{}, false, apperrors.Conflict("agent.session.load", "stored session identity does not match key")
	}
	return session, true, nil
}

func (s *RedisAgentSessionStore) Save(ctx context.Context, session port.AgentSession, ownerKey string) error {
	key, err := s.key(session.ID, ownerKey)
	if err != nil {
		return err
	}
	if s == nil || s.cache == nil {
		return apperrors.Unavailable("agent.session.save", nil)
	}
	session = normalizeAgentSession(session)
	raw, err := json.Marshal(session)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "agent.session.encode", err)
	}
	if len(raw) > maxAgentSessionBytes {
		return apperrors.Invalid("agent.session.save", "session context is too large")
	}
	if err := s.cache.Set(ctx, key, string(raw), s.ttl); err != nil {
		return apperrors.WrapUnavailable("agent.session.save", err)
	}
	return nil
}

func (s *RedisAgentSessionStore) Delete(ctx context.Context, sessionID, ownerKey string) error {
	key, err := s.key(sessionID, ownerKey)
	if err != nil {
		return err
	}
	if s == nil || s.cache == nil {
		return apperrors.Unavailable("agent.session.delete", nil)
	}
	if err := s.cache.Delete(ctx, key); err != nil {
		return apperrors.WrapUnavailable("agent.session.delete", err)
	}
	return nil
}

func (s *RedisAgentSessionStore) key(sessionID, ownerKey string) (string, error) {
	if s == nil {
		return "", apperrors.Unavailable("agent.session.key", nil)
	}
	sessionID, err := port.NormalizeAgentSessionID(sessionID)
	if err != nil || sessionID == "" {
		if err == nil {
			err = stderrors.New("session id is required")
		}
		return "", apperrors.Invalid("agent.session.key", err.Error())
	}
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return "", apperrors.Invalid("agent.session.key", "session owner is required")
	}
	hash := sha256.Sum256([]byte(ownerKey))
	return agentSessionKeyPrefix + hex.EncodeToString(hash[:]) + ":" + sessionID, nil
}

func normalizeAgentSession(session port.AgentSession) port.AgentSession {
	session.ID = strings.TrimSpace(session.ID)
	session.PromptVersion = strings.TrimSpace(session.PromptVersion)
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	session.UpdatedAt = time.Now().UTC()
	if len(session.Messages) > maxAgentSessionTurns*2 {
		session.Messages = append([]port.ChatMessage(nil), session.Messages[len(session.Messages)-maxAgentSessionTurns*2:]...)
	}
	return session
}

var _ port.AgentSessionStore = (*RedisAgentSessionStore)(nil)
