package writing

import (
	"context"
	"errors"
	"strings"
	"testing"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

func TestSuggestionAssistantReturnsStrictStructuredSuggestion(t *testing.T) {
	chat := &fakeChatGateway{response: port.ChatResponse{StructuredJSON: []byte(`{"category":" 后端 ","tags":[" Go ","Go","Redis"]}`)}}
	assistant, err := NewSuggestionAssistant(Config{Chat: chat, Model: "writing-model"})
	if err != nil {
		t.Fatal(err)
	}
	suggestion, err := assistant.Suggest(context.Background(), port.ContentSuggestionRequest{
		Title:   " 标题 ",
		Content: "文章内容",
	})
	if err != nil {
		t.Fatal(err)
	}
	if suggestion.Category != "后端" || len(suggestion.Tags) != 2 || suggestion.Tags[0] != "Go" || suggestion.Tags[1] != "Redis" {
		t.Fatalf("suggestion = %+v", suggestion)
	}
	if len(chat.requests) != 1 {
		t.Fatalf("chat requests = %d, want one", len(chat.requests))
	}
	request := chat.requests[0]
	if request.UseCase != port.AIUseCaseWriting || request.Model != "writing-model" || request.MaxOutputTokens != DefaultSuggestionMaxOutputTokens || request.StructuredOutput == nil || request.StructuredOutput.Name != SuggestionSchemaName || len(request.Tools) != 0 {
		t.Fatalf("structured request = %+v", request)
	}
	if !strings.Contains(string(request.StructuredOutput.JSONSchema), `"additionalProperties": false`) || !strings.Contains(request.Messages[1].Content, "文章内容") {
		t.Fatalf("structured schema/prompt = %s / %+v", request.StructuredOutput.JSONSchema, request.Messages)
	}
}

func TestSuggestionAssistantRejectsMalformedStructuredOutput(t *testing.T) {
	for name, raw := range map[string][]byte{
		"empty":          nil,
		"unknown field":  []byte(`{"category":"Go","tags":[],"extra":true}`),
		"missing field":  []byte(`{"category":"Go"}`),
		"empty category": []byte(`{"category":" ","tags":[]}`),
		"empty tag":      []byte(`{"category":"Go","tags":[""]}`),
		"trailing":       []byte(`{"category":"Go","tags":[]} {}`),
	} {
		t.Run(name, func(t *testing.T) {
			chat := &fakeChatGateway{response: port.ChatResponse{StructuredJSON: raw}}
			assistant, err := NewSuggestionAssistant(Config{Chat: chat})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := assistant.Suggest(context.Background(), port.ContentSuggestionRequest{Content: "content"}); !apperrors.IsAICode(err, apperrors.AICodeStructuredInvalid) {
				t.Fatalf("error = %v, want structured invalid", err)
			}
		})
	}
}

func TestSuggestionAssistantPropagatesErrorsAndCancellation(t *testing.T) {
	modelErr := errors.New("provider failed")
	chat := &fakeChatGateway{err: modelErr}
	assistant, err := NewSuggestionAssistant(Config{Chat: chat})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := assistant.Suggest(context.Background(), port.ContentSuggestionRequest{Content: "content"}); !errors.Is(err, modelErr) {
		t.Fatalf("model error = %v, want wrapped error", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	chat.err = nil
	if _, err := assistant.Suggest(ctx, port.ContentSuggestionRequest{Content: "content"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
}

func TestSuggestionAssistantEnforcesInputLimitBeforeModelCall(t *testing.T) {
	chat := &fakeChatGateway{}
	assistant, err := NewSuggestionAssistant(Config{Chat: chat, MaxInputRunes: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := assistant.Suggest(context.Background(), port.ContentSuggestionRequest{Title: "标题", Content: "正文超长"}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("input-limit error = %v, want validation", err)
	}
	if len(chat.requests) != 0 {
		t.Fatalf("input-limit failure called model %d times", len(chat.requests))
	}
	if _, err := NewSuggestionAssistant(Config{Chat: chat, MaxInputRunes: -1}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid input limit constructor error = %v, want validation", err)
	}
}
