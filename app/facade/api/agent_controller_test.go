package api

import (
	"benetnasch/app/domain/port"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type apiAgentEventStore struct {
	events []port.AgentChatEvent
}

type apiAgentChatService struct {
	called *bool
}

func (s apiAgentChatService) Chat(context.Context, port.AgentChatRequest, func(port.AgentChatEvent) error) error {
	*s.called = true
	return nil
}

func (s apiAgentChatService) DeleteSession(context.Context, string, string) error {
	*s.called = true
	return nil
}

func TestAgentChatDoesNotCallServiceWhenFeatureIsDisabled(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousService := agentChatService
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		agentChatService = previousService
	})
	called := false
	agentFeatureFlags = port.AgentFeatureFlags{}
	agentChatService = apiAgentChatService{called: &called}
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/agent/chat", strings.NewReader(`{"message":"hello"}`))
	AgentChat(context)
	if called {
		t.Fatal("disabled public chat called the application service")
	}
	if recorder.Header().Get("Content-Type") != "text/event-stream; charset=utf-8" || !strings.Contains(recorder.Body.String(), `"code":"ai_disabled"`) {
		t.Fatalf("disabled public chat response headers=%v body=%q", recorder.Header(), recorder.Body.String())
	}
}

func TestAgentSessionEndpointsDoNotCallDependenciesWhenChatIsDisabled(t *testing.T) {
	previousFlags := agentFeatureFlags
	previousService := agentChatService
	previousStore := agentEventStore
	t.Cleanup(func() {
		agentFeatureFlags = previousFlags
		agentChatService = previousService
		agentEventStore = previousStore
	})
	called := false
	agentFeatureFlags = port.AgentFeatureFlags{}
	agentChatService = apiAgentChatService{called: &called}
	agentEventStore = &apiAgentEventStore{}
	gin.SetMode(gin.TestMode)

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		called = false
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(method, "/agent/sessions/session-1/events", nil)
		if method == http.MethodGet {
			ReplayAgentSession(context)
		} else {
			DeleteAgentSession(context)
		}
		if called {
			t.Fatalf("disabled session endpoint %s called the application service", method)
		}
		if !strings.Contains(recorder.Body.String(), "公开对话暂未开启") {
			t.Fatalf("disabled session endpoint %s response = %q", method, recorder.Body.String())
		}
	}
}

func (s *apiAgentEventStore) Append(_ context.Context, _ string, event port.AgentChatEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s *apiAgentEventStore) Replay(context.Context, string, string, string, int64) ([]port.AgentChatEvent, error) {
	return s.events, nil
}

func (s *apiAgentEventStore) Delete(context.Context, string, string) error { return nil }

func TestWriteAgentSSESerializesOnlyPublicEventShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(writer)
	context.Request = httptest.NewRequest("POST", "/agent/chat", nil)
	prepareAgentSSE(context)
	if err := writeAgentSSE(context, port.AgentChatEvent{
		Kind:      port.StreamEventCitation,
		SessionID: "session-1",
		TurnID:    "turn-1",
		Citation:  &port.Citation{ArticleID: 7, Title: "公开文章", URL: "/articles/7"},
	}); err != nil {
		t.Fatal(err)
	}
	body := writer.Body.String()
	if !strings.Contains(body, "event: citation\n") || !strings.Contains(body, `"articleId":7`) || !strings.Contains(body, `"sessionId":"session-1"`) {
		t.Fatalf("SSE body = %q", body)
	}
	if strings.Contains(body, "ToolCalls") || strings.Contains(body, "SystemPrompt") {
		t.Fatalf("SSE leaked internal fields: %q", body)
	}
}

func TestDecodeAgentChatRequestRejectsServerOwnedFields(t *testing.T) {
	for _, body := range []string{
		`{"message":"hello","ownerKey":"user:1"}`,
		`{"message":"hello","privileged":true}`,
		`{"message":"hello","role":"admin"}`,
		`{"message":"hello","tool":"delete_article"}`,
	} {
		var request agentChatHTTPRequest
		if err := decodeAgentChatRequest(strings.NewReader(body), &request); err == nil {
			t.Fatalf("server-owned field was accepted: %s", body)
		}
	}
}

func TestDecodeAgentChatRequestRejectsTrailingJSON(t *testing.T) {
	var request agentChatHTTPRequest
	if err := decodeAgentChatRequest(strings.NewReader(`{"message":"hello"}{"message":"second"}`), &request); err == nil {
		t.Fatal("multiple JSON values were accepted")
	}
}

func TestDecodeAgentChatRequestKeepsPublicContract(t *testing.T) {
	var request agentChatHTTPRequest
	body := `{"sessionId":"session-1","message":"hello","timeRange":{"from":"2026-01-01","to":"2026-12-31"}}`
	if err := decodeAgentChatRequest(strings.NewReader(body), &request); err != nil {
		t.Fatal(err)
	}
	if request.SessionID != "session-1" || request.Message != "hello" || request.TimeRange == nil || request.TimeRange.From != "2026-01-01" {
		t.Fatalf("decoded request = %#v", request)
	}

	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"message":"hello"`) {
		t.Fatalf("public request was not decoded: %s", encoded)
	}
}

func TestDecodeAgentChatRequestRequiresBody(t *testing.T) {
	var request agentChatHTTPRequest
	if err := decodeAgentChatRequest(io.Reader(nil), &request); err == nil {
		t.Fatal("nil request body was accepted")
	}
}

func TestPublicSSEEventNameRejectsProviderInternalEvents(t *testing.T) {
	if _, ok := publicSSEEventName(port.StreamEventToolCall); ok {
		t.Fatal("tool call event must not be public")
	}
}

func TestAgentOwnerKeyUsesSocketPeerWhenForwardedHeadersAreUntrusted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, engine := gin.CreateTestContext(recorder)
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	context.Request = httptest.NewRequest(http.MethodPost, "/agent/chat", nil)
	context.Request.RemoteAddr = "198.51.100.20:443"
	context.Request.Header.Set("X-Forwarded-For", "203.0.113.7")

	if got := agentOwnerKey(context); got != "ip:198.51.100.20" {
		t.Fatalf("agentOwnerKey() = %q, want socket peer owner", got)
	}
}

func TestAgentSSEEmitterAddsEnvelopeAndPersistsPublicEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	store := &apiAgentEventStore{}
	emitter := newAgentSSEEmitter(context, store, "ip:one")
	if err := emitter.Emit(context, port.AgentChatEvent{
		Kind:      port.StreamEventDelta,
		SessionID: "session-1",
		TurnID:    "turn-1",
		Text:      "hello",
	}); err != nil {
		t.Fatal(err)
	}
	if len(store.events) != 1 || store.events[0].Seq != 1 || store.events[0].EventID == "" || store.events[0].Replay {
		t.Fatalf("stored event = %#v", store.events)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"eventId":"`) || !strings.Contains(body, `"seq":1`) || !strings.Contains(body, `"sessionId":"session-1"`) {
		t.Fatalf("SSE envelope = %q", body)
	}
}
