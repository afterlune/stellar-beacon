package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
)

const (
	articleChunksSearchResponseLimit       = 4 << 20
	articleChunksSearchFilter              = ArticleChunkIsDeleteField + " = 0 AND " + ArticleChunkStatusField + " = 1"
	articleChunksEmbeddingOperation        = "search.article_chunks.embedding"
	articleChunksEmbeddingResolveOperation = "search.article_chunks.embedding.resolve"
)

// MeiliArticleChunkSearcher reads the explicitly versioned article_chunks
// index. It is separate from MeiliArticleChunkIndex so public reads can be
// composed without granting the application layer index-management methods.
// Construction never contacts Meilisearch or the embedding provider.
type MeiliArticleChunkSearcher struct {
	baseURL       string
	apiKey        string
	httpClient    *http.Client
	spec          ArticleChunksIndexSpec
	router        port.ModelRouter
	semanticRatio float32
	metrics       *SearchMetricsObserver
}

type articleChunksSearchRequest struct {
	Query                 string                         `json:"q"`
	AttributesToRetrieve  []string                       `json:"attributesToRetrieve,omitempty"`
	AttributesToSearchOn  []string                       `json:"attributesToSearchOn,omitempty"`
	Limit                 int                            `json:"limit,omitempty"`
	Filter                string                         `json:"filter,omitempty"`
	AttributesToHighlight []string                       `json:"attributesToHighlight,omitempty"`
	HighlightPreTag       string                         `json:"highlightPreTag,omitempty"`
	HighlightPostTag      string                         `json:"highlightPostTag,omitempty"`
	ShowRankingScore      bool                           `json:"showRankingScore,omitempty"`
	Vector                []float32                      `json:"vector,omitempty"`
	Hybrid                *articleChunksSearchHybridSpec `json:"hybrid,omitempty"`
}

type articleChunksSearchHybridSpec struct {
	SemanticRatio float64 `json:"semanticRatio"`
	Embedder      string  `json:"embedder"`
}

type articleChunksSearchResponse struct {
	Hits json.RawMessage `json:"hits"`
}

type articleChunksSearchHit struct {
	ID              string   `json:"id"`
	ArticleID       int      `json:"articleId"`
	ArticleTitle    string   `json:"articleTitle"`
	Text            string   `json:"text"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	ArticleURL      string   `json:"articleUrl"`
	PublishedAtUnix int64    `json:"publishedAtUnix"`
	ChunkIndex      int      `json:"chunkIndex"`
	Status          int      `json:"status"`
	IsDelete        int      `json:"isDelete"`
	// Meilisearch can include every retrieved field in _formatted, even when
	// only title/text were requested for highlighting. Arrays such as tags are
	// therefore valid members of this object and must not make the whole hit
	// fail JSON decoding.
	Formatted    map[string]json.RawMessage `json:"_formatted"`
	RankingScore float64                    `json:"_rankingScore"`
}

var _ port.KnowledgeSearcher = (*MeiliArticleChunkSearcher)(nil)

func NewMeiliArticleChunkSearcher(conf *config.MeiliSearch, spec ArticleChunksIndexSpec, router port.ModelRouter) (*MeiliArticleChunkSearcher, error) {
	return NewMeiliArticleChunkSearcherWithSemanticRatio(conf, spec, router, port.DefaultSemanticRatio)
}

// NewMeiliArticleChunkSearcherWithSemanticRatio injects the configured
// default hybrid ratio. A zero value uses the safe 0.65 default; other values
// must be in the open interval (0, 1].
func NewMeiliArticleChunkSearcherWithSemanticRatio(conf *config.MeiliSearch, spec ArticleChunksIndexSpec, router port.ModelRouter, semanticRatio float32) (*MeiliArticleChunkSearcher, error) {
	if conf == nil {
		return NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio("", "", nil, spec, router, semanticRatio)
	}
	return NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio(conf.URL, conf.ApiKey, nil, spec, router, semanticRatio)
}

// NewMeiliArticleChunkSearcherWithHTTPClient makes the adapter deterministic
// in tests and lets callers provide transport-level timeouts and tracing.
func NewMeiliArticleChunkSearcherWithHTTPClient(baseURL, apiKey string, client *http.Client, spec ArticleChunksIndexSpec, router port.ModelRouter) (*MeiliArticleChunkSearcher, error) {
	return NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio(baseURL, apiKey, client, spec, router, port.DefaultSemanticRatio)
}

func NewMeiliArticleChunkSearcherWithHTTPClientAndSemanticRatio(baseURL, apiKey string, client *http.Client, spec ArticleChunksIndexSpec, router port.ModelRouter, semanticRatio float32) (*MeiliArticleChunkSearcher, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if semanticRatio == 0 {
		semanticRatio = port.DefaultSemanticRatio
	}
	if semanticRatio <= 0 || semanticRatio > 1 {
		return nil, apperrors.Invalid("search.article_chunks.semantic_ratio", "semantic ratio must be between 0 and 1")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &MeiliArticleChunkSearcher{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:        strings.TrimSpace(apiKey),
		httpClient:    client,
		spec:          spec,
		router:        router,
		semanticRatio: semanticRatio,
	}, nil
}

// SetMetricsObserver attaches the bounded runtime observer used by the
// operational diagnostics provider. The observer receives only index, mode,
// duration and outcome.
func (s *MeiliArticleChunkSearcher) SetMetricsObserver(observer *SearchMetricsObserver) {
	if s != nil {
		s.metrics = observer
	}
}

func (s *MeiliArticleChunkSearcher) Search(ctx context.Context, query port.KnowledgeQuery) (hits []port.KnowledgeHit, err error) {
	started := time.Now()
	if s != nil && s.metrics != nil {
		defer func() {
			mode := query.Mode
			if normalized, normalizeErr := query.NormalizeWithDefault(s.spec.UID, s.semanticRatio); normalizeErr == nil {
				mode = normalized.Mode
			}
			s.metrics.Observe(ctx, s.spec.UID, mode, time.Since(started), err)
		}()
	}
	if s == nil {
		return nil, apperrors.Unavailable("search.article_chunks", fmt.Errorf("search client is not configured"))
	}
	if err := s.spec.Validate(); err != nil {
		return nil, err
	}
	normalized, err := query.NormalizeWithDefault(s.spec.UID, s.semanticRatio)
	if err != nil {
		return nil, apperrors.Invalid("search.article_chunks.query", err.Error())
	}
	if normalized.Index != s.spec.UID {
		return nil, apperrors.Invalid("search.article_chunks.index", "query index does not match the configured article chunk index")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	request := articleChunksSearchRequest{
		Query:                 normalized.Query,
		AttributesToRetrieve:  articleChunksSearchRetrievableFields(),
		AttributesToSearchOn:  []string{ArticleChunkTitleField, ArticleChunkTextField},
		Limit:                 normalized.Limit,
		Filter:                articleChunksFilterExpression(normalized.Filter),
		AttributesToHighlight: []string{ArticleChunkTitleField, ArticleChunkTextField},
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
		ShowRankingScore:      true,
	}
	switch normalized.Mode {
	case port.SearchModeKeyword:
		// The explicit keyword mode is useful for a versioned chunk index, but
		// the existing public endpoint continues to use the legacy articles
		// index through MeiliSearcher.Search.
	case port.SearchModeHybrid, port.SearchModeSemantic:
		vector, err := s.embedQuery(ctx, normalized.Query)
		if err != nil {
			return nil, err
		}
		request.Vector = vector
		request.Hybrid = &articleChunksSearchHybridSpec{
			SemanticRatio: float64(normalized.SemanticRatio),
			Embedder:      "default",
		}
		if normalized.Mode == port.SearchModeSemantic {
			// Meilisearch treats an empty q with ratio 1 as pure semantic
			// search. The vector remains the user-provided query embedding.
			request.Query = ""
		}
	default:
		return nil, apperrors.Invalid("search.article_chunks.mode", "unsupported search mode")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.encode", err)
	}
	responseBody, err := s.doSearch(ctx, body)
	if err != nil {
		return nil, err
	}
	var decoded articleChunksSearchResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.decode", err)
	}
	if err := validateMeiliSearchHits(decoded.Hits); err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.decode", err)
	}
	var decodedHits []articleChunksSearchHit
	if err := json.Unmarshal(decoded.Hits, &decodedHits); err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.decode", err)
	}
	return decodeArticleChunkSearchHits(s.spec.UID, decodedHits), nil
}

func (s *MeiliArticleChunkSearcher) embedQuery(ctx context.Context, query string) ([]float32, error) {
	if s.router == nil {
		return nil, apperrors.Unavailable(articleChunksEmbeddingOperation, fmt.Errorf("embedding model router is not configured"))
	}
	gateway, route, err := s.router.ResolveEmbedding(ctx, port.AIUseCaseEmbedding)
	if err != nil {
		return nil, apperrors.WrapUnavailable(articleChunksEmbeddingResolveOperation, err)
	}
	if gateway == nil {
		return nil, apperrors.Unavailable(articleChunksEmbeddingOperation, fmt.Errorf("embedding gateway is not configured"))
	}
	if route.Provider != s.spec.Provider || route.Model != s.spec.Model {
		return nil, apperrors.Invalid("search.article_chunks.embedding_contract", "resolved embedding route does not match the index specification")
	}
	result, err := gateway.Embed(ctx, port.EmbeddingRequest{
		Inputs:  []string{query},
		Model:   s.spec.Model,
		Version: s.spec.ModelVersion,
	})
	if err != nil {
		return nil, apperrors.WrapUnavailable(articleChunksEmbeddingOperation, err)
	}
	if len(result.Vectors) != 1 {
		return nil, apperrors.Unavailable("search.article_chunks.embedding_contract", fmt.Errorf("embedding provider returned %d vectors, want 1", len(result.Vectors)))
	}
	if strings.TrimSpace(result.Model) != s.spec.Model || strings.TrimSpace(result.Version) != s.spec.ModelVersion || result.Dimension != s.spec.Dimension {
		return nil, apperrors.Unavailable("search.article_chunks.embedding_contract", fmt.Errorf("embedding provider returned an incompatible model contract"))
	}
	if err := s.spec.ValidateVector(result.Vectors[0]); err != nil {
		return nil, apperrors.Unavailable("search.article_chunks.embedding_contract", err)
	}
	return append([]float32(nil), result.Vectors[0]...), nil
}

func (s *MeiliArticleChunkSearcher) doSearch(ctx context.Context, body []byte) ([]byte, error) {
	if strings.TrimSpace(s.baseURL) == "" {
		return nil, apperrors.Unavailable("search.article_chunks", fmt.Errorf("Meilisearch URL is not configured"))
	}
	endpoint := s.baseURL + "/indexes/" + url.PathEscape(s.spec.UID) + "/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, apperrors.WrapUnavailable("search.article_chunks.request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, apperrors.WrapUnavailable("search.article_chunks.request", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, articleChunksSearchResponseLimit))
	if err != nil {
		return nil, apperrors.WrapUnavailable("search.article_chunks.response", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, apperrors.Unavailable("search.article_chunks", fmt.Errorf("Meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	return responseBody, nil
}

func articleChunksFilterExpression(filter port.KnowledgeFilter) string {
	clauses := []string{articleChunksSearchFilter}
	if filter.Category != "" {
		clauses = append(clauses, ArticleChunkCategoryField+" = "+strconv.Quote(filter.Category))
	}
	for _, tag := range filter.Tags {
		clauses = append(clauses, ArticleChunkTagsField+" = "+strconv.Quote(tag))
	}
	if filter.Year > 0 {
		start := time.Date(filter.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(1, 0, 0)
		clauses = append(clauses,
			ArticleChunkPublishedAtUnixField+" >= "+strconv.FormatInt(start.Unix(), 10),
			ArticleChunkPublishedAtUnixField+" < "+strconv.FormatInt(end.Unix(), 10),
		)
	}
	if !filter.From.IsZero() {
		clauses = append(clauses, ArticleChunkPublishedAtUnixField+" >= "+strconv.FormatInt(filterTimestampBound(filter.From), 10))
	}
	if !filter.To.IsZero() {
		clauses = append(clauses, ArticleChunkPublishedAtUnixField+" < "+strconv.FormatInt(filterTimestampBound(filter.To), 10))
	}
	return strings.Join(clauses, " AND ")
}

func filterTimestampBound(value time.Time) int64 {
	value = value.UTC()
	seconds := value.Unix()
	if value.Nanosecond() != 0 {
		return seconds + 1
	}
	return seconds
}

func articleChunksSearchRetrievableFields() []string {
	return []string{
		ArticleChunkIDField,
		ArticleChunkArticleIDField,
		ArticleChunkTitleField,
		ArticleChunkTextField,
		ArticleChunkCategoryField,
		ArticleChunkTagsField,
		ArticleChunkURLField,
		ArticleChunkPublishedAtField,
		ArticleChunkPublishedAtUnixField,
		ArticleChunkIndexField,
		ArticleChunkStatusField,
		ArticleChunkIsDeleteField,
	}
}

func decodeArticleChunkSearchHits(index string, hits []articleChunksSearchHit) []port.KnowledgeHit {
	result := make([]port.KnowledgeHit, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	for _, hit := range hits {
		id := strings.TrimSpace(hit.ID)
		if id == "" || hit.ArticleID <= 0 || strings.TrimSpace(hit.ArticleTitle) == "" || strings.TrimSpace(hit.Text) == "" || !port.IsPublicArticle(hit.Status, hit.IsDelete) {
			// Search indexes can contain stale documents during a lifecycle
			// race. Fail closed and leave repair to the index worker.
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		metadata := map[string]string{
			"articleId":       strconv.Itoa(hit.ArticleID),
			"category":        strings.TrimSpace(hit.Category),
			"tags":            strings.Join(hit.Tags, ","),
			"publishedAtUnix": strconv.FormatInt(hit.PublishedAtUnix, 10),
			"chunkIndex":      strconv.Itoa(hit.ChunkIndex),
			"status":          strconv.Itoa(hit.Status),
			"isDelete":        strconv.Itoa(hit.IsDelete),
		}
		result = append(result, port.KnowledgeHit{
			Index:      strings.TrimSpace(index),
			ID:         id,
			Title:      hit.ArticleTitle,
			Text:       hit.Text,
			URL:        hit.ArticleURL,
			Score:      hit.RankingScore,
			Highlights: copyFormattedStrings(hit.Formatted),
			Metadata:   metadata,
		})
	}
	return result
}

func copyFormattedStrings(source map[string]json.RawMessage) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, rawValue := range source {
		var value string
		if err := json.Unmarshal(rawValue, &value); err != nil {
			// _formatted also contains arrays and other non-highlighted
			// values. Only string values are useful to the public result.
			continue
		}
		result[key] = value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
