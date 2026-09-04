// Package mock provides a deterministic HTTP model server for isolated
// contract tests. It is not used by the production bootstrap.
package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

type Request struct {
	Protocol string
	Path     string
	Body     json.RawMessage
}

type Response struct {
	Status     int
	Headers    map[string]string
	Body       any
	StreamBody string
}

type Responder func(context.Context, Request) (Response, error)

type Server struct {
	httpServer *httptest.Server
	responder  Responder
	mu         sync.Mutex
	requests   []Request
}

func NewServer(responder Responder) *Server {
	s := &Server{responder: responder}
	s.httpServer = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *Server) URL() string {
	if s == nil || s.httpServer == nil {
		return ""
	}
	return s.httpServer.URL
}

func (s *Server) Close() {
	if s != nil && s.httpServer != nil {
		s.httpServer.Close()
	}
}

func (s *Server) Requests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	requests := make([]Request, len(s.requests))
	copy(requests, s.requests)
	return requests
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}
	request := Request{
		Protocol: protocolForPath(r.URL.Path),
		Path:     r.URL.Path,
		Body:     append(json.RawMessage(nil), body...),
	}
	s.mu.Lock()
	s.requests = append(s.requests, request)
	s.mu.Unlock()

	responder := s.responder
	if responder == nil {
		responder = defaultResponder
	}
	response, err := responder(r.Context(), request)
	if err != nil {
		response = Response{Status: http.StatusInternalServerError, Body: map[string]string{"error": err.Error()}}
	}
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}
	if response.StreamBody != "" {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(response.Status)
		_, _ = io.WriteString(w, response.StreamBody)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Status)
	if response.Body != nil {
		if err := json.NewEncoder(w).Encode(response.Body); err != nil {
			return
		}
	}
}

func protocolForPath(path string) string {
	switch path {
	case "/v1/responses":
		return "openai_responses"
	case "/v1/messages":
		return "anthropic_messages"
	case "/v1/chat/completions":
		return "openai_chat_completions"
	case "/v1/embeddings":
		return "openai_embeddings"
	default:
		return "unknown"
	}
}

func defaultResponder(_ context.Context, request Request) (Response, error) {
	if requestWantsStream(request.Body) {
		return Response{StreamBody: streamBody(request.Protocol)}, nil
	}
	switch request.Protocol {
	case "anthropic_messages":
		return Response{Body: map[string]any{
			"id":      "mock-message",
			"type":    "message",
			"role":    "assistant",
			"content": []map[string]string{{"type": "text", "text": "mock response"}},
			"model":   "mock-model",
		}}, nil
	case "openai_responses":
		return Response{Body: map[string]any{
			"id": "mock-response",
			"output": []map[string]any{{
				"type":    "message",
				"content": []map[string]string{{"type": "output_text", "text": "mock response"}},
			}},
		}}, nil
	case "openai_chat_completions":
		return Response{Body: map[string]any{
			"id": "mock-chat-completion",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]string{"role": "assistant", "content": "mock response"},
				"finish_reason": "stop",
			}},
		}}, nil
	case "openai_embeddings":
		return Response{Body: map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"object": "embedding", "index": 0, "embedding": []float32{0.1, 0.2, 0.3}},
			},
			"model": "mock-embedding-model",
		}}, nil
	default:
		return Response{}, fmt.Errorf("unsupported mock model path: %s", request.Path)
	}
}

func requestWantsStream(body json.RawMessage) bool {
	var request struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &request) == nil && request.Stream
}

func streamBody(protocol string) string {
	switch protocol {
	case "anthropic_messages":
		return "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"mock-message-stream\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"mock-model\",\"content\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\n" +
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"mock stream\"}}\n\n" +
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}\n\n" +
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	case "openai_responses":
		return "data: {\"type\":\"response.created\",\"response\":{\"id\":\"mock-response-stream\",\"object\":\"response\",\"created_at\":1,\"status\":\"in_progress\",\"model\":\"mock-model\",\"output\":[]}}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"mock stream\",\"item_id\":\"item-1\",\"output_index\":0,\"content_index\":0,\"sequence_number\":1,\"logprobs\":[]}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"mock-response-stream\",\"object\":\"response\",\"created_at\":1,\"status\":\"completed\",\"model\":\"mock-model\",\"output\":[]}}\n\n"
	case "openai_chat_completions":
		return "data: {\"id\":\"mock-chat-stream\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"mock stream\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"mock-chat-stream\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"mock-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"
	default:
		return ""
	}
}
