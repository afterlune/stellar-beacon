package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

type pressureAgentSessionStore struct {
	mu       sync.Mutex
	sessions map[string]port.AgentSession
}

func (s *pressureAgentSessionStore) Load(_ context.Context, sessionID, _ string) (port.AgentSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	return session, ok, nil
}

func (s *pressureAgentSessionStore) Save(_ context.Context, session port.AgentSession, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *pressureAgentSessionStore) Delete(context.Context, string, string) error { return nil }

type pressureAgentTools struct{}

func (pressureAgentTools) Definitions() []port.ToolDefinition { return nil }

func (pressureAgentTools) Execute(context.Context, string, json.RawMessage) (port.PublicAgentToolResult, error) {
	return port.PublicAgentToolResult{}, fmt.Errorf("pressure test tool should not be called")
}

type pressureAgentChatGateway struct {
	hold      chan struct{}
	started   chan struct{}
	active    int32
	maxActive int32
}

func (g *pressureAgentChatGateway) Generate(context.Context, port.ChatRequest) (port.ChatResponse, error) {
	return port.ChatResponse{Text: "unused"}, nil
}

func (g *pressureAgentChatGateway) Stream(ctx context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	active := atomic.AddInt32(&g.active, 1)
	defer atomic.AddInt32(&g.active, -1)
	for {
		previous := atomic.LoadInt32(&g.maxActive)
		if active <= previous || atomic.CompareAndSwapInt32(&g.maxActive, previous, active) {
			break
		}
	}
	select {
	case g.started <- struct{}{}:
	default:
	}
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: "pressure-run", Provider: "test"}); err != nil {
		return err
	}
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, RunID: "pressure-run", Text: "完成"}); err != nil {
		return err
	}
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: "pressure-run"}); err != nil {
		return err
	}
	select {
	case <-g.hold:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestAgentChatSupportsTwentyConcurrentSessionsWithinBoundedSSESlots(t *testing.T) {
	gateway := &pressureAgentChatGateway{hold: make(chan struct{}), started: make(chan struct{}, 20)}
	store := &pressureAgentSessionStore{sessions: make(map[string]port.AgentSession)}
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", PromptVersion: "v1", SystemPrompt: "基础人设", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: newTestAgentSessionCoordinator(),
		Tools:       pressureAgentTools{},
		Quota:       agentChatQuota{},
		Limits:      AgentChatLimits{MaxConcurrent: 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	const sessions = 20
	var waitGroup sync.WaitGroup
	errorsCh := make(chan error, sessions)
	var events int32
	for index := 0; index < sessions; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			err := service.Chat(context.Background(), port.AgentChatRequest{
				SessionID: fmt.Sprintf("pressure-session-%d", index),
				Message:   "并发测试",
				OwnerKey:  fmt.Sprintf("ip:pressure-%d", index),
			}, func(port.AgentChatEvent) error {
				atomic.AddInt32(&events, 1)
				return nil
			})
			errorsCh <- err
		}(index)
	}

	for index := 0; index < 2; index++ {
		select {
		case <-gateway.started:
		case <-time.After(2 * time.Second):
			close(gateway.hold)
			waitGroup.Wait()
			t.Fatal("bounded concurrent streams did not start")
		}
	}
	close(gateway.hold)
	waitGroup.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent Agent session failed: %v", err)
		}
	}
	if maxActive := atomic.LoadInt32(&gateway.maxActive); maxActive > 2 {
		t.Fatalf("active SSE slots = %d, want at most 2", maxActive)
	}
	if atomic.LoadInt32(&events) < sessions*3 || len(store.sessions) != sessions {
		t.Fatalf("events=%d sessions=%d, want at least %d events and %d sessions", events, len(store.sessions), sessions*3, sessions)
	}
}
