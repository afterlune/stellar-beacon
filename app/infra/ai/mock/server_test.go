package mock

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestServerRecordsProtocolAndReturnsDeterministicResponse(t *testing.T) {
	server := NewServer(nil)
	defer server.Close()

	response, err := http.Post(server.URL()+"/v1/chat/completions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"] != "mock-chat-completion" {
		t.Fatalf("unexpected mock payload: %#v", payload)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Protocol != "openai_chat_completions" {
		t.Fatalf("unexpected requests: %#v", requests)
	}
}

func TestServerUsesCustomResponder(t *testing.T) {
	server := NewServer(func(ctx context.Context, request Request) (Response, error) {
		if request.Protocol != "anthropic_messages" {
			t.Fatalf("unexpected protocol: %s", request.Protocol)
		}
		if ctx == nil {
			t.Fatal("request context is nil")
		}
		return Response{Status: http.StatusAccepted, Body: map[string]string{"ok": "true"}}, nil
	})
	defer server.Close()

	response, err := http.Post(server.URL()+"/v1/messages", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
	}
}
