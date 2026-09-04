package api

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type agentChatHTTPRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
	TimeRange *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"timeRange"`
}

type agentSSEUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type agentSSEPayload struct {
	EventID   string            `json:"eventId,omitempty"`
	Seq       int64             `json:"seq,omitempty"`
	Replay    bool              `json:"replay,omitempty"`
	SessionID string            `json:"sessionId,omitempty"`
	TurnID    string            `json:"turnId,omitempty"`
	RunID     string            `json:"runId,omitempty"`
	Provider  string            `json:"provider,omitempty"`
	Protocol  string            `json:"protocol,omitempty"`
	Model     string            `json:"model,omitempty"`
	Text      string            `json:"text,omitempty"`
	Opening   string            `json:"opening,omitempty"`
	Citation  *port.Citation    `json:"citation,omitempty"`
	State     map[string]string `json:"state,omitempty"`
	Usage     *agentSSEUsage    `json:"usage,omitempty"`
	Code      string            `json:"code,omitempty"`
	Message   string            `json:"message,omitempty"`
}

// AgentChat
// @Summary      Agent 公开对话
// @Description  以 SSE 返回公开 Agent 的人格化对话事件；不会接受客户端身份、权限、工具或 Provider 字段
// @Accept       json
// @Produce      text/event-stream
// @Param        request body agentChatHTTPRequest true "公开对话请求"
// @Success      200 {string} string
// @Failure      400 {object} model.ResultVO
// @Router       /agent/chat [POST]
//
// AgentChat exposes the public, anonymous Agent as a small SSE protocol.
// Request and response bodies are handled here so the application service
// remains independent from Gin and HTTP flushing details.
func AgentChat(c *gin.Context) {
	if !agentFeatureFlags.PublicChat {
		prepareAgentSSE(c)
		_ = writeAgentSSE(c, port.AgentChatEvent{
			Kind:  port.StreamEventError,
			Text:  "公开对话暂未开启",
			State: map[string]string{"code": string(errors.AICodeDisabled)},
		})
		return
	}
	var request agentChatHTTPRequest
	if err := decodeAgentChatRequest(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("agent.chat.request", "request body is invalid")))
		return
	}
	ownerKey := agentOwnerKey(c)
	agentRequest := port.AgentChatRequest{
		SessionID: request.SessionID,
		Message:   request.Message,
		OwnerKey:  ownerKey,
		RequestID: strings.TrimSpace(c.GetHeader("X-Request-ID")),
	}
	if request.TimeRange != nil {
		agentRequest.TimeRange = port.KnowledgeFilterInput{
			From: request.TimeRange.From,
			To:   request.TimeRange.To,
		}
	}

	prepareAgentSSE(c)
	ctx := c.Request.Context()
	emitter := newAgentSSEEmitter(c, agentEventStore, ownerKey)
	err := agentChatService.Chat(ctx, agentRequest, func(event port.AgentChatEvent) error {
		return emitter.Emit(ctx, event)
	})
	if err == nil || stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return
	}
	_ = emitter.Emit(ctx, port.AgentChatEvent{
		Kind:      port.StreamEventError,
		SessionID: agentRequest.SessionID,
		Text:      publicAgentErrorMessage(err),
		State: map[string]string{
			"code": string(publicAgentErrorCode(err)),
		},
	})
}

// decodeAgentChatRequest keeps the public HTTP contract closed. In
// particular, identity, privilege, tool and provider fields must never be
// accepted from a browser request, even if a future request struct grows
// server-only fields. The application service still receives the owner key
// from the authenticated context, never from JSON.
func decodeAgentChatRequest(reader io.Reader, request *agentChatHTTPRequest) error {
	if reader == nil || request == nil {
		return stderrors.New("agent chat request is required")
	}
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return stderrors.New("agent chat request contains multiple JSON values")
		}
		return err
	}
	return nil
}

// DeleteAgentSession
// @Summary      删除 Agent 会话
// @Description  只删除当前调用者范围内的短期匿名会话及可回放事件；不存在的会话按幂等成功处理
// @Param        id path string true "会话 ID"
// @Success      200 {object} model.ResultVO
// @Router       /agent/sessions/{id} [DELETE]
//
// DeleteAgentSession removes only the caller's anonymous session. A missing
// Redis key is intentionally idempotent at the store boundary.
func DeleteAgentSession(c *gin.Context) {
	if !agentFeatureFlags.PublicChat {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("公开对话暂未开启"))
		return
	}
	ownerKey := agentOwnerKey(c)
	sessionID := c.Param("id")
	if err := agentChatService.DeleteSession(c.Request.Context(), sessionID, ownerKey); err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	if agentEventStore != nil {
		if err := agentEventStore.Delete(c.Request.Context(), ownerKey, sessionID); err != nil {
			c.JSON(http.StatusOK, model.ResultFromError(err))
			return
		}
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ReplayAgentSession
// @Summary      恢复 Agent 会话事件
// @Description  按 afterSeq 回放当前 turn 的短期公开 SSE 事件
// @Produce      text/event-stream
// @Param        id path string true "会话 ID"
// @Param        afterSeq query int false "仅回放序号大于该值的事件"
// @Param        turnId query string false "限制为当前 turn"
// @Success      200 {string} string
// @Router       /agent/sessions/{id}/events [GET]
//
// ReplayAgentSession replays only short-lived public events from the current
// turn. It never calls the model and explicitly filters stale turn IDs.
func ReplayAgentSession(c *gin.Context) {
	if !agentFeatureFlags.PublicChat {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("公开对话暂未开启"))
		return
	}
	if agentEventStore == nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("对话恢复暂未开启"))
		return
	}
	sessionID, err := port.NormalizeAgentSessionID(c.Param("id"))
	if err != nil || sessionID == "" {
		c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("agent.events.replay", "session id is invalid")))
		return
	}
	afterSeq := int64(0)
	if raw := strings.TrimSpace(c.Query("afterSeq")); raw != "" {
		afterSeq, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || afterSeq < 0 {
			c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("agent.events.replay", "afterSeq is invalid")))
			return
		}
	}
	turnID := strings.TrimSpace(c.Query("turnId"))
	if turnID != "" {
		if turnID, err = port.NormalizeAgentSessionID(turnID); err != nil {
			c.JSON(http.StatusBadRequest, model.ResultFromError(errors.Invalid("agent.events.replay", "turnId is invalid")))
			return
		}
	}
	events, err := agentEventStore.Replay(c.Request.Context(), agentOwnerKey(c), sessionID, turnID, afterSeq)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	prepareAgentSSE(c)
	for _, event := range events {
		event.Replay = true
		if err := writeAgentSSE(c, event); err != nil {
			return
		}
	}
}

type agentSSEEmitter struct {
	writer    *gin.Context
	store     port.AgentEventStore
	ownerKey  string
	sessionID string
	turnID    string
	seq       int64
}

func newAgentSSEEmitter(writer *gin.Context, store port.AgentEventStore, ownerKey string) *agentSSEEmitter {
	return &agentSSEEmitter{writer: writer, store: store, ownerKey: ownerKey}
}

func (e *agentSSEEmitter) Emit(ctx context.Context, event port.AgentChatEvent) error {
	if e == nil || e.writer == nil {
		return errors.Invalid("agent.sse.emit", "event writer is required")
	}
	if _, ok := publicSSEEventName(event.Kind); !ok {
		return errors.Invalid("agent.sse.event", "event type is not public")
	}
	if event.SessionID != "" {
		e.sessionID = event.SessionID
	}
	if event.TurnID != "" {
		e.turnID = event.TurnID
	}
	if event.SessionID == "" {
		event.SessionID = e.sessionID
	}
	if event.TurnID == "" {
		event.TurnID = e.turnID
	}
	e.seq++
	event.EventID = uuid.NewString()
	event.Seq = e.seq
	event.Replay = false
	if e.store != nil && event.SessionID != "" && event.TurnID != "" {
		if err := e.store.Append(ctx, e.ownerKey, event); err != nil {
			return err
		}
	}
	return writeAgentSSE(e.writer, event)
}

func prepareAgentSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func writeAgentSSE(c *gin.Context, event port.AgentChatEvent) error {
	eventName, ok := publicSSEEventName(event.Kind)
	if !ok {
		return errors.Invalid("agent.sse.event", "event type is not public")
	}
	payload := agentSSEPayload{
		EventID:   event.EventID,
		Seq:       event.Seq,
		Replay:    event.Replay,
		SessionID: event.SessionID,
		TurnID:    event.TurnID,
		RunID:     event.RunID,
		Provider:  event.Provider,
		Protocol:  string(event.Protocol),
		Model:     event.Model,
		Text:      event.Text,
		Opening:   event.Opening,
		Citation:  event.Citation,
		State:     event.State,
		Usage:     usagePayload(event.Usage),
	}
	if event.Kind == port.StreamEventError {
		payload.Code = strings.TrimSpace(event.State["code"])
		if payload.Code == "" {
			payload.Code = string(errors.AICodeProviderUnavailable)
		}
		payload.Message = event.Text
		payload.Text = ""
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(c.Writer, "event: "+eventName+"\ndata: "+string(data)+"\n\n"); err != nil {
		return err
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func publicSSEEventName(kind port.StreamEventKind) (string, bool) {
	switch kind {
	case port.StreamEventMeta, port.StreamEventDelta, port.StreamEventCitation, port.StreamEventState, port.StreamEventDone, port.StreamEventError:
		return string(kind), true
	default:
		return "", false
	}
}

func usagePayload(usage *port.TokenUsage) *agentSSEUsage {
	if usage == nil {
		return nil
	}
	return &agentSSEUsage{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, TotalTokens: usage.TotalTokens}
}

func agentOwnerKey(c *gin.Context) string {
	if value, ok := c.Get("userInfo"); ok {
		if user, ok := value.(port.UserDetailsDTO); ok && user.UserInfoId > 0 {
			return "user:" + strconv.Itoa(user.UserInfoId)
		}
	}
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = strings.TrimSpace(c.RemoteIP())
	}
	if ip == "" {
		ip = "anonymous"
	}
	return "ip:" + ip
}

func publicAgentErrorCode(err error) errors.AICode {
	if code := errors.AICodeOf(err); code != "" {
		return code
	}
	if stderrors.Is(err, context.Canceled) {
		return errors.AICodeProviderUnavailable
	}
	return errors.AICodeProviderUnavailable
}

func publicAgentErrorMessage(err error) string {
	switch publicAgentErrorCode(err) {
	case errors.AICodeDisabled:
		return "我还没有醒来，公开对话功能暂未开启。"
	case errors.AICodeInvalidRequest:
		return "请求格式不正确"
	case errors.AICodeRateLimited:
		return "请求过于频繁，请稍后再试"
	default:
		return "我现在暂时离线，无法查阅文章与星图；请稍后再来。"
	}
}
