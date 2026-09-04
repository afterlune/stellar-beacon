package einoadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/mock"
)

func TestProviderGenerateContractsUseExpectedHTTPProtocols(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()

	tests := []struct {
		name       string
		provider   string
		protocol   port.ProviderProtocol
		baseURL    string
		newAdapter func(context.Context, Config) (*ChatAdapter, error)
	}{
		{
			name:       "openai responses",
			provider:   "openai",
			protocol:   port.ProviderProtocolOpenAIResponses,
			baseURL:    server.URL() + "/v1",
			newAdapter: NewOpenAIResponses,
		},
		{
			name:       "anthropic messages",
			provider:   "anthropic",
			protocol:   port.ProviderProtocolAnthropicMessages,
			baseURL:    server.URL(),
			newAdapter: NewAnthropicMessages,
		},
		{
			name:       "openai chat completions",
			provider:   "sglang",
			protocol:   port.ProviderProtocolOpenAIChatCompletions,
			baseURL:    server.URL() + "/v1",
			newAdapter: NewOpenAIChatCompletions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := tt.newAdapter(context.Background(), Config{
				Provider:      tt.provider,
				Protocol:      tt.protocol,
				Model:         "contract-model",
				APIKey:        "test-key",
				BaseURL:       tt.baseURL,
				Timeout:       5 * time.Second,
				MaxConcurrent: 1,
			})
			if err != nil {
				t.Fatalf("new adapter error = %v", err)
			}
			response, err := adapter.Generate(context.Background(), port.ChatRequest{
				Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
			})
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if response.Text != "mock response" {
				t.Fatalf("response text = %q, want mock response", response.Text)
			}
			if response.Provider != tt.provider || response.Protocol != tt.protocol || response.Model != "contract-model" || response.RunID == "" {
				t.Fatalf("response metadata = %#v", response)
			}
		})
	}

	requests := server.Requests()
	if len(requests) != len(tests) {
		t.Fatalf("mock request count = %d, want %d", len(requests), len(tests))
	}
	seen := make(map[string]bool, len(requests))
	for _, request := range requests {
		seen[request.Protocol] = true
		if request.Path == "" {
			t.Fatal("mock request path is empty")
		}
	}
	for _, protocol := range []string{"openai_responses", "anthropic_messages", "openai_chat_completions"} {
		if !seen[protocol] {
			t.Fatalf("protocol %q was not exercised", protocol)
		}
	}
}

func TestOpenAIChatCompletionsContractMapsDeepSeekVisionImage(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()

	adapter, err := NewOpenAIChatCompletions(context.Background(), Config{
		Provider:      "openai",
		Model:         "deepseek-v4-flash-vision-exp",
		APIKey:        "test-key",
		BaseURL:       server.URL() + "/v1",
		Timeout:       5 * time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatalf("new DeepSeek vision adapter error = %v", err)
	}

	response, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{
			Role:    port.ChatRoleUser,
			Content: "请描述图片",
			ContentParts: []port.ChatMessagePart{{
				Type:   port.ChatMessagePartTypeImageURL,
				URL:    "https://example.com/image.png",
				Detail: "low",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if response.Text != "mock response" {
		t.Fatalf("response text = %q, want mock response", response.Text)
	}

	requests := server.Requests()
	if len(requests) != 1 {
		t.Fatalf("mock request count = %d, want 1", len(requests))
	}
	var payload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL struct {
					URL    string `json:"url"`
					Detail string `json:"detail"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(requests[0].Body, &payload); err != nil {
		t.Fatalf("decode chat completions request: %v; body=%s", err, requests[0].Body)
	}
	if payload.Model != "deepseek-v4-flash-vision-exp" || len(payload.Messages) != 1 || payload.Messages[0].Role != "user" || len(payload.Messages[0].Content) != 2 {
		t.Fatalf("unexpected vision request envelope: %#v", payload)
	}
	image := payload.Messages[0].Content[1]
	if image.Type != "image_url" || image.ImageURL.URL != "https://example.com/image.png" || image.ImageURL.Detail != "low" {
		t.Fatalf("unexpected vision request image block: %#v", image)
	}
}

func TestProviderContractRejectsMissingRequiredCredentials(t *testing.T) {
	_, err := NewOpenAIResponses(context.Background(), Config{Model: "model"})
	if err == nil {
		t.Fatal("OpenAI Responses adapter accepted missing API key")
	}
	_, err = NewAnthropicMessages(context.Background(), Config{Model: "model"})
	if err == nil {
		t.Fatal("Anthropic adapter accepted missing API key")
	}
	_, err = NewOpenAIChatCompletions(context.Background(), Config{Provider: "openai", Model: "model"})
	if err == nil {
		t.Fatal("OpenAI Chat Completions adapter accepted missing API key")
	}
	if _, err = NewOpenAIChatCompletions(context.Background(), Config{Model: "model"}); err != nil {
		t.Fatalf("SGLang-compatible adapter rejected empty optional API key: %v", err)
	}
}

func TestProviderStreamContractsNormalizeProtocolDeltas(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()

	tests := []struct {
		name       string
		provider   string
		protocol   port.ProviderProtocol
		baseURL    string
		newAdapter func(context.Context, Config) (*ChatAdapter, error)
	}{
		{
			name:       "openai responses",
			provider:   "openai",
			protocol:   port.ProviderProtocolOpenAIResponses,
			baseURL:    server.URL() + "/v1",
			newAdapter: NewOpenAIResponses,
		},
		{
			name:       "anthropic messages",
			provider:   "anthropic",
			protocol:   port.ProviderProtocolAnthropicMessages,
			baseURL:    server.URL(),
			newAdapter: NewAnthropicMessages,
		},
		{
			name:       "openai chat completions",
			provider:   "sglang",
			protocol:   port.ProviderProtocolOpenAIChatCompletions,
			baseURL:    server.URL() + "/v1",
			newAdapter: NewOpenAIChatCompletions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := tt.newAdapter(context.Background(), Config{
				Provider:      tt.provider,
				Protocol:      tt.protocol,
				Model:         "contract-model",
				APIKey:        "test-key",
				BaseURL:       tt.baseURL,
				Timeout:       5 * time.Second,
				MaxConcurrent: 1,
			})
			if err != nil {
				t.Fatalf("new adapter error = %v", err)
			}
			var textOutput strings.Builder
			var events []port.ChatStreamEvent
			err = adapter.Stream(context.Background(), port.ChatRequest{
				Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
			}, func(event port.ChatStreamEvent) error {
				events = append(events, event)
				if event.Kind == port.StreamEventDelta {
					textOutput.WriteString(event.Text)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Stream() error = %v", err)
			}
			if textOutput.String() != "mock stream" {
				t.Fatalf("stream text = %q, want mock stream", textOutput.String())
			}
			if len(events) < 3 || events[0].Kind != port.StreamEventMeta || events[len(events)-1].Kind != port.StreamEventDone {
				t.Fatalf("stream events = %#v, want meta and done", events)
			}
			for _, event := range events {
				if event.RunID == "" || event.RunID != events[0].RunID {
					t.Fatalf("event RunID is not stable: %#v", events)
				}
			}
		})
	}
}

func TestSGLangToolCallingContract(t *testing.T) {
	server := mock.NewServer(func(_ context.Context, request mock.Request) (mock.Response, error) {
		if request.Path != "/v1/chat/completions" {
			return mock.Response{Status: http.StatusNotFound, Body: map[string]any{"error": "unexpected path"}}, nil
		}
		if requestWantsStreamForTest(request.Body) {
			return mock.Response{StreamBody: sglangToolStreamBody()}, nil
		}
		return mock.Response{Body: map[string]any{
			"id": "sglang-tool-response",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []map[string]any{{
						"id":   "call-sglang",
						"type": "function",
						"function": map[string]string{
							"name":      "search",
							"arguments": `{"query":"eino"}`,
						},
					}},
				},
				"finish_reason": "tool_calls",
			}},
		}}, nil
	})
	defer server.Close()

	adapter, err := NewOpenAIChatCompletions(context.Background(), Config{
		Provider:      "sglang",
		Model:         "contract-model",
		APIKey:        "test-key",
		BaseURL:       server.URL() + "/v1",
		Timeout:       5 * time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatalf("new SGLang adapter error = %v", err)
	}
	request := port.ChatRequest{
		Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "search Eino"}},
		Tools: []port.ToolDefinition{{
			Name:        "search",
			Description: "Search public documents",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		}},
		ToolChoice: port.ToolChoiceRequired,
	}
	response, err := adapter.Generate(context.Background(), request)
	if err != nil {
		t.Fatalf("SGLang Generate() error = %v", err)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call-sglang" || response.ToolCalls[0].Name != "search" || string(response.ToolCalls[0].Arguments) != `{"query":"eino"}` {
		t.Fatalf("SGLang tool response = %#v", response.ToolCalls)
	}

	var events []port.ChatStreamEvent
	if err := adapter.Stream(context.Background(), request, func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("SGLang Stream() error = %v", err)
	}
	if len(events) != 4 || events[0].Kind != port.StreamEventMeta || events[1].Kind != port.StreamEventToolCall || events[2].Kind != port.StreamEventToolCall || events[3].Kind != port.StreamEventDone {
		t.Fatalf("SGLang stream events = %#v", events)
	}
	if len(events[1].ToolCalls) != 1 || events[1].ToolCalls[0].ID != "call-sglang" || string(events[1].ToolCalls[0].Arguments) != `{"query":"` {
		t.Fatalf("SGLang first tool call delta = %#v", events[1])
	}
	if len(events[2].ToolCalls) != 1 || string(events[2].ToolCalls[0].Arguments) != `eino"}` || events[3].ToolCalls[0].Name != "search" {
		t.Fatalf("SGLang stream tool calls = %#v", events)
	}

	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("SGLang request count = %d, want 2", len(requests))
	}
	for _, recorded := range requests {
		var payload struct {
			Tools      []json.RawMessage `json:"tools"`
			ToolChoice json.RawMessage   `json:"tool_choice"`
			Stream     bool              `json:"stream"`
		}
		if err := json.Unmarshal(recorded.Body, &payload); err != nil {
			t.Fatalf("decode SGLang request: %v", err)
		}
		if len(payload.Tools) != 1 || !sglangRequiredToolChoice(payload.ToolChoice) {
			t.Fatalf("SGLang tool request = %#v", payload)
		}
	}
}

func sglangToolStreamBody() string {
	return `data: {"id":"sglang-tool-stream","object":"chat.completion.chunk","created":1,"model":"contract-model","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call-sglang","type":"function","function":{"name":"search","arguments":"{\"query\":\""}}]},"finish_reason":null}]}

data: {"id":"sglang-tool-stream","object":"chat.completion.chunk","created":1,"model":"contract-model","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"eino\"}"}}]},"finish_reason":null}]}

data: {"id":"sglang-tool-stream","object":"chat.completion.chunk","created":1,"model":"contract-model","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`
}

func sglangRequiredToolChoice(raw json.RawMessage) bool {
	var choice any
	if json.Unmarshal(raw, &choice) != nil {
		return false
	}
	if value, ok := choice.(string); ok {
		return value == "required"
	}
	object, ok := choice.(map[string]any)
	if !ok || object["type"] != "function" {
		return false
	}
	function, ok := object["function"].(map[string]any)
	name, nameOK := function["name"].(string)
	return ok && nameOK && name == "search"
}

func TestSGLangErrorContractRetries429ButNot400(t *testing.T) {
	t.Run("429 is retried", func(t *testing.T) {
		var calls atomic.Int32
		server := mock.NewServer(func(_ context.Context, _ mock.Request) (mock.Response, error) {
			if calls.Add(1) == 1 {
				return mock.Response{Status: http.StatusTooManyRequests, Body: map[string]any{"error": map[string]string{"message": "busy"}}}, nil
			}
			return mock.Response{Body: map[string]any{
				"id":      "ok",
				"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": "recovered"}, "finish_reason": "stop"}},
			}}, nil
		})
		defer server.Close()
		adapter, err := NewOpenAIChatCompletions(context.Background(), Config{
			Provider:      "sglang",
			Model:         "contract-model",
			BaseURL:       server.URL() + "/v1",
			Timeout:       5 * time.Second,
			MaxRetries:    1,
			MaxConcurrent: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		response, err := adapter.Generate(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}}})
		if err != nil || response.Text != "recovered" {
			t.Fatalf("Generate() response=%#v error=%v", response, err)
		}
		if calls.Load() != 2 {
			t.Fatalf("SGLang calls = %d, want 2", calls.Load())
		}
	})

	t.Run("400 is not retried", func(t *testing.T) {
		var calls atomic.Int32
		server := mock.NewServer(func(_ context.Context, _ mock.Request) (mock.Response, error) {
			calls.Add(1)
			return mock.Response{Status: http.StatusBadRequest, Body: map[string]any{"error": map[string]string{"message": "bad request"}}}, nil
		})
		defer server.Close()
		adapter, err := NewOpenAIChatCompletions(context.Background(), Config{
			Provider:      "sglang",
			Model:         "contract-model",
			BaseURL:       server.URL() + "/v1",
			MaxRetries:    2,
			MaxConcurrent: 1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := adapter.Generate(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}}}); err == nil {
			t.Fatal("Generate() error = nil, want bad request")
		}
		if calls.Load() != 1 {
			t.Fatalf("SGLang calls = %d, want 1", calls.Load())
		}
	})
}

func requestWantsStreamForTest(body json.RawMessage) bool {
	var request struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &request) == nil && request.Stream
}
