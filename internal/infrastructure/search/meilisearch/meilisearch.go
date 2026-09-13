package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"io"
	"net/http"
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

func (s *MeiliSearcher) Search(ctx context.Context, keywords string) ([]port.ArticleSearchHit, error) {
	if s == nil {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	if s.baseURL != "" {
		return s.searchHTTP(ctx, keywords)
	}
	if s.client == nil {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("search client is not configured"))
	}
	index := s.client.Index("articles")
	response, err := index.SearchWithContext(ctx, keywords, &meilisearch.SearchRequest{
		AttributesToRetrieve:  []string{"*"},
		Limit:                 1000,
		Offset:                0,
		AttributesToHighlight: []string{"articleTitle", "articleContent"},
		CropLength:            50,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	})
	if err != nil {
		return nil, errors.Unavailable("search.articles", err)
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
		AttributesToRetrieve:  []string{"*"},
		Limit:                 1000,
		AttributesToHighlight: []string{"articleTitle", "articleContent"},
		CropLength:            50,
		HighlightPreTag:       preTag,
		HighlightPostTag:      postTag,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Unavailable("search.encode", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/indexes/articles/search", bytes.NewReader(body))
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
		return nil, errors.Unavailable("search.request", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, errors.Unavailable("search.response", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.Unavailable("search.articles", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	var decoded struct {
		Hits []json.RawMessage `json:"hits"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, errors.Unavailable("search.decode", err)
	}
	return decodeSearchHits(decoded.Hits)
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
