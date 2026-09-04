package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	stderrors "errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type agentChatProfileRepository struct {
	profile port.AgentProfile
}

func (f agentChatProfileRepository) Get(context.Context, string) (port.AgentProfile, error) {
	return f.profile, nil
}

func (f agentChatProfileRepository) Save(context.Context, port.AgentProfile) error { return nil }

type agentChatSessionStore struct {
	session port.AgentSession
	found   bool
	saved   bool
}

func (s *agentChatSessionStore) Load(context.Context, string, string) (port.AgentSession, bool, error) {
	return s.session, s.found, nil
}

func (s *agentChatSessionStore) Save(_ context.Context, session port.AgentSession, _ string) error {
	s.session = session
	s.saved = true
	return nil
}

func (s *agentChatSessionStore) Delete(context.Context, string, string) error { return nil }

type testAgentSessionCoordinator struct {
	mu      sync.Mutex
	locks   map[string]chan struct{}
	waiters chan struct{}
}

func newTestAgentSessionCoordinator() *testAgentSessionCoordinator {
	return &testAgentSessionCoordinator{locks: make(map[string]chan struct{})}
}

func (c *testAgentSessionCoordinator) Acquire(ctx context.Context, ownerKey, sessionID string) (port.AgentSessionLease, error) {
	key := ownerKey + "\x00" + sessionID
	c.mu.Lock()
	lock, ok := c.locks[key]
	if !ok {
		lock = make(chan struct{}, 1)
		c.locks[key] = lock
	}
	c.mu.Unlock()
	select {
	case lock <- struct{}{}:
		return &testAgentSessionLease{lock: lock}, nil
	default:
		if c.waiters != nil {
			select {
			case c.waiters <- struct{}{}:
			default:
			}
		}
	}
	select {
	case lock <- struct{}{}:
		return &testAgentSessionLease{lock: lock}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type testAgentSessionLease struct {
	lock chan struct{}
}

func (l *testAgentSessionLease) KeepAlive(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	<-ctx.Done()
	return nil
}

func (l *testAgentSessionLease) Release() error {
	if l != nil && l.lock != nil {
		<-l.lock
	}
	return nil
}

type failingAgentSessionLease struct {
	err      error
	released chan struct{}
}

func (l *failingAgentSessionLease) KeepAlive(context.Context) error {
	return l.err
}

func (l *failingAgentSessionLease) Release() error {
	if l != nil && l.released != nil {
		close(l.released)
	}
	return nil
}

type leaseLossCoordinator struct {
	lease port.AgentSessionLease
}

func (c leaseLossCoordinator) Acquire(context.Context, string, string) (port.AgentSessionLease, error) {
	return c.lease, nil
}

type delayedLeaseLoss struct {
	err     error
	started chan struct{}
	lost    chan struct{}
	failed  chan struct{}
}

func (l *delayedLeaseLoss) KeepAlive(ctx context.Context) error {
	close(l.started)
	select {
	case <-l.lost:
		close(l.failed)
		return l.err
	case <-ctx.Done():
		return nil
	}
}

func (l *delayedLeaseLoss) Release() error { return nil }

func TestStartAgentSessionLeaseCancelsTurnWhenKeepAliveFails(t *testing.T) {
	leaseLost := stderrors.New("session lease lost")
	lease := &failingAgentSessionLease{err: leaseLost, released: make(chan struct{})}
	turnCtx, finish := startAgentSessionLease(context.Background(), lease)

	select {
	case <-turnCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("turn context was not canceled after keepalive failure")
	}
	if err := finish(turnCtx.Err()); !stderrors.Is(err, leaseLost) {
		t.Fatalf("finish() error = %v, want keepalive error", err)
	}
	select {
	case <-lease.released:
	case <-time.After(time.Second):
		t.Fatal("lease was not released")
	}
}

func TestAgentChatDoesNotEmitDoneAfterSessionLeaseIsLost(t *testing.T) {
	leaseLost := stderrors.New("session lease lost")
	lease := &delayedLeaseLoss{
		err:     leaseLost,
		started: make(chan struct{}),
		lost:    make(chan struct{}),
		failed:  make(chan struct{}),
	}
	store := &agentChatSessionStore{}
	gateway := &agentChatGateway{stream: func(_ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
		<-lease.started
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: "run-1"}); err != nil {
			return err
		}
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, RunID: "run-1", Text: "回答"}); err != nil {
			return err
		}
		close(lease.lost)
		<-lease.failed
		return nil
	}}
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", PromptVersion: "v1", SystemPrompt: "基础人设", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: leaseLossCoordinator{lease: lease},
		Tools:       &agentChatTools{},
		Quota:       agentChatQuota{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var events []port.AgentChatEvent
	err = service.Chat(context.Background(), port.AgentChatRequest{Message: "查询", OwnerKey: "ip:1"}, func(event port.AgentChatEvent) error {
		events = append(events, event)
		return nil
	})
	if !stderrors.Is(err, leaseLost) {
		t.Fatalf("Chat() error = %v, want lease loss", err)
	}
	if containsAgentEvent(events, port.StreamEventDone) {
		t.Fatalf("events = %#v, want no terminal done event after lease loss", events)
	}
}

type agentChatTools struct {
	called []string
}

type agentChatQuota struct{}

func (agentChatQuota) Allow(context.Context, string, bool) (port.AgentQuotaDecision, error) {
	return port.AgentQuotaDecision{Allowed: true}, nil
}

func (t *agentChatTools) Definitions() []port.ToolDefinition {
	return []port.ToolDefinition{{Name: port.PublicToolSearchArticles, Parameters: json.RawMessage(`{"type":"object"}`)}}
}

func (t *agentChatTools) Execute(_ context.Context, name string, _ json.RawMessage) (port.PublicAgentToolResult, error) {
	t.called = append(t.called, name)
	return port.PublicAgentToolResult{
		Content:   `{"results":[{"articleId":1,"title":"公开文章"}]}`,
		Citations: []port.Citation{{ArticleID: 1, Title: "公开文章", URL: "/articles/1"}},
	}, nil
}

type agentChatGateway struct {
	stream        func(port.ChatRequest, func(port.ChatStreamEvent) error) error
	gen           port.ChatResponse
	streamRequest port.ChatRequest
	last          port.ChatRequest
}

func (g *agentChatGateway) Generate(_ context.Context, request port.ChatRequest) (port.ChatResponse, error) {
	g.last = request
	return g.gen, nil
}

func (g *agentChatGateway) Stream(_ context.Context, request port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	g.streamRequest = request
	if g.stream != nil {
		return g.stream(request, emit)
	}
	return nil
}

func newAgentChatServiceForTest(gateway *agentChatGateway, store *agentChatSessionStore, tools *agentChatTools) *MyAgentChatService {
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", Name: "Benetnasch", PromptVersion: "v1", SystemPrompt: "基础人设", Opening: "你好", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: newTestAgentSessionCoordinator(),
		Tools:       tools,
		Quota:       agentChatQuota{},
	})
	if err != nil {
		panic(err)
	}
	return service
}

func TestAgentChatStreamsAnswerAndStoresBoundedConversation(t *testing.T) {
	store := &agentChatSessionStore{}
	tools := &agentChatTools{}
	gateway := &agentChatGateway{stream: func(request port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
		if len(request.Tools) != 1 || !strings.Contains(request.Messages[0].Content, "运行时安全契约") {
			t.Fatalf("unsafe or missing public chat request: %#v", request)
		}
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: "run-1", Provider: "test", Model: "mock"}); err != nil {
			return err
		}
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, RunID: "run-1", Text: "你好"}); err != nil {
			return err
		}
		return emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: "run-1", Provider: "test", Model: "mock", Usage: &port.TokenUsage{TotalTokens: 3}})
	}}
	service := newAgentChatServiceForTest(gateway, store, tools)
	var events []port.AgentChatEvent
	if err := service.Chat(context.Background(), port.AgentChatRequest{Message: "你好", OwnerKey: "ip:1"}, func(event port.AgentChatEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !store.saved || len(store.session.Messages) != 2 || store.session.Messages[1].Content != "你好" {
		t.Fatalf("saved session = %#v", store.session)
	}
	if len(events) < 3 || events[0].Kind != port.StreamEventMeta || events[len(events)-1].Kind != port.StreamEventDone {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Opening != "你好" || events[len(events)-1].RunID != "run-1" {
		t.Fatalf("events missing opening/run identity: %#v", events)
	}
	if gateway.streamRequest.MaxOutputTokens != defaultAgentMaxOutputTokens {
		t.Fatalf("stream MaxOutputTokens = %d, want %d", gateway.streamRequest.MaxOutputTokens, defaultAgentMaxOutputTokens)
	}
}

func TestAgentChatUsesConfiguredOutputBudgetForToolRound(t *testing.T) {
	store := &agentChatSessionStore{}
	tools := &agentChatTools{}
	gateway := &agentChatGateway{
		stream: func(_ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			return emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: "tool-run", ToolCalls: []port.ToolCall{{
				ID: "call-1", Name: port.PublicToolSearchArticles, Arguments: json.RawMessage(`{"query":"公开"}`),
			}}})
		},
		gen: port.ChatResponse{RunID: "answer-run", Provider: "test", Text: "根据公开文章回答"},
	}
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", PromptVersion: "v1", SystemPrompt: "基础人设", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: newTestAgentSessionCoordinator(),
		Tools:       tools,
		Quota:       agentChatQuota{},
		Limits:      AgentChatLimits{MaxOutputTokens: 777},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Chat(context.Background(), port.AgentChatRequest{Message: "找文章", OwnerKey: "ip:1"}, func(port.AgentChatEvent) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if gateway.streamRequest.MaxOutputTokens != 777 || gateway.last.MaxOutputTokens != 777 {
		t.Fatalf("configured output budget not propagated: stream=%d generate=%d", gateway.streamRequest.MaxOutputTokens, gateway.last.MaxOutputTokens)
	}
}

type serializedAgentSessionStore struct {
	mu       sync.Mutex
	session  port.AgentSession
	found    bool
	loadSize []int
}

func (s *serializedAgentSessionStore) Load(context.Context, string, string) (port.AgentSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadSize = append(s.loadSize, len(s.session.Messages))
	return s.session, s.found, nil
}

func (s *serializedAgentSessionStore) Save(_ context.Context, session port.AgentSession, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = session
	s.found = true
	return nil
}

func (s *serializedAgentSessionStore) Delete(context.Context, string, string) error { return nil }

type serializedAgentChatGateway struct {
	mu           sync.Mutex
	calls        int
	firstStarted chan struct{}
	releaseFirst chan struct{}
}

func (g *serializedAgentChatGateway) Generate(context.Context, port.ChatRequest) (port.ChatResponse, error) {
	return port.ChatResponse{Text: "unused"}, nil
}

func (g *serializedAgentChatGateway) Stream(ctx context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	g.mu.Lock()
	g.calls++
	call := g.calls
	g.mu.Unlock()
	runID := "run-" + string(rune('0'+call))
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: runID}); err != nil {
		return err
	}
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, RunID: runID, Text: "回答"}); err != nil {
		return err
	}
	if call == 1 {
		close(g.firstStarted)
		select {
		case <-g.releaseFirst:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: runID})
}

func TestAgentChatSerializesConcurrentTurnsForOneSession(t *testing.T) {
	store := &serializedAgentSessionStore{}
	coordinator := newTestAgentSessionCoordinator()
	coordinator.waiters = make(chan struct{}, 1)
	gateway := &serializedAgentChatGateway{firstStarted: make(chan struct{}), releaseFirst: make(chan struct{})}
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", PromptVersion: "v1", SystemPrompt: "基础人设", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: coordinator,
		Tools:       &agentChatTools{},
		Quota:       agentChatQuota{},
	})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	chat := func() {
		results <- service.Chat(context.Background(), port.AgentChatRequest{
			SessionID: "session-1",
			Message:   "并发消息",
			OwnerKey:  "ip:one",
		}, func(port.AgentChatEvent) error { return nil })
	}
	go chat()
	select {
	case <-gateway.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first Agent turn did not start")
	}
	go chat()
	select {
	case <-coordinator.waiters:
	case <-time.After(time.Second):
		close(gateway.releaseFirst)
		<-results
		t.Fatal("second Agent turn did not wait for the session lease")
	}
	close(gateway.releaseFirst)
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatalf("Agent turn %d failed: %v", i+1, err)
		}
	}
	store.mu.Lock()
	loads := append([]int(nil), store.loadSize...)
	store.mu.Unlock()
	if len(loads) != 2 || loads[0] != 0 || loads[1] != 2 {
		t.Fatalf("session load sizes = %#v, want [0 2]", loads)
	}
}

func TestAgentChatExecutesOnlyRegisteredReadToolAndEmitsCitation(t *testing.T) {
	store := &agentChatSessionStore{}
	tools := &agentChatTools{}
	gateway := &agentChatGateway{
		stream: func(_ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
			if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: "tool-run", Provider: "test"}); err != nil {
				return err
			}
			return emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: "tool-run", ToolCalls: []port.ToolCall{{ID: "call-1", Name: port.PublicToolSearchArticles, Arguments: json.RawMessage(`{"query":"公开"}`)}}})
		},
		gen: port.ChatResponse{RunID: "answer-run", Provider: "test", Text: "根据公开文章回答"},
	}
	service := newAgentChatServiceForTest(gateway, store, tools)
	var events []port.AgentChatEvent
	if err := service.Chat(context.Background(), port.AgentChatRequest{SessionID: "session-1", Message: "找文章", OwnerKey: "ip:1"}, func(event port.AgentChatEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(tools.called) != 1 || tools.called[0] != port.PublicToolSearchArticles {
		t.Fatalf("tool calls = %#v", tools.called)
	}
	if len(gateway.last.Tools) != 0 || gateway.last.ToolChoice != port.ToolChoiceNone {
		t.Fatalf("final generation retained tools: %#v", gateway.last)
	}
	if !containsAgentEvent(events, port.StreamEventCitation) || !containsAgentEvent(events, port.StreamEventDone) {
		t.Fatalf("events = %#v", events)
	}
	if !store.saved || store.session.Messages[1].Content != "根据公开文章回答" {
		t.Fatalf("saved tool answer = %#v", store.session)
	}
}

func TestAgentChatDoesNotForwardProviderCitation(t *testing.T) {
	store := &agentChatSessionStore{}
	gateway := &agentChatGateway{stream: func(_ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventCitation, RunID: "run-1", Citation: &port.Citation{
			ArticleID: 7,
			Title:     " 公开文章 ",
			URL:       "javascript:alert(1)",
			Score:     0.8,
		}}); err != nil {
			return err
		}
		if err := emit(port.ChatStreamEvent{Kind: port.StreamEventDelta, RunID: "run-1", Text: "根据资料回答"}); err != nil {
			return err
		}
		return emit(port.ChatStreamEvent{Kind: port.StreamEventDone, RunID: "run-1"})
	}}
	service := newAgentChatServiceForTest(gateway, store, &agentChatTools{})
	var citations []*port.Citation
	if err := service.Chat(context.Background(), port.AgentChatRequest{Message: "查询", OwnerKey: "ip:1"}, func(event port.AgentChatEvent) error {
		if event.Kind == port.StreamEventCitation {
			citations = append(citations, event.Citation)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(citations) != 0 {
		t.Fatalf("provider citations = %#v, want no provider citation on public stream", citations)
	}
}

func TestAgentChatStopsStreamingWhenToolCallLimitIsExceeded(t *testing.T) {
	store := &agentChatSessionStore{}
	tools := &agentChatTools{}
	callbackStopped := false
	gateway := &agentChatGateway{stream: func(_ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
		calls := make([]port.ToolCall, 0, 5)
		for index := 0; index < 5; index++ {
			calls = append(calls, port.ToolCall{ID: "call-" + string(rune('1'+index)), Name: port.PublicToolSearchArticles})
		}
		err := emit(port.ChatStreamEvent{Kind: port.StreamEventToolCall, RunID: "run-1", ToolCalls: calls})
		if err == nil {
			t.Fatal("stream callback accepted too many tool calls")
		}
		callbackStopped = true
		return err
	}}
	service := newAgentChatServiceForTest(gateway, store, tools)
	err := service.Chat(context.Background(), port.AgentChatRequest{Message: "查询", OwnerKey: "ip:1"}, func(port.AgentChatEvent) error {
		return nil
	})
	if !callbackStopped || !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("stream limit error = %v, callbackStopped=%v", err, callbackStopped)
	}
	if len(tools.called) != 0 || store.saved {
		t.Fatalf("tool calls or session save occurred after stream limit: calls=%#v saved=%v", tools.called, store.saved)
	}
}

func containsAgentEvent(events []port.AgentChatEvent, kind port.StreamEventKind) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}

func TestRuntimeSystemPromptUsesServerOwnedRhythmVariant(t *testing.T) {
	prompt := runtimeSystemPrompt("基础人设", port.AgentRhythmSnapshot{Phase: port.AgentRhythmNight, Timezone: "Asia/Shanghai"}, map[port.AgentRhythmPhase]string{
		port.AgentRhythmNight: "夜间测试变体",
	})
	if !strings.Contains(prompt, "夜间测试变体") || !strings.Contains(prompt, "由服务端时钟决定") {
		t.Fatalf("prompt = %q", prompt)
	}
}
