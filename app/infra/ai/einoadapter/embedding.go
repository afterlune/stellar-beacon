package einoadapter

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	openaiembedding "github.com/cloudwego/eino-ext/libs/acl/openai"
)

const (
	// DefaultEmbeddingMaxInputBytes keeps direct gateway callers within the
	// same request-size envelope as the article index projector.
	DefaultEmbeddingMaxInputBytes = 256 << 10
	maxEmbeddingInputBytes        = 4 << 20
	// AliBailian's compatible embedding endpoint currently accepts at most
	// sixteen inputs per HTTP request. Keep the application-level batch size
	// independent from this provider transport limit and split transparently
	// so a larger, already-persisted index contract remains resumable.
	maxAliBailianEmbeddingInputsPerRequest = 16
)

// EmbeddingConfig keeps the Eino embedding client and provider details behind
// the infra boundary. Application packages only see port.EmbeddingGateway.
type EmbeddingConfig struct {
	Provider      string
	Model         string
	APIKey        string
	BaseURL       string
	Dimension     int
	Timeout       time.Duration
	MaxConcurrent int
	MaxInputBytes int
	HTTPClient    *http.Client
}

type EmbeddingAdapter struct {
	client    *openaiembedding.EmbeddingClient
	config    EmbeddingConfig
	semaphore chan struct{}
}

var _ port.EmbeddingGateway = (*EmbeddingAdapter)(nil)

func NewOpenAIEmbedding(ctx context.Context, config EmbeddingConfig) (*EmbeddingAdapter, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config.Provider = strings.ToLower(strings.TrimSpace(config.Provider))
	config.Model = strings.TrimSpace(config.Model)
	if config.Provider != "openai" && config.Provider != "sglang" && config.Provider != "alibailian" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding provider is not OpenAI-compatible"))
	}
	if config.Model == "" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding model is required"))
	}
	if (config.Provider == "openai" || config.Provider == "alibailian") && strings.TrimSpace(config.APIKey) == "" {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding API key is required"))
	}
	if config.Dimension < 0 {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding dimension cannot be negative"))
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = 2
	}
	if config.MaxConcurrent > 128 {
		config.MaxConcurrent = 128
	}
	if config.MaxInputBytes < 0 {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding input byte limit cannot be negative"))
	}
	if config.MaxInputBytes == 0 {
		config.MaxInputBytes = DefaultEmbeddingMaxInputBytes
	}
	if config.MaxInputBytes > maxEmbeddingInputBytes {
		return nil, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.configure", errors.New("embedding input byte limit exceeds the safety maximum"))
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	clientConfig := &openaiembedding.EmbeddingConfig{
		APIKey:     config.APIKey,
		BaseURL:    config.BaseURL,
		Model:      config.Model,
		HTTPClient: config.HTTPClient,
	}
	// SGLang's OpenAI-compatible embedding endpoint fixes the output
	// dimension to the model dimension and rejects the optional OpenAI
	// `dimensions` request field. Keep the returned-vector dimension check
	// below for every provider, but omit this request hint for SGLang.
	if config.Dimension > 0 && config.Provider != "sglang" {
		dimension := config.Dimension
		clientConfig.Dimensions = &dimension
	}
	client, err := openaiembedding.NewEmbeddingClient(ctx, clientConfig)
	if err != nil {
		return nil, apperrors.WrapAIUnavailable("embedding.configure", err)
	}
	return &EmbeddingAdapter{
		client:    client,
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrent),
	}, nil
}

func (a *EmbeddingAdapter) Embed(ctx context.Context, request port.EmbeddingRequest) (port.EmbeddingResult, error) {
	if a == nil || a.client == nil {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding adapter is not initialized"))
	}
	if len(request.Inputs) == 0 {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.embed", errors.New("at least one input is required"))
	}
	for _, input := range request.Inputs {
		if strings.TrimSpace(input) == "" {
			return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.embed", errors.New("embedding inputs must not be empty"))
		}
	}
	inputBytes := 0
	for _, input := range request.Inputs {
		if len(input) > a.config.MaxInputBytes-inputBytes {
			return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.embed", errors.New("embedding input exceeds the configured byte limit"))
		}
		inputBytes += len(input)
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = a.config.Model
	}
	if model != a.config.Model {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeInvalidRequest, "embedding.embed", errors.New("request model does not match the resolved embedding route"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()
	select {
	case a.semaphore <- struct{}{}:
		defer func() { <-a.semaphore }()
	case <-ctx.Done():
		return port.EmbeddingResult{}, ctx.Err()
	}

	vectors, err := a.embedProviderInputs(ctx, request.Inputs)
	if err != nil {
		return port.EmbeddingResult{}, apperrors.WrapAIUnavailable("embedding.embed", err)
	}
	if len(vectors) != len(request.Inputs) {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding provider returned a different number of vectors"))
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding provider returned empty vectors"))
	}
	dimension := len(vectors[0])
	if a.config.Dimension > 0 && dimension != a.config.Dimension {
		return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding provider returned an incompatible vector dimension"))
	}
	result := make([][]float32, len(vectors))
	for index, vector := range vectors {
		if len(vector) != dimension {
			return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding provider returned inconsistent vector dimensions"))
		}
		result[index] = make([]float32, len(vector))
		for position, value := range vector {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding provider returned a non-finite vector value"))
			}
			converted := float32(value)
			if math.IsNaN(float64(converted)) || math.IsInf(float64(converted), 0) {
				return port.EmbeddingResult{}, apperrors.NewAI(apperrors.AICodeProviderUnavailable, "embedding.embed", errors.New("embedding vector cannot be represented as float32"))
			}
			result[index][position] = converted
		}
	}
	return port.EmbeddingResult{
		Vectors:   result,
		Model:     model,
		Dimension: dimension,
		Version:   strings.TrimSpace(request.Version),
	}, nil
}

func (a *EmbeddingAdapter) embedProviderInputs(ctx context.Context, inputs []string) ([][]float64, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("embedding adapter is not initialized")
	}
	batchSize := len(inputs)
	if a.config.Provider == "alibailian" && batchSize > maxAliBailianEmbeddingInputsPerRequest {
		batchSize = maxAliBailianEmbeddingInputsPerRequest
	}
	vectors := make([][]float64, 0, len(inputs))
	for start := 0; start < len(inputs); start += batchSize {
		end := start + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		batchVectors, err := a.client.EmbedStrings(ctx, inputs[start:end])
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, batchVectors...)
	}
	return vectors, nil
}
