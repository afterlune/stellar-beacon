// Package einoadapter keeps Eino and provider SDK types behind the domain
// ports. No application package should import this package directly; the
// bootstrap may construct its ModelRouter and pass that port inward.
package einoadapter

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/cloudwego/eino-ext/components/model/agenticclaude"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	einoschema "github.com/eino-contrib/jsonschema"
	"github.com/google/uuid"
	chatopenai "github.com/meguminnnnnnnnn/go-openai"
	"github.com/openai/openai-go/v3"
)

type agenticModel interface {
	Generate(context.Context, []*schema.AgenticMessage, ...einomodel.Option) (*schema.AgenticMessage, error)
	Stream(context.Context, []*schema.AgenticMessage, ...einomodel.Option) (*schema.StreamReader[*schema.AgenticMessage], error)
}

type Config struct {
	Provider                string
	Protocol                port.ProviderProtocol
	Model                   string
	APIKey                  string
	BaseURL                 string
	Timeout                 time.Duration
	MaxRetries              int
	MaxConcurrent           int
	CircuitFailureThreshold int
	CircuitResetTimeout     time.Duration
	HTTPClient              *http.Client
	Observer                port.AIRunObserver
}

type ChatAdapter struct {
	model     agenticModel
	config    Config
	semaphore chan struct{}
	breaker   *circuitBreaker
}

var _ port.ChatGateway = (*ChatAdapter)(nil)

// CircuitState is intentionally diagnostic-only; it does not expose the
// provider implementation through the domain ChatGateway contract.
func (a *ChatAdapter) CircuitState() string {
	if a == nil {
		return "disabled"
	}
	return a.breaker.stateName()
}

func NewOpenAIResponses(ctx context.Context, config Config) (*ChatAdapter, error) {
	config.Protocol = port.ProviderProtocolOpenAIResponses
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "openai.responses.configure", errors.New("API key is required"))
	}
	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	zeroRetries := 0
	model, err := agenticopenai.NewResponsesModel(ctx, &agenticopenai.ResponsesConfig{
		APIKey:     config.APIKey,
		BaseURL:    config.BaseURL,
		Model:      config.Model,
		Timeout:    &config.Timeout,
		HTTPClient: config.HTTPClient,
		MaxRetries: &zeroRetries,
	})
	if err != nil {
		return nil, providerSetupError("openai.responses.configure", err)
	}
	return newChatAdapter(model, config)
}

func NewOpenAIChatCompletions(ctx context.Context, config Config) (*ChatAdapter, error) {
	config.Protocol = port.ProviderProtocolOpenAIChatCompletions
	config.Provider = defaultProvider(config.Provider, "sglang")
	if config.Provider == "openai" && strings.TrimSpace(config.APIKey) == "" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "openai.chat.configure", errors.New("API key is required"))
	}
	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	model, err := agenticopenai.NewChatModel(ctx, &agenticopenai.ChatConfig{
		APIKey:     config.APIKey,
		BaseURL:    config.BaseURL,
		Model:      config.Model,
		Timeout:    config.Timeout,
		HTTPClient: config.HTTPClient,
	})
	if err != nil {
		return nil, providerSetupError("openai.chat.configure", err)
	}
	return newChatAdapter(model, config)
}

func NewAnthropicMessages(ctx context.Context, config Config) (*ChatAdapter, error) {
	config.Protocol = port.ProviderProtocolAnthropicMessages
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "anthropic.messages.configure", errors.New("API key is required"))
	}
	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	model, err := agenticclaude.New(ctx, &agenticclaude.Config{
		APIKey:         config.APIKey,
		BaseURL:        config.BaseURL,
		Model:          config.Model,
		MaxTokens:      maxOutputTokens,
		RequestTimeout: config.Timeout,
		HTTPClient:     config.HTTPClient,
	})
	if err != nil {
		return nil, providerSetupError("anthropic.messages.configure", err)
	}
	return newChatAdapter(model, config)
}

const (
	maxOutputTokens             = 4096
	maxChatContentParts         = 32
	maxImageURLLength           = 8192
	maxImageDecodedByteLength   = 32 * 1024 * 1024
	maxImageBase64ByteLength    = 4 * ((maxImageDecodedByteLength + 2) / 3)
	maxImageRequestPayloadBytes = 44 * 1024 * 1024
)

func newChatAdapter(model agenticModel, config Config) (*ChatAdapter, error) {
	if model == nil {
		return nil, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.adapter.configure", errors.New("model is nil"))
	}
	return &ChatAdapter{
		model:     model,
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
		breaker:   newCircuitBreaker(config.CircuitFailureThreshold, config.CircuitResetTimeout),
	}, nil
}

func normalizeConfig(config Config) (Config, error) {
	config.Provider = defaultProvider(config.Provider, string(config.Protocol))
	config.Model = strings.TrimSpace(config.Model)
	if config.Model == "" {
		return Config{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.adapter.configure", errors.New("model is required"))
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.MaxRetries > 3 {
		config.MaxRetries = 3
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 2
	}
	if config.MaxConcurrent > 128 {
		config.MaxConcurrent = 128
	}
	if config.CircuitFailureThreshold <= 0 {
		config.CircuitFailureThreshold = 3
	}
	if config.CircuitResetTimeout <= 0 {
		config.CircuitResetTimeout = 10 * time.Second
	}
	return config, nil
}

func defaultProvider(provider, fallback string) string {
	if strings.TrimSpace(provider) != "" {
		return strings.TrimSpace(provider)
	}
	return fallback
}

func providerSetupError(op string, err error) error {
	return apperrors.WrapAIUnavailable(op, err)
}

func (a *ChatAdapter) Generate(ctx context.Context, request port.ChatRequest) (response port.ChatResponse, err error) {
	if a == nil || a.model == nil {
		return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.generate", errors.New("chat adapter is not initialized"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	messages, err := toAgenticMessages(request.Messages)
	if err != nil {
		return port.ChatResponse{}, err
	}
	structured, err := newStructuredValidator(request.StructuredOutput)
	if err != nil {
		return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.generate.structured", err)
	}
	messages = prependStructuredInstruction(messages, structured)
	options, err := requestOptions(request)
	if err != nil {
		return port.ChatResponse{}, err
	}
	if !a.breaker.allow(time.Now().UTC()) {
		return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeCircuitOpen, "ai.generate", errors.New("provider circuit is open"))
	}
	localRunID := uuid.NewString()
	startedAt := time.Now().UTC()
	settled := false
	defer func() {
		if !settled {
			a.breaker.abandon()
		}
		a.observeRun(ctx, request, localRunID, startedAt, time.Time{}, response, err)
	}()
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()
	release, err := a.acquire(ctx)
	if err != nil {
		return port.ChatResponse{}, err
	}
	defer release()

	var message *schema.AgenticMessage
	for attempt := 0; attempt <= a.config.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := waitRetry(ctx, attempt); err != nil {
				return port.ChatResponse{}, err
			}
		}
		message, err = a.model.Generate(ctx, messages, options...)
		if err == nil {
			break
		}
		retryable := shouldRetry(err)
		if !retryable || attempt == a.config.MaxRetries {
			a.breaker.failure(retryable, time.Now().UTC())
			settled = true
			return port.ChatResponse{}, providerCallError("ai.generate", err)
		}
	}
	response, err = normalizeResponse(message, a.config, localRunID)
	if err != nil {
		a.breaker.failure(true, time.Now().UTC())
		settled = true
		return response, err
	}
	if structured != nil {
		structuredJSON, validationErr := structured.validate(response.Text)
		if validationErr != nil {
			repairRequest := structuredRepairMessages(messages, structured, response.Text, validationErr)
			repairedMessage, repairErr := a.model.Generate(ctx, repairRequest, options...)
			if repairErr != nil {
				retryable := shouldRetry(repairErr)
				a.breaker.failure(retryable, time.Now().UTC())
				settled = true
				return port.ChatResponse{}, providerCallError("ai.generate.structured_repair", repairErr)
			}
			response, err = normalizeResponse(repairedMessage, a.config, localRunID)
			if err != nil {
				a.breaker.failure(true, time.Now().UTC())
				settled = true
				return response, err
			}
			structuredJSON, validationErr = structured.validate(response.Text)
			if validationErr != nil {
				a.breaker.failure(false, time.Now().UTC())
				settled = true
				return port.ChatResponse{}, apperrors.NewAI(apperrors.AICodeStructuredInvalid, "ai.generate.structured", validationErr)
			}
			response.RepairAttempts = 1
		}
		response.StructuredJSON = append([]byte(nil), structuredJSON...)
	}
	a.breaker.success()
	settled = true
	return response, nil
}

func (a *ChatAdapter) Stream(ctx context.Context, request port.ChatRequest, emit func(port.ChatStreamEvent) error) (err error) {
	if a == nil || a.model == nil {
		return apperrors.NewAI(apperrors.AICodeProviderUnavailable, "ai.stream", errors.New("chat adapter is not initialized"))
	}
	if emit == nil {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.stream", errors.New("stream callback is nil"))
	}
	if request.StructuredOutput != nil {
		return apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.stream.structured", errors.New("structured output is supported by Generate only"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	messages, err := toAgenticMessages(request.Messages)
	if err != nil {
		return err
	}
	options, err := requestOptions(request)
	if err != nil {
		return err
	}
	if !a.breaker.allow(time.Now().UTC()) {
		return apperrors.NewAI(apperrors.AICodeCircuitOpen, "ai.stream", errors.New("provider circuit is open"))
	}
	localRunID := uuid.NewString()
	startedAt := time.Now().UTC()
	var firstTokenAt time.Time
	var finalResponse port.ChatResponse
	settled := false
	defer func() {
		if !settled {
			a.breaker.abandon()
		}
		a.observeRun(ctx, request, localRunID, startedAt, firstTokenAt, finalResponse, err)
	}()
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()
	release, err := a.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	baseEvent := port.ChatStreamEvent{
		Kind:     port.StreamEventMeta,
		RunID:    localRunID,
		Provider: a.config.Provider,
		Protocol: a.config.Protocol,
		Model:    a.config.Model,
	}
	if err := emit(baseEvent); err != nil {
		return err
	}

	for attempt := 0; attempt <= a.config.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := waitRetry(ctx, attempt); err != nil {
				return err
			}
		}
		stream, err := a.model.Stream(ctx, messages, options...)
		if err != nil {
			retryable := shouldRetry(err)
			if retryable && attempt < a.config.MaxRetries {
				continue
			}
			a.breaker.failure(retryable, time.Now().UTC())
			settled = true
			return providerCallError("ai.stream", err)
		}
		if stream == nil {
			err = errors.New("provider returned a nil stream")
			retryable := true
			if attempt < a.config.MaxRetries {
				continue
			}
			a.breaker.failure(retryable, time.Now().UTC())
			settled = true
			return providerCallError("ai.stream", err)
		}

		emittedDelta := false
		var streamChunks []*schema.AgenticMessage
		for {
			chunk, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				stream.Close()
				var finalMessage *schema.AgenticMessage
				if len(streamChunks) > 0 {
					finalMessage, err = schema.ConcatAgenticMessages(streamChunks)
					if err != nil {
						a.breaker.failure(true, time.Now().UTC())
						settled = true
						return providerCallError("ai.stream.concat", err)
					}
				}
				finalResponse, err = normalizeResponse(finalMessage, a.config, localRunID)
				if err != nil {
					a.breaker.failure(true, time.Now().UTC())
					settled = true
					return err
				}
				a.breaker.success()
				settled = true
				if err := emit(port.ChatStreamEvent{
					Kind:          port.StreamEventDone,
					RunID:         localRunID,
					Provider:      finalResponse.Provider,
					Protocol:      finalResponse.Protocol,
					Model:         finalResponse.Model,
					ProviderRunID: finalResponse.ProviderRunID,
					ToolCalls:     append([]port.ToolCall(nil), finalResponse.ToolCalls...),
					Usage:         &finalResponse.Usage,
				}); err != nil {
					return err
				}
				return nil
			}
			if recvErr != nil {
				stream.Close()
				retryable := shouldRetry(recvErr)
				if !emittedDelta && retryable && attempt < a.config.MaxRetries {
					break
				}
				a.breaker.failure(retryable, time.Now().UTC())
				settled = true
				return providerCallError("ai.stream", recvErr)
			}
			if chunk == nil {
				continue
			}
			streamChunks = append(streamChunks, chunk)
			text := assistantText(chunk)
			if text != "" {
				emittedDelta = true
				if firstTokenAt.IsZero() {
					firstTokenAt = time.Now().UTC()
				}
				if err := emit(port.ChatStreamEvent{
					Kind:     port.StreamEventDelta,
					RunID:    localRunID,
					Provider: a.config.Provider,
					Protocol: a.config.Protocol,
					Model:    a.config.Model,
					Text:     text,
				}); err != nil {
					stream.Close()
					return err
				}
			}
			toolCalls := toolCallDeltasFromMessage(chunk)
			if len(toolCalls) > 0 {
				emittedDelta = true
				if firstTokenAt.IsZero() {
					firstTokenAt = time.Now().UTC()
				}
				if err := emit(port.ChatStreamEvent{
					Kind:      port.StreamEventToolCall,
					RunID:     localRunID,
					Provider:  a.config.Provider,
					Protocol:  a.config.Protocol,
					Model:     a.config.Model,
					ToolCalls: toolCalls,
				}); err != nil {
					stream.Close()
					return err
				}
			}
		}
	}
	a.breaker.failure(true, time.Now().UTC())
	settled = true
	return providerCallError("ai.stream", errors.New("stream retry limit reached"))
}

func (a *ChatAdapter) acquire(ctx context.Context) (func(), error) {
	select {
	case a.semaphore <- struct{}{}:
		return func() { <-a.semaphore }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func requestOptions(request port.ChatRequest) ([]einomodel.Option, error) {
	if request.MaxOutputTokens < 0 {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("max output tokens cannot be negative"))
	}
	if request.Temperature != nil && (*request.Temperature < 0 || *request.Temperature > 2) {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("temperature must be between 0 and 2"))
	}
	options := make([]einomodel.Option, 0, 5)
	if request.Model != "" {
		options = append(options, einomodel.WithModel(request.Model))
	}
	if request.MaxOutputTokens > 0 {
		options = append(options, einomodel.WithMaxTokens(request.MaxOutputTokens))
	}
	if request.Temperature != nil {
		options = append(options, einomodel.WithTemperature(*request.Temperature))
	}
	tools, err := toToolInfos(request.Tools)
	if err != nil {
		return nil, err
	}
	if len(tools) > 0 {
		options = append(options, einomodel.WithTools(tools))
	}
	if request.ToolChoice != "" {
		choice, err := toAgenticToolChoice(request.ToolChoice, len(tools))
		if err != nil {
			return nil, err
		}
		options = append(options, einomodel.WithAgenticToolChoice(choice))
	}
	return options, nil
}

func toAgenticMessages(messages []port.ChatMessage) ([]*schema.AgenticMessage, error) {
	if len(messages) == 0 {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("at least one message is required"))
	}
	converted := make([]*schema.AgenticMessage, 0, len(messages))
	for _, message := range messages {
		if len(message.ContentParts) > 0 && message.Role != port.ChatRoleUser {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("content parts are supported only in user messages"))
		}
		switch message.Role {
		case port.ChatRoleSystem:
			converted = append(converted, schema.SystemAgenticMessage(message.Content))
		case port.ChatRoleUser:
			blocks, err := userContentBlocks(message)
			if err != nil {
				return nil, err
			}
			converted = append(converted, &schema.AgenticMessage{
				Role:          schema.AgenticRoleTypeUser,
				ContentBlocks: blocks,
			})
		case port.ChatRoleAssistant:
			blocks, err := assistantContentBlocks(message)
			if err != nil {
				return nil, err
			}
			converted = append(converted, &schema.AgenticMessage{
				Role:          schema.AgenticRoleTypeAssistant,
				ContentBlocks: blocks,
			})
		case port.ChatRoleTool:
			if strings.TrimSpace(message.ToolCallID) == "" {
				return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("tool message requires a tool call ID"))
			}
			converted = append(converted, &schema.AgenticMessage{
				Role: schema.AgenticRoleTypeUser,
				ContentBlocks: []*schema.ContentBlock{
					schema.NewContentBlock(&schema.FunctionToolResult{
						CallID: message.ToolCallID,
						Name:   message.Name,
						Content: []*schema.FunctionToolResultContentBlock{{
							Type: schema.FunctionToolResultContentBlockTypeText,
							Text: &schema.UserInputText{Text: message.Content},
						}},
					}),
				},
			})
		default:
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("unsupported message role %q", message.Role))
		}
	}
	return converted, nil
}

func userContentBlocks(message port.ChatMessage) ([]*schema.ContentBlock, error) {
	if len(message.ContentParts) > maxChatContentParts {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("too many content parts"))
	}
	blocks := make([]*schema.ContentBlock, 0, len(message.ContentParts)+1)
	imagePayloadBytes := 0
	if message.Content != "" {
		blocks = append(blocks, schema.NewContentBlock(&schema.UserInputText{Text: message.Content}))
	}
	for index, part := range message.ContentParts {
		switch part.Type {
		case port.ChatMessagePartTypeText:
			if part.Text == "" {
				return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("text content part %d is empty", index))
			}
			blocks = append(blocks, schema.NewContentBlock(&schema.UserInputText{Text: part.Text}))
		case port.ChatMessagePartTypeImageURL:
			image, err := normalizeImagePart(part, index)
			if err != nil {
				return nil, err
			}
			imagePayloadBytes += len(image.URL) + len(image.Base64Data)
			if imagePayloadBytes > maxImageRequestPayloadBytes {
				return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("image content exceeds the request size limit"))
			}
			blocks = append(blocks, schema.NewContentBlock(image))
		default:
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("unsupported content part type at index %d", index))
		}
	}
	if len(blocks) == 0 {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("user message requires text or content parts"))
	}
	return blocks, nil
}

func normalizeImagePart(part port.ChatMessagePart, index int) (*schema.UserInputImage, error) {
	imageURL := strings.TrimSpace(part.URL)
	base64Data := strings.TrimSpace(part.Base64Data)
	if (imageURL == "") == (base64Data == "") {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("image content part %d must contain exactly one URL or base64 value", index))
	}
	if imageURL != "" {
		if len(imageURL) > maxImageURLLength {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("image URL content part %d is too long", index))
		}
		if err := validateRemoteImageURL(imageURL); err != nil {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("image URL content part %d is invalid", index))
		}
	}
	mimeType := strings.ToLower(strings.TrimSpace(part.MIMEType))
	if base64Data != "" {
		if len(base64Data) > maxImageBase64ByteLength {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("base64 image content part %d is too large", index))
		}
		if mimeType == "" || !supportedImageMIMEType(mimeType) {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("base64 image content part %d has an unsupported MIME type", index))
		}
		decoded, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("base64 image content part %d is invalid", index))
		}
		if len(decoded) > maxImageDecodedByteLength {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("base64 image content part %d is too large", index))
		}
	}
	if mimeType != "" && !supportedImageMIMEType(mimeType) {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("image content part %d has an unsupported MIME type", index))
	}
	detail := schema.ImageURLDetailAuto
	switch strings.ToLower(strings.TrimSpace(part.Detail)) {
	case "", "auto":
	case "low":
		detail = schema.ImageURLDetailLow
	case "high":
		detail = schema.ImageURLDetailHigh
	default:
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("image content part %d has an unsupported detail", index))
	}
	return &schema.UserInputImage{
		URL:        imageURL,
		Base64Data: base64Data,
		MIMEType:   mimeType,
		Detail:     detail,
	}, nil
}

func validateRemoteImageURL(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("image URL must be an HTTP(S) URL")
	}
	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return errors.New("private image hosts are not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return errors.New("private image hosts are not allowed")
	}
	return nil
}

func supportedImageMIMEType(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func assistantContentBlocks(message port.ChatMessage) ([]*schema.ContentBlock, error) {
	blocks := make([]*schema.ContentBlock, 0, len(message.ToolCalls)+1)
	if message.Content != "" || len(message.ToolCalls) == 0 {
		blocks = append(blocks, schema.NewContentBlock(&schema.AssistantGenText{Text: message.Content}))
	}
	for _, call := range message.ToolCalls {
		id := strings.TrimSpace(call.ID)
		name := strings.TrimSpace(call.Name)
		if id == "" || name == "" {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", errors.New("assistant tool calls require an ID and name"))
		}
		arguments, err := normalizedToolArguments(call.Arguments)
		if err != nil {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request", fmt.Errorf("tool %q arguments: %w", name, err))
		}
		blocks = append(blocks, schema.NewContentBlock(&schema.FunctionToolCall{
			CallID:    id,
			Name:      name,
			Arguments: string(arguments),
		}))
	}
	return blocks, nil
}

func toToolInfos(definitions []port.ToolDefinition) ([]*schema.ToolInfo, error) {
	if len(definitions) == 0 {
		return nil, nil
	}
	tools := make([]*schema.ToolInfo, 0, len(definitions))
	seen := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", errors.New("tool name is required"))
		}
		if len(name) > 64 {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", fmt.Errorf("tool %q name exceeds 64 bytes", name))
		}
		if _, exists := seen[name]; exists {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", fmt.Errorf("duplicate tool name %q", name))
		}
		seen[name] = struct{}{}
		tool := &schema.ToolInfo{
			Name: name,
			Desc: strings.TrimSpace(definition.Description),
		}
		if len(definition.Parameters) > 0 {
			var object map[string]json.RawMessage
			if err := json.Unmarshal(definition.Parameters, &object); err != nil || object == nil {
				if err == nil {
					err = errors.New("schema must be a JSON object")
				}
				return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", fmt.Errorf("tool %q parameters: %w", name, err))
			}
			parameters := &einoschema.Schema{}
			if err := json.Unmarshal(definition.Parameters, parameters); err != nil {
				return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", fmt.Errorf("tool %q parameters: %w", name, err))
			}
			tool.ParamsOneOf = schema.NewParamsOneOfByJSONSchema(parameters)
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func toAgenticToolChoice(choice port.ToolChoice, toolCount int) (*schema.AgenticToolChoice, error) {
	var toolChoice schema.ToolChoice
	switch choice {
	case port.ToolChoiceAuto:
		toolChoice = schema.ToolChoiceAllowed
	case port.ToolChoiceNone:
		toolChoice = schema.ToolChoiceForbidden
	case port.ToolChoiceRequired:
		if toolCount == 0 {
			return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", errors.New("required tool choice needs at least one tool"))
		}
		toolChoice = schema.ToolChoiceForced
	default:
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "ai.request.tools", fmt.Errorf("unsupported tool choice %q", choice))
	}
	return &schema.AgenticToolChoice{Type: toolChoice}, nil
}

func normalizedToolArguments(arguments json.RawMessage) ([]byte, error) {
	trimmed := strings.TrimSpace(string(arguments))
	if trimmed == "" {
		return []byte("{}"), nil
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, errors.New("arguments must be valid JSON")
	}
	return []byte(trimmed), nil
}

func normalizeResponse(message *schema.AgenticMessage, config Config, runID string) (port.ChatResponse, error) {
	if message == nil {
		return port.ChatResponse{}, providerCallError("ai.response", errors.New("provider returned an empty response"))
	}
	if strings.TrimSpace(runID) == "" {
		runID = uuid.NewString()
	}
	response := port.ChatResponse{
		Provider: config.Provider,
		Protocol: config.Protocol,
		Model:    config.Model,
		RunID:    runID,
		Text:     assistantText(message),
	}
	toolCalls, err := toolCallsFromMessage(message)
	if err != nil {
		return port.ChatResponse{}, providerCallError("ai.response.tool_calls", err)
	}
	response.ToolCalls = toolCalls
	if message.ResponseMeta != nil {
		if usage := message.ResponseMeta.TokenUsage; usage != nil {
			response.Usage = port.TokenUsage{
				InputTokens:  usage.PromptTokens,
				OutputTokens: usage.CompletionTokens,
				TotalTokens:  usage.TotalTokens,
			}
		}
		switch extension := message.ResponseMeta.Extension.(type) {
		case *agenticopenai.ChatResponseMetaExtension:
			response.ProviderRunID = extension.ID
			response.FinishReason = extension.FinishReason
		}
		if extension := message.ResponseMeta.OpenAIExtension; extension != nil {
			response.ProviderRunID = extension.ID
			response.FinishReason = string(extension.Status)
		}
		if extension := message.ResponseMeta.ClaudeExtension; extension != nil {
			response.ProviderRunID = extension.ID
			response.FinishReason = extension.StopReason
		}
	}
	return response, nil
}

func toolCallsFromMessage(message *schema.AgenticMessage) ([]port.ToolCall, error) {
	if message == nil {
		return nil, nil
	}
	var calls []port.ToolCall
	for _, block := range message.ContentBlocks {
		if block == nil || block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
			continue
		}
		call := block.FunctionToolCall
		id := strings.TrimSpace(call.CallID)
		name := strings.TrimSpace(call.Name)
		if id == "" || name == "" {
			return nil, errors.New("provider tool call is missing an ID or name")
		}
		arguments, err := normalizedToolArguments([]byte(call.Arguments))
		if err != nil {
			return nil, fmt.Errorf("provider tool %q arguments: %w", name, err)
		}
		calls = append(calls, port.ToolCall{ID: id, Name: name, Arguments: arguments})
	}
	return calls, nil
}

func toolCallDeltasFromMessage(message *schema.AgenticMessage) []port.ToolCall {
	if message == nil {
		return nil
	}
	var calls []port.ToolCall
	for _, block := range message.ContentBlocks {
		if block == nil || block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
			continue
		}
		call := block.FunctionToolCall
		calls = append(calls, port.ToolCall{
			ID:        strings.TrimSpace(call.CallID),
			Name:      strings.TrimSpace(call.Name),
			Arguments: []byte(call.Arguments),
		})
	}
	return calls
}

func assistantText(message *schema.AgenticMessage) string {
	if message == nil {
		return ""
	}
	var builder strings.Builder
	for _, block := range message.ContentBlocks {
		if block != nil && block.AssistantGenText != nil {
			builder.WriteString(block.AssistantGenText.Text)
		}
	}
	return builder.String()
}

func shouldRetry(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if status, ok := providerStatusCode(err); ok {
		return retryableHTTPStatus(status)
	}
	// Transport and decoding errors do not expose an HTTP status. They are
	// retryable before the first streamed delta; callers still bound the number
	// of attempts and the overall timeout.
	return true
}

func providerStatusCode(err error) (int, bool) {
	var chatError *chatopenai.APIError
	if errors.As(err, &chatError) && chatError != nil {
		return chatError.HTTPStatusCode, true
	}
	var openAIError *openai.Error
	if errors.As(err, &openAIError) && openAIError != nil {
		return openAIError.StatusCode, true
	}
	var anthropicError *anthropic.Error
	if errors.As(err, &anthropicError) && anthropicError != nil {
		return anthropicError.StatusCode, true
	}
	return 0, false
}

func retryableHTTPStatus(status int) bool {
	switch {
	case status == http.StatusRequestTimeout,
		status == http.StatusConflict,
		status == http.StatusTooManyRequests,
		status >= http.StatusInternalServerError:
		return true
	case status >= http.StatusBadRequest:
		return false
	default:
		return false
	}
}

func waitRetry(ctx context.Context, attempt int) error {
	delay := 50 * time.Millisecond * time.Duration(1<<(attempt-1))
	if delay > 500*time.Millisecond {
		delay = 500 * time.Millisecond
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func providerCallError(op string, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		// Provider errors may contain response bodies or echoed request details;
		// keep those behind the typed error for callers and log only a stable code.
		slog.Debug("AI provider call failed", "operation", op, "error_code", runErrorCode(err))
	}
	return apperrors.WrapAIUnavailable(op, err)
}

func (a *ChatAdapter) observeRun(ctx context.Context, request port.ChatRequest, runID string, startedAt, firstTokenAt time.Time, response port.ChatResponse, err error) {
	if a == nil || a.config.Observer == nil {
		return
	}
	completedAt := time.Now().UTC()
	run := port.AgentRun{
		ID:             runID,
		RequestID:      request.Metadata["request_id"],
		UseCase:        request.UseCase,
		Provider:       a.config.Provider,
		Protocol:       a.config.Protocol,
		Model:          a.config.Model,
		ProviderRunID:  response.ProviderRunID,
		Status:         runStatus(err),
		Usage:          response.Usage,
		FirstTokenAt:   firstTokenAt,
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
		ErrorCode:      runErrorCode(err),
		SanitizedError: runErrorCode(err),
	}
	run.Duration = completedAt.Sub(startedAt)
	if !firstTokenAt.IsZero() {
		run.FirstTokenLatency = firstTokenAt.Sub(startedAt)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// A request can be canceled immediately after the provider returns. The
	// observer still receives the completed snapshot without inheriting that
	// cancellation.
	a.config.Observer.ObserveAIRun(context.WithoutCancel(ctx), run)
}

func runStatus(err error) port.AIRunStatus {
	if err == nil {
		return port.AIRunSucceeded
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return port.AIRunCanceled
	}
	return port.AIRunFailed
}

func runErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if code := apperrors.AICodeOf(err); code != "" {
		return string(code)
	}
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "context_deadline_exceeded"
	}
	return string(apperrors.AICodeProviderUnavailable)
}
