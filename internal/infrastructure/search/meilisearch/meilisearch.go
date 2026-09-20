package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/meilisearch/meilisearch-go"
)

const (
	preTag       = "<mark>"
	postTag      = "</mark>"
	publicFilter = "isDelete = 0 AND status = 1 AND moderationStatus != 'hidden'"
)

type MeiliSearcher struct {
	client     meilisearch.ServiceManager
	baseURL    string
	apiKey     string
	httpClient *http.Client
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

func (s *MeiliSearcher) Search(ctx context.Context, keywords string, offset, limit int) (port.ArticleSearchPage, error) {
	if s == nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}
	if s.baseURL != "" {
		return s.searchHTTP(ctx, keywords, offset, limit)
	}
	if s.client == nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	index := s.client.Index("articles")
	response, err := index.SearchWithContext(ctx, keywords, &meilisearch.SearchRequest{
		AttributesToRetrieve:  []string{"*"},
		Limit:                 int64(limit),
		Offset:                int64(offset),
		Filter:                publicFilter,
		AttributesToHighlight: []string{"articleTitle", "articleContent"},
		CropLength:            50,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	})
	if err != nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.articles", err)
	}
	hits, err := decodeSearchHitsFromSDK(response.Hits)
	if err != nil {
		return port.ArticleSearchPage{}, err
	}
	return port.ArticleSearchPage{Hits: hits, Total: response.EstimatedTotalHits}, nil
}

// searchHTTP deliberately uses a small explicit request payload instead of
// the newest SDK's SearchRequest. Recent SDKs always serialize newer fields
// (for example hybrid: null), which older but still deployed Meilisearch
// servers reject as unknown fields. The stable search fields below work with
// both the old and current API versions.
func (s *MeiliSearcher) searchHTTP(ctx context.Context, keywords string, offset, limit int) (port.ArticleSearchPage, error) {
	payload := struct {
		Query                 string   `json:"q"`
		AttributesToRetrieve  []string `json:"attributesToRetrieve,omitempty"`
		Limit                 int      `json:"limit,omitempty"`
		Offset                int      `json:"offset,omitempty"`
		Filter                string   `json:"filter,omitempty"`
		AttributesToHighlight []string `json:"attributesToHighlight,omitempty"`
		CropLength            int      `json:"cropLength,omitempty"`
		HighlightPreTag       string   `json:"highlightPreTag,omitempty"`
		HighlightPostTag      string   `json:"highlightPostTag,omitempty"`
	}{
		Query:                 keywords,
		AttributesToRetrieve:  []string{"*"},
		Limit:                 limit,
		Offset:                offset,
		Filter:                publicFilter,
		AttributesToHighlight: []string{"articleTitle", "articleContent"},
		CropLength:            50,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.encode", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/indexes/articles/search", bytes.NewReader(body))
	if err != nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.request", err)
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
		return port.ArticleSearchPage{}, errors.Unavailable("search.request", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.response", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return port.ArticleSearchPage{}, errors.Unavailable("search.articles", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	var decoded struct {
		Hits               []json.RawMessage `json:"hits"`
		EstimatedTotalHits int64             `json:"estimatedTotalHits"`
		TotalHits          int64             `json:"totalHits"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return port.ArticleSearchPage{}, errors.Unavailable("search.decode", err)
	}
	total := decoded.EstimatedTotalHits
	if total == 0 {
		total = decoded.TotalHits
	}
	hits, err := decodeSearchHits(decoded.Hits)
	if err != nil {
		return port.ArticleSearchPage{}, err
	}
	return port.ArticleSearchPage{Hits: hits, Total: total}, nil
}

func decodeSearchHitsFromSDK(hits meilisearch.Hits) ([]port.ArticleSearchHit, error) {
	result := make([]port.ArticleSearchHit, 0, len(hits))
	for _, hit := range hits {
		data, err := json.Marshal(hit)
		if err != nil {
			return nil, errors.Unavailable("search.decode", err)
		}
		decoded, err := decodeSearchHit(data)
		if err != nil {
			return nil, err
		}
		result = append(result, decoded)
	}
	return result, nil
}

func decodeSearchHits(hits []json.RawMessage) ([]port.ArticleSearchHit, error) {
	result := make([]port.ArticleSearchHit, 0, len(hits))
	for _, hit := range hits {
		decoded, err := decodeSearchHit(hit)
		if err != nil {
			return nil, err
		}
		result = append(result, decoded)
	}
	return result, nil
}

func decodeSearchHit(hit json.RawMessage) (port.ArticleSearchHit, error) {
	var decoded struct {
		ID             int               `json:"id"`
		ArticleTitle   string            `json:"articleTitle"`
		ArticleContent string            `json:"articleContent"`
		IsDelete       int               `json:"isDelete"`
		Status         int               `json:"status"`
		Formatted      map[string]string `json:"_formatted"`
	}
	if err := json.Unmarshal(hit, &decoded); err != nil {
		return port.ArticleSearchHit{}, errors.Unavailable("search.decode", err)
	}
	return port.ArticleSearchHit{
		ArticleSearch: port.ArticleSearch{
			Id:             decoded.ID,
			ArticleTitle:   decoded.ArticleTitle,
			ArticleContent: decoded.ArticleContent,
			IsDelete:       decoded.IsDelete,
			Status:         decoded.Status,
		},
		HighlightedTitle:   decoded.Formatted["articleTitle"],
		HighlightedContent: decoded.Formatted["articleContent"],
	}, nil
}

var _ port.ArticleSearcher = (*MeiliSearcher)(nil)
