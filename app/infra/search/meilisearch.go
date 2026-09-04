package search

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const (
	preTag  = "<mark>"
	postTag = "</mark>"
)

type MeiliSearcher struct {
	client     meilisearch.ServiceManager
	baseURL    string
	apiKey     string
	httpClient *http.Client
	knowledge  port.KnowledgeSearcher
	metrics    *SearchMetricsObserver
}

func NewMeiliSearcher(conf *config.MeiliSearch) *MeiliSearcher {
	if conf == nil {
		return &MeiliSearcher{}
	}
	return &MeiliSearcher{
		baseURL:    strings.TrimRight(conf.URL, "/"),
		apiKey:     conf.ApiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func NewMeiliSearcherWithClient(client meilisearch.ServiceManager) *MeiliSearcher {
	return &MeiliSearcher{client: client}
}

// SetArticleChunkSearcher attaches the optional versioned vector index. The
// legacy articles keyword search remains available when it is nil.
func (s *MeiliSearcher) SetArticleChunkSearcher(searcher port.KnowledgeSearcher) {
	if s != nil {
		s.knowledge = searcher
	}
}

// SetMetricsObserver attaches the bounded runtime observer. It does not
// retain query text or filters and is optional for callers that only need the
// search port.
func (s *MeiliSearcher) SetMetricsObserver(observer *SearchMetricsObserver) {
	if s != nil {
		s.metrics = observer
	}
}

func (s *MeiliSearcher) SearchWithMode(ctx context.Context, keywords string, mode port.SearchMode) ([]port.ArticleSearchHit, error) {
	return s.SearchWithModeAndFilter(ctx, keywords, mode, port.KnowledgeFilter{})
}

func (s *MeiliSearcher) SearchWithModeAndFilter(ctx context.Context, keywords string, mode port.SearchMode, filter port.KnowledgeFilter) ([]port.ArticleSearchHit, error) {
	if s == nil {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	normalizedMode, err := port.NormalizeSearchMode(string(mode))
	if err != nil {
		return nil, errors.Invalid("search.articles.mode", err.Error())
	}
	keywords = strings.TrimSpace(keywords)
	if keywords == "" {
		return nil, errors.Invalid("search.articles.query", "search query is required")
	}
	normalizedFilter, err := filter.Normalize()
	if err != nil {
		return nil, errors.Invalid("search.articles.filter", err.Error())
	}
	if normalizedMode == port.SearchModeKeyword && normalizedFilter.Empty() {
		return s.Search(ctx, keywords)
	}
	if s.knowledge == nil {
		return nil, errors.Unavailable("search.article_chunks", fmt.Errorf("article chunk search is not configured"))
	}
	hits, err := s.knowledge.Search(ctx, port.KnowledgeQuery{Query: keywords, Mode: normalizedMode, Filter: normalizedFilter})
	if err != nil {
		if !embeddingSearchFallbackEligible(normalizedMode, err) {
			return nil, classifyKnowledgeSearchError(err)
		}
		slog.WarnContext(ctx, "article search degraded to keyword mode", "requested_mode", normalizedMode, "error_kind", errors.KindOf(err), "ai_code", errors.AICodeOf(err))
		return s.keywordFallback(ctx, keywords, normalizedFilter)
	}
	return articleSearchHitsFromKnowledge(hits, normalizedMode), nil
}

func (s *MeiliSearcher) keywordFallback(ctx context.Context, keywords string, filter port.KnowledgeFilter) ([]port.ArticleSearchHit, error) {
	if !filter.Empty() {
		// The legacy articles index intentionally does not contain category,
		// tag, or publication-time fields. Keep filters intact by retrying the
		// versioned chunk index in keyword mode instead of returning unfiltered
		// legacy results.
		hits, err := s.knowledge.Search(ctx, port.KnowledgeQuery{
			Query:  keywords,
			Mode:   port.SearchModeKeyword,
			Filter: filter,
		})
		if err != nil {
			return nil, classifyKnowledgeSearchError(err)
		}
		return articleSearchHitsFromKnowledge(hits, port.SearchModeKeyword), nil
	}

	// With no structured filters, use the stable legacy index. This keeps the
	// fallback available even when the versioned Chunk index has not yet been
	// provisioned, while Search still classifies a real Meilisearch outage.
	hits, err := s.Search(ctx, keywords)
	if err != nil {
		return nil, err
	}
	for index := range hits {
		if hits[index].Relevance == nil {
			hits[index].Relevance = &port.ArticleSearchRelevance{
				Mode:  port.SearchModeKeyword,
				Index: ArticlesIndexUID,
			}
		}
	}
	return hits, nil
}

func classifyKnowledgeSearchError(err error) error {
	if err == nil || stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) || errors.KindOf(err) != errors.KindInternal {
		return err
	}
	return errors.WrapUnavailable("search.article_chunks", err)
}

func embeddingSearchFallbackEligible(mode port.SearchMode, err error) bool {
	if mode != port.SearchModeHybrid && mode != port.SearchModeSemantic {
		return false
	}
	if err == nil || stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return false
	}
	switch errors.Op(err) {
	case articleChunksEmbeddingOperation, articleChunksEmbeddingResolveOperation:
		// Only errors emitted by the embedding phase can trigger this fallback.
	default:
		return false
	}
	switch errors.AICodeOf(err) {
	case "", errors.AICodeDisabled, errors.AICodeProviderUnavailable, errors.AICodeCircuitOpen, errors.AICodeRateLimited:
		return true
	default:
		return false
	}
}

func articleSearchHitsFromKnowledge(hits []port.KnowledgeHit, mode port.SearchMode) []port.ArticleSearchHit {
	result := make([]port.ArticleSearchHit, 0, len(hits))
	seen := make(map[int]struct{}, len(hits))
	for _, hit := range hits {
		articleID, ok := articleIDFromKnowledgeHit(hit)
		if !ok {
			continue
		}
		if _, exists := seen[articleID]; exists {
			continue
		}
		seen[articleID] = struct{}{}
		result = append(result, port.ArticleSearchHit{
			ArticleSearch: port.ArticleSearch{
				Id:             articleID,
				ArticleTitle:   hit.Title,
				ArticleContent: hit.Text,
				IsDelete:       0,
				Status:         port.PublicArticleStatus,
			},
			HighlightedTitle:   hit.Highlights[ArticleChunkTitleField],
			HighlightedContent: hit.Highlights[ArticleChunkTextField],
			Source:             articleSearchSourceFromKnowledgeHit(articleID, hit),
			Relevance: &port.ArticleSearchRelevance{
				Score: hit.Score,
				Mode:  mode,
				Index: strings.TrimSpace(hit.Index),
			},
		})
	}
	return result
}

func articleSearchSourceFromKnowledgeHit(articleID int, hit port.KnowledgeHit) *port.ArticleSearchSource {
	source := &port.ArticleSearchSource{
		ArticleID:  articleID,
		Title:      hit.Title,
		URL:        strings.TrimSpace(hit.URL),
		ChunkID:    strings.TrimSpace(hit.ID),
		ChunkIndex: knowledgeMetadataInt(hit.Metadata, "chunkIndex"),
		Category:   strings.TrimSpace(hit.Metadata["category"]),
		Tags:       knowledgeMetadataTags(hit.Metadata["tags"]),
	}
	if publishedAtUnix, err := strconv.ParseInt(strings.TrimSpace(hit.Metadata["publishedAtUnix"]), 10, 64); err == nil && publishedAtUnix > 0 {
		source.PublishedAtUnix = publishedAtUnix
	}
	return source
}

func knowledgeMetadataInt(metadata map[string]string, key string) int {
	value, err := strconv.Atoi(strings.TrimSpace(metadata[key]))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func knowledgeMetadataTags(value string) []string {
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			tags = append(tags, part)
		}
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

func articleIDFromKnowledgeHit(hit port.KnowledgeHit) (int, bool) {
	if value := strings.TrimSpace(hit.Metadata["articleId"]); value != "" {
		if articleID, err := strconv.Atoi(value); err == nil && articleID > 0 {
			return articleID, true
		}
	}
	const prefix = "article-"
	const marker = "-chunk-"
	id := strings.TrimSpace(hit.ID)
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	id = strings.TrimPrefix(id, prefix)
	articleIDText, _, ok := strings.Cut(id, marker)
	if !ok {
		return 0, false
	}
	articleID, err := strconv.Atoi(articleIDText)
	return articleID, err == nil && articleID > 0
}

func (s *MeiliSearcher) Search(ctx context.Context, keywords string) (hits []port.ArticleSearchHit, err error) {
	started := time.Now()
	if s != nil && s.metrics != nil {
		defer func() {
			s.metrics.Observe(ctx, ArticlesIndexUID, port.SearchModeKeyword, time.Since(started), err)
		}()
	}
	if s == nil {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if s.baseURL != "" {
		return s.searchHTTP(ctx, keywords)
	}
	if s.client == nil {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	index := s.client.Index(ArticlesIndexUID)
	response, err := index.SearchWithContext(ctx, keywords, &meilisearch.SearchRequest{
		AttributesToRetrieve:  articleKeywordRetrievableFields(),
		AttributesToSearchOn:  articleKeywordSearchableFields(),
		Limit:                 ArticleSearchLimit,
		Offset:                0,
		AttributesToHighlight: articleKeywordSearchableFields(),
		CropLength:            ArticleCropLength,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	})
	if err != nil {
		return nil, errors.WrapUnavailable("search.articles", err)
	}
	if response == nil {
		return nil, errors.Unavailable("search.articles", stderrors.New("Meilisearch returned an empty search response"))
	}
	result := make([]port.ArticleSearchHit, 0, len(response.Hits))
	for _, hit := range response.Hits {
		var decoded struct {
			ID             int               `json:"id"`
			ArticleTitle   string            `json:"articleTitle"`
			ArticleContent string            `json:"articleContent"`
			IsDelete       int               `json:"isDelete"`
			Status         int               `json:"status"`
			Formatted      map[string]string `json:"_formatted"`
		}
		data, err := json.Marshal(hit)
		if err != nil {
			return nil, errors.Unavailable("search.decode", err)
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, errors.Unavailable("search.decode", err)
		}
		if !isPublicArticleHit(decoded.IsDelete, decoded.Status) {
			continue
		}
		if decoded.ID <= 0 {
			// A malformed public hit must never be turned into an article with a
			// synthetic or zero ID.
			continue
		}
		result = append(result, port.ArticleSearchHit{
			ArticleSearch: port.ArticleSearch{
				Id:             decoded.ID,
				ArticleTitle:   decoded.ArticleTitle,
				ArticleContent: decoded.ArticleContent,
				IsDelete:       decoded.IsDelete,
				Status:         decoded.Status,
			},
			HighlightedTitle:   decoded.Formatted["articleTitle"],
			HighlightedContent: decoded.Formatted["articleContent"],
		})
	}
	return result, nil
}

// searchHTTP deliberately uses a small explicit request payload instead of
// the newest SDK's SearchRequest. Recent SDKs always serialize newer fields
// (for example hybrid: null), which older but still deployed Meilisearch
// servers reject as unknown fields. The stable search fields below work with
// both the old and current API versions.
func (s *MeiliSearcher) searchHTTP(ctx context.Context, keywords string) ([]port.ArticleSearchHit, error) {
	payload := struct {
		Query                 string   `json:"q"`
		AttributesToRetrieve  []string `json:"attributesToRetrieve,omitempty"`
		Limit                 int      `json:"limit,omitempty"`
		Offset                int      `json:"offset,omitempty"`
		AttributesToHighlight []string `json:"attributesToHighlight,omitempty"`
		CropLength            int      `json:"cropLength,omitempty"`
		HighlightPreTag       string   `json:"highlightPreTag,omitempty"`
		HighlightPostTag      string   `json:"highlightPostTag,omitempty"`
	}{
		Query:                 keywords,
		AttributesToRetrieve:  articleKeywordRetrievableFields(),
		Limit:                 ArticleSearchLimit,
		AttributesToHighlight: articleKeywordSearchableFields(),
		CropLength:            ArticleCropLength,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Unavailable("search.encode", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/indexes/"+ArticlesIndexUID+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, errors.Unavailable("search.request", err)
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
		return nil, errors.WrapUnavailable("search.request", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, errors.WrapUnavailable("search.response", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("Meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	var decoded struct {
		Hits json.RawMessage `json:"hits"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, errors.Unavailable("search.decode", err)
	}
	if err := validateMeiliSearchHits(decoded.Hits); err != nil {
		return nil, errors.Unavailable("search.decode", err)
	}
	var hits []json.RawMessage
	if err := json.Unmarshal(decoded.Hits, &hits); err != nil {
		return nil, errors.Unavailable("search.decode", err)
	}
	return decodeSearchHits(hits)
}

func validateMeiliSearchHits(hits json.RawMessage) error {
	trimmed := bytes.TrimSpace(hits)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return stderrors.New("Meilisearch response does not contain a hits array")
	}
	if trimmed[0] != '[' {
		return stderrors.New("Meilisearch response hits must be an array")
	}
	return nil
}

func decodeSearchHits(hits []json.RawMessage) ([]port.ArticleSearchHit, error) {
	result := make([]port.ArticleSearchHit, 0, len(hits))
	for _, hit := range hits {
		var decoded struct {
			ID             int               `json:"id"`
			ArticleTitle   string            `json:"articleTitle"`
			ArticleContent string            `json:"articleContent"`
			IsDelete       int               `json:"isDelete"`
			Status         int               `json:"status"`
			Formatted      map[string]string `json:"_formatted"`
		}
		if err := json.Unmarshal(hit, &decoded); err != nil {
			return nil, errors.Unavailable("search.decode", err)
		}
		if !isPublicArticleHit(decoded.IsDelete, decoded.Status) {
			continue
		}
		if decoded.ID <= 0 {
			continue
		}
		result = append(result, port.ArticleSearchHit{
			ArticleSearch: port.ArticleSearch{
				Id:             decoded.ID,
				ArticleTitle:   decoded.ArticleTitle,
				ArticleContent: decoded.ArticleContent,
				IsDelete:       decoded.IsDelete,
				Status:         decoded.Status,
			},
			HighlightedTitle:   decoded.Formatted["articleTitle"],
			HighlightedContent: decoded.Formatted["articleContent"],
		})
	}
	return result, nil
}

var _ port.ArticleSearcher = (*MeiliSearcher)(nil)
var _ port.ArticleModeSearcher = (*MeiliSearcher)(nil)
var _ port.ArticleFilteredModeSearcher = (*MeiliSearcher)(nil)
