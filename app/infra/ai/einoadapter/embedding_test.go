package einoadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/ai/mock"
)

func TestEmbeddingAdapterPreservesContextErrors(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	adapter, err := NewOpenAIEmbedding(context.Background(), EmbeddingConfig{
		Provider:      "openai",
		Model:         "embedding-model",
		APIKey:        "test-key",
		BaseURL:       server.URL() + "/v1",
		Timeout:       time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.Embed(ctx, port.EmbeddingRequest{Inputs: []string{"hello"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Embed() error = %v, want context canceled", err)
	}
	if requests := server.Requests(); len(requests) != 0 {
		t.Fatalf("canceled embedding made %d provider requests", len(requests))
	}
}

func TestEmbeddingAdapterRejectsOversizedInputBeforeProviderCall(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	adapter, err := NewOpenAIEmbedding(context.Background(), EmbeddingConfig{
		Provider:      "openai",
		Model:         "embedding-model",
		APIKey:        "test-key",
		BaseURL:       server.URL() + "/v1",
		MaxInputBytes: 5,
		Timeout:       time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Embed(context.Background(), port.EmbeddingRequest{Inputs: []string{"hello!"}}); !apperrors.IsAICode(err, apperrors.AICodeInvalidRequest) {
		t.Fatalf("Embed() error = %v, want invalid request", err)
	}
	if requests := server.Requests(); len(requests) != 0 {
		t.Fatalf("oversized embedding made %d provider requests", len(requests))
	}
}

func TestEmbeddingAdapterAcceptsAliBailianAsOpenAICompatible(t *testing.T) {
	server := mock.NewServer(nil)
	defer server.Close()
	adapter, err := NewOpenAIEmbedding(context.Background(), EmbeddingConfig{
		Provider:      "alibailian",
		Model:         "qwen3.7-text-embedding",
		APIKey:        "alibailian-test-key",
		BaseURL:       server.URL() + "/v1",
		Dimension:     3,
		Timeout:       time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Embed(context.Background(), port.EmbeddingRequest{Inputs: []string{"hello"}, Model: "qwen3.7-text-embedding", Version: "qwen3.7-text-embedding"})
	if err != nil || result.Dimension != 3 || len(result.Vectors) != 1 {
		t.Fatalf("Embed() result = %#v, error = %v", result, err)
	}
}

func TestEmbeddingAdapterSplitsAliBailianProviderBatches(t *testing.T) {
	server := mock.NewServer(func(_ context.Context, request mock.Request) (mock.Response, error) {
		var payload struct {
			Input []string `json:"input"`
		}
		if err := json.Unmarshal(request.Body, &payload); err != nil {
			return mock.Response{}, err
		}
		if len(payload.Input) == 0 || len(payload.Input) > maxAliBailianEmbeddingInputsPerRequest {
			return mock.Response{}, errors.New("AliBailian request exceeded provider batch limit")
		}
		data := make([]map[string]any, len(payload.Input))
		for index := range payload.Input {
			data[index] = map[string]any{
				"object":    "embedding",
				"index":     index,
				"embedding": []float32{0.1, 0.2, 0.3},
			}
		}
		return mock.Response{Body: map[string]any{
			"object": "list",
			"data":   data,
			"model":  "qwen3.7-text-embedding",
		}}, nil
	})
	defer server.Close()

	adapter, err := NewOpenAIEmbedding(context.Background(), EmbeddingConfig{
		Provider:      "alibailian",
		Model:         "qwen3.7-text-embedding",
		APIKey:        "alibailian-test-key",
		BaseURL:       server.URL() + "/v1",
		Dimension:     3,
		Timeout:       time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	inputs := make([]string, 17)
	for index := range inputs {
		inputs[index] = "embedding input"
	}
	result, err := adapter.Embed(context.Background(), port.EmbeddingRequest{
		Inputs:  inputs,
		Model:   "qwen3.7-text-embedding",
		Version: "qwen3.7-text-embedding",
	})
	if err != nil || len(result.Vectors) != len(inputs) || result.Dimension != 3 {
		t.Fatalf("Embed() result = %#v, error = %v", result, err)
	}
	requests := server.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider requests = %d, want 2", len(requests))
	}
	for index, request := range requests {
		var payload struct {
			Input []string `json:"input"`
		}
		if err := json.Unmarshal(request.Body, &payload); err != nil {
			t.Fatal(err)
		}
		want := maxAliBailianEmbeddingInputsPerRequest
		if index == 1 {
			want = 1
		}
		if len(payload.Input) != want {
			t.Fatalf("request %d input count = %d, want %d", index, len(payload.Input), want)
		}
	}
}

func TestEmbeddingAdapterOmitsDimensionsForSGLang(t *testing.T) {
	server := mock.NewServer(func(_ context.Context, request mock.Request) (mock.Response, error) {
		var payload map[string]any
		if err := json.Unmarshal(request.Body, &payload); err != nil {
			return mock.Response{}, err
		}
		if _, exists := payload["dimensions"]; exists {
			return mock.Response{}, errors.New("SGLang embedding request unexpectedly included dimensions")
		}
		return mock.Response{Body: map[string]any{
			"object": "list",
			"data":   []map[string]any{{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}}},
			"model":  "Qwen/Qwen3-Embedding-0.6B",
		}}, nil
	})
	defer server.Close()

	adapter, err := NewOpenAIEmbedding(context.Background(), EmbeddingConfig{
		Provider:      "sglang",
		Model:         "Qwen/Qwen3-Embedding-0.6B",
		BaseURL:       server.URL() + "/v1",
		Dimension:     3,
		Timeout:       time.Second,
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Embed(context.Background(), port.EmbeddingRequest{
		Inputs: []string{"hello"},
		Model:  "Qwen/Qwen3-Embedding-0.6B",
	})
	if err != nil || result.Dimension != 3 || len(result.Vectors) != 1 {
		t.Fatalf("Embed() result = %#v, error = %v", result, err)
	}
}

func TestProviderCallErrorPreservesContextErrors(t *testing.T) {
	if err := providerCallError("ai.generate", context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("providerCallError() = %v, want deadline exceeded", err)
	}
}

func TestProviderCallErrorDoesNotLogProviderDetail(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	secretDetail := "provider body contains prompt=private-content api_key=should-not-log"
	if err := providerCallError("ai.generate", errors.New(secretDetail)); err == nil {
		t.Fatal("providerCallError() returned nil")
	}
	if strings.Contains(logs.String(), secretDetail) || strings.Contains(logs.String(), "private-content") {
		t.Fatalf("provider detail leaked into log: %q", logs.String())
	}
	if !strings.Contains(logs.String(), "error_code=ai_provider_unavailable") {
		t.Fatalf("sanitized error code missing from log: %q", logs.String())
	}
}
