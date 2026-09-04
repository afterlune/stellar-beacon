package writing

import (
	"context"
	"errors"
	"testing"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

func TestContentAnalyzerReturnsStructuredResult(t *testing.T) {
	chat := &fakeChatGateway{response: port.ChatResponse{
		RunID:          "run-1",
		StructuredJSON: []byte(`{"summary":" 摘要 ","category":" 后端 ","tags":[" Go ","Go","Redis"]}`),
	}}
	analyzer, err := NewContentAnalyzer(Config{Chat: chat, Model: "writing-model"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := analyzer.Analyze(context.Background(), port.ContentUnderstandingRequest{ArticleID: 42, Title: "标题", Content: "文章正文"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ArticleID != 42 || result.RunID != "run-1" || result.Summary != "摘要" || result.Category != "后端" || len(result.Tags) != 2 || result.Tags[0] != "Go" || result.Tags[1] != "Redis" {
		t.Fatalf("content understanding = %+v", result)
	}
	if len(chat.requests) != 1 || chat.requests[0].StructuredOutput == nil || chat.requests[0].StructuredOutput.Name != contentUnderstandingSchemaName || chat.requests[0].MaxOutputTokens != DefaultContentUnderstandingMaxOutputTokens {
		t.Fatalf("request = %+v", chat.requests)
	}
}

func TestContentAnalyzerRejectsMalformedStructuredResult(t *testing.T) {
	for name, raw := range map[string][]byte{
		"empty":           nil,
		"missing summary": []byte(`{"category":"Go","tags":[]}`),
		"unknown field":   []byte(`{"summary":"s","category":"Go","tags":[],"extra":true}`),
		"empty summary":   []byte(`{"summary":" ","category":"Go","tags":[]}`),
		"trailing":        []byte(`{"summary":"s","category":"Go","tags":[]} {}`),
	} {
		t.Run(name, func(t *testing.T) {
			analyzer, err := NewContentAnalyzer(Config{Chat: &fakeChatGateway{response: port.ChatResponse{StructuredJSON: raw}}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := analyzer.Analyze(context.Background(), port.ContentUnderstandingRequest{ArticleID: 1, Content: "content"}); !apperrors.IsAICode(err, apperrors.AICodeStructuredInvalid) {
				t.Fatalf("error = %v, want structured invalid", err)
			}
		})
	}
}

func TestContentAnalyzerValidatesArticleAndContext(t *testing.T) {
	chat := &fakeChatGateway{}
	analyzer, err := NewContentAnalyzer(Config{Chat: chat})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := analyzer.Analyze(context.Background(), port.ContentUnderstandingRequest{Content: "content"}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid article ID error kind = %v, want validation", apperrors.KindOf(err))
	}
	if _, err := analyzer.Analyze(context.Background(), port.ContentUnderstandingRequest{ArticleID: 1}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("empty content error kind = %v, want validation", apperrors.KindOf(err))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := analyzer.Analyze(ctx, port.ContentUnderstandingRequest{ArticleID: 1, Content: "content"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled error = %v, want context.Canceled", err)
	}
}

func TestContentAnalyzerEnforcesInputLimitBeforeModelCall(t *testing.T) {
	chat := &fakeChatGateway{}
	analyzer, err := NewContentAnalyzer(Config{Chat: chat, MaxInputRunes: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := analyzer.Analyze(context.Background(), port.ContentUnderstandingRequest{ArticleID: 1, Title: "标题", Content: "正文超长"}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("input-limit error = %v, want validation", err)
	}
	if len(chat.requests) != 0 {
		t.Fatalf("input-limit failure called model %d times", len(chat.requests))
	}
	if _, err := NewContentAnalyzer(Config{Chat: chat, MaxInputRunes: -1}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("invalid input limit constructor error = %v, want validation", err)
	}
}
