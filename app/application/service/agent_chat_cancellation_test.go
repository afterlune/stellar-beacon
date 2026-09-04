package service

import (
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"testing"
)

type cancellationAgentChatGateway struct {
	streamStarted chan struct{}
}

func (g *cancellationAgentChatGateway) Generate(context.Context, port.ChatRequest) (port.ChatResponse, error) {
	return port.ChatResponse{Text: "unused"}, nil
}

func (g *cancellationAgentChatGateway) Stream(ctx context.Context, _ port.ChatRequest, emit func(port.ChatStreamEvent) error) error {
	close(g.streamStarted)
	if err := emit(port.ChatStreamEvent{Kind: port.StreamEventMeta, RunID: "run-cancel"}); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

type cancellationAgentChatTools struct{}

func (cancellationAgentChatTools) Definitions() []port.ToolDefinition {
	return []port.ToolDefinition{{Name: port.PublicToolSearchArticles, Parameters: json.RawMessage(`{"type":"object"}`)}}
}

func (cancellationAgentChatTools) Execute(context.Context, string, json.RawMessage) (port.PublicAgentToolResult, error) {
	return port.PublicAgentToolResult{}, nil
}

func TestAgentChatPropagatesRequestCancellationToGateway(t *testing.T) {
	gateway := &cancellationAgentChatGateway{streamStarted: make(chan struct{})}
	store := &agentChatSessionStore{}
	service, err := NewAgentChatService(AgentChatServiceDeps{
		Chat: gateway,
		Profiles: agentChatProfileRepository{profile: port.AgentProfile{
			ID: "benetnasch-public", Name: "B", PromptVersion: "v1", SystemPrompt: "prompt", Enabled: true,
		}},
		Sessions:    store,
		Coordinator: newTestAgentSessionCoordinator(),
		Tools:       cancellationAgentChatTools{},
		Quota:       agentChatQuota{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		result <- service.Chat(ctx, port.AgentChatRequest{Message: "hello", OwnerKey: "ip:1"}, func(event port.AgentChatEvent) error {
			if event.Kind == port.StreamEventMeta && event.RunID == "run-cancel" {
				cancel()
			}
			return nil
		})
	}()
	select {
	case <-gateway.streamStarted:
	case <-ctx.Done():
	}
	if err := <-result; err == nil || err != context.Canceled {
		t.Fatalf("Chat() error = %v, want context.Canceled", err)
	}
	if store.saved {
		t.Fatal("canceled chat persisted a session")
	}
}
