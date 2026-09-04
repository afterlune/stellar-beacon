package einoadapter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go/v3"
)

type scriptedModel struct {
	generate         func(int) (*schema.AgenticMessage, error)
	stream           func(int) (*schema.StreamReader[*schema.AgenticMessage], error)
	generateCalls    int
	streamCalls      int
	generateMessages []*schema.AgenticMessage
	streamMessages   []*schema.AgenticMessage
	generateOptions  *model.Options
	streamOptions    *model.Options
}

func (m *scriptedModel) Generate(_ context.Context, messages []*schema.AgenticMessage, options ...model.Option) (*schema.AgenticMessage, error) {
	m.generateCalls++
	m.generateMessages = messages
	m.generateOptions = model.GetCommonOptions(&model.Options{}, options...)
	if m.generate == nil {
		return nil, errors.New("generate script is not configured")
	}
	return m.generate(m.generateCalls)
}

func (m *scriptedModel) Stream(_ context.Context, messages []*schema.AgenticMessage, options ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	m.streamCalls++
	m.streamMessages = messages
	m.streamOptions = model.GetCommonOptions(&model.Options{}, options...)
	if m.stream == nil {
		return nil, errors.New("stream script is not configured")
	}
	return m.stream(m.streamCalls)
}

func testAdapter(t *testing.T, model agenticModel, retries int) *ChatAdapter {
	t.Helper()
	adapter, err := newChatAdapter(model, Config{
		Provider:      "test-provider",
		Protocol:      port.ProviderProtocolOpenAIResponses,
		Model:         "test-model",
		Timeout:       time.Second,
		MaxRetries:    retries,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatalf("newChatAdapter() error = %v", err)
	}
	return adapter
}

type recordingObserver struct {
	runs []port.AgentRun
}

func (o *recordingObserver) ObserveAIRun(_ context.Context, run port.AgentRun) {
	o.runs = append(o.runs, run)
}

func testRequest() port.ChatRequest {
	return port.ChatRequest{
		Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
	}
}

func textMessage(text string) *schema.AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.AssistantGenText{Text: text}),
		},
	}
}

func toolCallMessage(id, name, arguments string) *schema.AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.FunctionToolCall{
				CallID:    id,
				Name:      name,
				Arguments: arguments,
			}),
		},
	}
}

func streamFrom(messages ...*schema.AgenticMessage) *schema.StreamReader[*schema.AgenticMessage] {
	return schema.StreamReaderFromArray(messages)
}

func streamWithError(chunks []*schema.AgenticMessage, streamErr error) *schema.StreamReader[*schema.AgenticMessage] {
	reader, writer := schema.Pipe[*schema.AgenticMessage](len(chunks) + 1)
	go func() {
		defer writer.Close()
		for _, chunk := range chunks {
			writer.Send(chunk, nil)
		}
		writer.Send(nil, streamErr)
	}()
	return reader
}

func TestChatAdapterGenerateKeepsLocalRunIDSeparateFromProviderID(t *testing.T) {
	model := &scriptedModel{
		generate: func(call int) (*schema.AgenticMessage, error) {
			if call == 1 {
				return nil, errors.New("temporary upstream failure")
			}
			return &schema.AgenticMessage{
				Role:          schema.AgenticRoleTypeAssistant,
				ContentBlocks: textMessage("ok").ContentBlocks,
				ResponseMeta: &schema.AgenticResponseMeta{
					Extension: &agenticopenai.ChatResponseMetaExtension{ID: "provider-run-1"},
				},
			}, nil
		},
	}
	response, err := testAdapter(t, model, 1).Generate(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if model.generateCalls != 2 {
		t.Fatalf("Generate() calls = %d, want 2", model.generateCalls)
	}
	if response.RunID == "" || response.RunID == "provider-run-1" {
		t.Fatalf("RunID = %q, want a local non-provider ID", response.RunID)
	}
	if response.ProviderRunID != "provider-run-1" {
		t.Fatalf("ProviderRunID = %q, want provider-run-1", response.ProviderRunID)
	}
}

func TestChatAdapterRecordsCompletedGenerateRun(t *testing.T) {
	observer := &recordingObserver{}
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return textMessage("observed"), nil
		},
	}
	adapter, err := newChatAdapter(model, Config{
		Provider:      "test-provider",
		Protocol:      port.ProviderProtocolOpenAIResponses,
		Model:         "test-model",
		Timeout:       time.Second,
		MaxConcurrent: 1,
		Observer:      observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := adapter.Generate(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(observer.runs) != 1 {
		t.Fatalf("observed runs = %d, want 1", len(observer.runs))
	}
	run := observer.runs[0]
	if run.ID != response.RunID || run.Provider != "test-provider" || run.Model != "test-model" || run.Status != port.AIRunSucceeded {
		t.Fatalf("observed run = %#v", run)
	}
	if run.CompletedAt.IsZero() || run.StartedAt.IsZero() || run.Duration < 0 {
		t.Fatalf("observed timing = %#v", run)
	}
}

func TestChatAdapterStreamRetriesOnlyBeforeFirstDelta(t *testing.T) {
	model := &scriptedModel{
		stream: func(call int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			if call == 1 {
				return streamWithError(nil, errors.New("temporary stream failure")), nil
			}
			return streamFrom(textMessage("recovered")), nil
		},
	}
	adapter := testAdapter(t, model, 1)
	var events []port.ChatStreamEvent
	if err := adapter.Stream(context.Background(), testRequest(), func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if model.streamCalls != 2 {
		t.Fatalf("Stream() calls = %d, want 2", model.streamCalls)
	}
	if len(events) != 3 || events[0].Kind != port.StreamEventMeta || events[1].Kind != port.StreamEventDelta || events[2].Kind != port.StreamEventDone {
		t.Fatalf("events = %#v, want meta, delta, done", events)
	}
	if events[0].RunID == "" || events[0].RunID != events[1].RunID || events[1].RunID != events[2].RunID {
		t.Fatalf("event RunIDs are not stable: %#v", events)
	}
}

func TestChatAdapterStreamNeverRetriesAfterPartialOutput(t *testing.T) {
	model := &scriptedModel{
		stream: func(int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			return streamWithError([]*schema.AgenticMessage{textMessage("partial")}, errors.New("failure after output")), nil
		},
	}
	adapter := testAdapter(t, model, 2)
	var events []port.ChatStreamEvent
	err := adapter.Stream(context.Background(), testRequest(), func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err == nil {
		t.Fatal("Stream() error = nil, want provider failure")
	}
	if model.streamCalls != 1 {
		t.Fatalf("Stream() calls = %d, want 1 after partial output", model.streamCalls)
	}
	if len(events) != 2 || events[1].Kind != port.StreamEventDelta || events[1].Text != "partial" {
		t.Fatalf("events = %#v, want meta and partial delta", events)
	}
}

func TestChatAdapterStreamRecordsFirstTokenTiming(t *testing.T) {
	observer := &recordingObserver{}
	model := &scriptedModel{
		stream: func(int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			return streamFrom(textMessage("first token")), nil
		},
	}
	adapter, err := newChatAdapter(model, Config{
		Provider:      "test-provider",
		Protocol:      port.ProviderProtocolOpenAIResponses,
		Model:         "test-model",
		Timeout:       time.Second,
		MaxConcurrent: 1,
		Observer:      observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.Stream(context.Background(), testRequest(), func(port.ChatStreamEvent) error { return nil }); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(observer.runs) != 1 {
		t.Fatalf("observed runs = %d, want 1", len(observer.runs))
	}
	run := observer.runs[0]
	if run.Status != port.AIRunSucceeded || run.FirstTokenAt.IsZero() || run.FirstTokenLatency < 0 || run.Duration < run.FirstTokenLatency {
		t.Fatalf("observed stream timing = %#v", run)
	}
}

func TestChatAdapterCircuitBreakerFailsFastAfterRetryableFailures(t *testing.T) {
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return nil, errors.New("temporary upstream failure")
		},
	}
	adapter, err := newChatAdapter(model, Config{
		Provider:                "test-provider",
		Protocol:                port.ProviderProtocolOpenAIResponses,
		Model:                   "test-model",
		Timeout:                 time.Second,
		MaxConcurrent:           1,
		CircuitFailureThreshold: 2,
		CircuitResetTimeout:     time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := adapter.Generate(context.Background(), testRequest()); err == nil {
			t.Fatal("Generate() error = nil, want provider failure")
		}
	}
	if _, err := adapter.Generate(context.Background(), testRequest()); !apperrors.IsAICode(err, apperrors.AICodeCircuitOpen) {
		t.Fatalf("third Generate() error = %v, want circuit open", err)
	}
	if model.generateCalls != 2 {
		t.Fatalf("provider calls = %d, want 2 after circuit opened", model.generateCalls)
	}
}

func TestChatAdapterMapsToolCallsAndToolResults(t *testing.T) {
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return toolCallMessage("call-1", "search", `{"query":"eino"}`), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	response, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{
			{Role: port.ChatRoleUser, Content: "find Eino"},
			{
				Role: port.ChatRoleAssistant,
				ToolCalls: []port.ToolCall{{
					ID:        "call-0",
					Name:      "search",
					Arguments: json.RawMessage(`{"query":"go"}`),
				}},
			},
			{Role: port.ChatRoleTool, Name: "search", ToolCallID: "call-0", Content: `{"hits":1}`},
		},
		Tools: []port.ToolDefinition{{
			Name:        "search",
			Description: "Search public documents",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		}},
		ToolChoice: port.ToolChoiceRequired,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call-1" || response.ToolCalls[0].Name != "search" || string(response.ToolCalls[0].Arguments) != `{"query":"eino"}` {
		t.Fatalf("tool calls = %#v", response.ToolCalls)
	}
	if len(model.generateMessages) != 3 || model.generateMessages[1].Role != schema.AgenticRoleTypeAssistant || model.generateMessages[2].Role != schema.AgenticRoleTypeUser {
		t.Fatalf("provider messages = %#v", model.generateMessages)
	}
	if len(model.generateMessages[1].ContentBlocks) != 1 || model.generateMessages[1].ContentBlocks[0].FunctionToolCall == nil {
		t.Fatalf("assistant tool call message = %#v", model.generateMessages[1])
	}
	if len(model.generateMessages[2].ContentBlocks) != 1 || model.generateMessages[2].ContentBlocks[0].FunctionToolResult == nil {
		t.Fatalf("tool result message = %#v", model.generateMessages[2])
	}
	if model.generateOptions == nil || len(model.generateOptions.Tools) != 1 || model.generateOptions.Tools[0].Name != "search" {
		t.Fatalf("provider tools = %#v", model.generateOptions)
	}
	if model.generateOptions.AgenticToolChoice == nil || model.generateOptions.AgenticToolChoice.Type != schema.ToolChoiceForced {
		t.Fatalf("provider tool choice = %#v", model.generateOptions.AgenticToolChoice)
	}
}

func TestToAgenticMessagesMapsVisionImageParts(t *testing.T) {
	messages, err := toAgenticMessages([]port.ChatMessage{{
		Role:    port.ChatRoleUser,
		Content: "请描述这张图",
		ContentParts: []port.ChatMessagePart{{
			Type:   port.ChatMessagePartTypeImageURL,
			URL:    "https://example.com/image.png",
			Detail: "high",
		}},
	}})
	if err != nil {
		t.Fatalf("toAgenticMessages() error = %v", err)
	}
	if len(messages) != 1 || len(messages[0].ContentBlocks) != 2 {
		t.Fatalf("mapped messages = %#v", messages)
	}
	if messages[0].ContentBlocks[0].UserInputText == nil || messages[0].ContentBlocks[0].UserInputText.Text != "请描述这张图" {
		t.Fatalf("text content block = %#v", messages[0].ContentBlocks[0])
	}
	image := messages[0].ContentBlocks[1].UserInputImage
	if image == nil || image.URL != "https://example.com/image.png" || image.Detail != schema.ImageURLDetailHigh {
		t.Fatalf("image content block = %#v", messages[0].ContentBlocks[1])
	}
}

func TestToAgenticMessagesMapsBase64VisionImageParts(t *testing.T) {
	messages, err := toAgenticMessages([]port.ChatMessage{{
		Role: port.ChatRoleUser,
		ContentParts: []port.ChatMessagePart{{
			Type:       port.ChatMessagePartTypeImageURL,
			Base64Data: "AQID",
			MIMEType:   "image/png",
		}},
	}})
	if err != nil {
		t.Fatalf("toAgenticMessages() error = %v", err)
	}
	image := messages[0].ContentBlocks[0].UserInputImage
	if image == nil || image.Base64Data != "AQID" || image.MIMEType != "image/png" {
		t.Fatalf("base64 image content block = %#v", messages[0].ContentBlocks[0])
	}
}

func TestToAgenticMessagesRejectsUnsafeVisionImageParts(t *testing.T) {
	cases := []port.ChatMessage{
		{Role: port.ChatRoleUser, ContentParts: []port.ChatMessagePart{{Type: port.ChatMessagePartTypeImageURL, URL: "http://127.0.0.1/private.png"}}},
		{Role: port.ChatRoleUser, ContentParts: []port.ChatMessagePart{{Type: port.ChatMessagePartTypeImageURL, Base64Data: "not-base64", MIMEType: "image/png"}}},
		{Role: port.ChatRoleSystem, ContentParts: []port.ChatMessagePart{{Type: port.ChatMessagePartTypeImageURL, URL: "https://example.com/image.png"}}},
	}
	for index, message := range cases {
		if _, err := toAgenticMessages([]port.ChatMessage{message}); !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
			t.Fatalf("case %d error = %v, want invalid request", index, err)
		}
	}
}

func TestChatAdapterStreamsToolCallEvents(t *testing.T) {
	model := &scriptedModel{
		stream: func(int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			return streamFrom(toolCallMessage("call-1", "search", `{"query":"eino"}`)), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	var events []port.ChatStreamEvent
	if err := adapter.Stream(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "find Eino"}},
		Tools:    []port.ToolDefinition{{Name: "search", Parameters: json.RawMessage(`{"type":"object"}`)}},
	}, func(event port.ChatStreamEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if len(events) != 3 || events[0].Kind != port.StreamEventMeta || events[1].Kind != port.StreamEventToolCall || events[2].Kind != port.StreamEventDone {
		t.Fatalf("stream events = %#v", events)
	}
	if len(events[1].ToolCalls) != 1 || events[1].ToolCalls[0].Name != "search" || string(events[1].ToolCalls[0].Arguments) != `{"query":"eino"}` {
		t.Fatalf("tool call event = %#v", events[1])
	}
	if len(events[2].ToolCalls) != 1 || events[2].ToolCalls[0].ID != "call-1" {
		t.Fatalf("done tool calls = %#v", events[2])
	}
}

func TestChatAdapterRejectsInvalidToolDefinitions(t *testing.T) {
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return textMessage("unused"), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	_, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: port.ChatRoleUser, Content: "hello"}},
		Tools:    []port.ToolDefinition{{Name: "search", Parameters: json.RawMessage(`[]`)}},
	})
	if !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("Generate() error = %v, want invalid request", err)
	}
	if model.generateCalls != 0 {
		t.Fatalf("provider calls = %d, want 0", model.generateCalls)
	}
}

func structuredTestSpec() *port.StructuredOutputSpec {
	return &port.StructuredOutputSpec{
		Name: "answer",
		JSONSchema: json.RawMessage(`{
			"type":"object",
			"properties":{"answer":{"type":"string"}},
			"required":["answer"],
			"additionalProperties":false
		}`),
	}
}

func TestChatAdapterStructuredOutputRepairsOnceAndKeepsRunID(t *testing.T) {
	validator, validatorErr := newStructuredValidator(structuredTestSpec())
	if validatorErr != nil {
		t.Fatal(validatorErr)
	}
	if _, validationErr := validator.validate(`{"answer":123}`); validationErr == nil {
		t.Fatal("validator accepted an integer for string answer")
	}
	model := &scriptedModel{
		generate: func(call int) (*schema.AgenticMessage, error) {
			if call == 1 {
				return textMessage(`{"answer":123}`), nil
			}
			return textMessage(`{"answer":"ok"}`), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	response, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages:         testRequest().Messages,
		StructuredOutput: structuredTestSpec(),
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if model.generateCalls != 2 || response.RepairAttempts != 1 {
		t.Fatalf("calls=%d repair_attempts=%d, want one repair", model.generateCalls, response.RepairAttempts)
	}
	if response.RunID == "" || string(response.StructuredJSON) != `{"answer":"ok"}` {
		t.Fatalf("structured response = %#v", response)
	}
}

func TestChatAdapterStructuredOutputFailsAfterSingleRepair(t *testing.T) {
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return textMessage(`{"answer":123}`), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	_, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages:         testRequest().Messages,
		StructuredOutput: structuredTestSpec(),
	})
	if !apperrors.IsAICode(err, apperrors.AICodeStructuredInvalid) {
		t.Fatalf("Generate() error = %v, want structured output error", err)
	}
	if model.generateCalls != 2 {
		t.Fatalf("provider calls = %d, want exactly one repair", model.generateCalls)
	}
}

func TestChatAdapterRejectsInvalidStructuredSchemaBeforeProviderCall(t *testing.T) {
	model := &scriptedModel{
		generate: func(int) (*schema.AgenticMessage, error) {
			return textMessage(`{"ok":true}`), nil
		},
	}
	adapter := testAdapter(t, model, 0)
	_, err := adapter.Generate(context.Background(), port.ChatRequest{
		Messages: testRequest().Messages,
		StructuredOutput: &port.StructuredOutputSpec{
			JSONSchema: json.RawMessage(`{"type":`),
		},
	})
	if !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("Generate() error = %v, want invalid request", err)
	}
	if model.generateCalls != 0 {
		t.Fatalf("provider calls = %d, want 0 for invalid schema", model.generateCalls)
	}
}

func TestChatAdapterRejectsStructuredOutputOnStream(t *testing.T) {
	adapter := testAdapter(t, &scriptedModel{}, 0)
	if err := adapter.Stream(context.Background(), port.ChatRequest{
		Messages:         testRequest().Messages,
		StructuredOutput: structuredTestSpec(),
	}, func(port.ChatStreamEvent) error { return nil }); !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("Stream() error = %v, want invalid request", err)
	}
}

func TestChatAdapterStreamRejectsCanceledContextBeforeProviderCall(t *testing.T) {
	model := &scriptedModel{
		stream: func(int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			return nil, context.Canceled
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := testAdapter(t, model, 2).Stream(ctx, testRequest(), func(port.ChatStreamEvent) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Stream() error = %v, want context.Canceled", err)
	}
	if model.streamCalls > 1 {
		t.Fatalf("Stream() calls = %d, want no retry after cancellation", model.streamCalls)
	}
}

func TestChatAdapterStreamStopsWhenConsumerDisconnects(t *testing.T) {
	model := &scriptedModel{
		stream: func(int) (*schema.StreamReader[*schema.AgenticMessage], error) {
			return streamFrom(textMessage("first"), textMessage("second")), nil
		},
	}
	adapter := testAdapter(t, model, 2)
	events := 0
	err := adapter.Stream(context.Background(), testRequest(), func(event port.ChatStreamEvent) error {
		events++
		if event.Kind == port.StreamEventDelta {
			return context.Canceled
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Stream() error = %v, want context.Canceled", err)
	}
	if model.streamCalls != 1 {
		t.Fatalf("Stream() calls = %d, want 1 after consumer disconnect", model.streamCalls)
	}
	if events != 2 {
		t.Fatalf("events = %d, want meta and first delta", events)
	}
}

func TestShouldRetryUsesProviderHTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "openai bad request", err: &openai.Error{StatusCode: http.StatusBadRequest}, want: false},
		{name: "openai rate limit", err: &openai.Error{StatusCode: http.StatusTooManyRequests}, want: true},
		{name: "openai server failure", err: &openai.Error{StatusCode: http.StatusBadGateway}, want: true},
		{name: "anthropic unauthorized", err: &anthropic.Error{StatusCode: http.StatusUnauthorized}, want: false},
		{name: "anthropic timeout", err: &anthropic.Error{StatusCode: http.StatusRequestTimeout}, want: true},
		{name: "generic transport", err: errors.New("connection reset"), want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRetry(test.err); got != test.want {
				t.Fatalf("shouldRetry() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestStreamWithErrorDeliversErrorAfterChunks(t *testing.T) {
	reader := streamWithError([]*schema.AgenticMessage{textMessage("chunk")}, io.ErrUnexpectedEOF)
	if chunk, err := reader.Recv(); err != nil || assistantText(chunk) != "chunk" {
		t.Fatalf("first Recv() = (%v, %v), want chunk", chunk, err)
	}
	if _, err := reader.Recv(); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("second Recv() error = %v, want unexpected EOF", err)
	}
	reader.Close()
}
