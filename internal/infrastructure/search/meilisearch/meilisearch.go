package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// CheckHealth performs a read-only Meilisearch health request.
func (s *MeiliSearcher) CheckHealth(ctx context.Context) error {
	if s == nil || s.baseURL == "" {
		return fmt.Errorf("meilisearch health endpoint is not configured")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("meilisearch health returned status %d", response.StatusCode)
	}
	return nil
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
		ID               int               `json:"id"`
		UserID           int               `json:"userId"`
		ArticleCover     string            `json:"articleCover"`
		ArticleTitle     string            `json:"articleTitle"`
		ArticleContent   string            `json:"articleContent"`
		CategoryName     string            `json:"categoryName"`
		CreateTime       time.Time         `json:"createTime"`
		UpdateTime       time.Time         `json:"updateTime"`
		Author           port.PublicAuthor `json:"author"`
		IsDelete         int               `json:"isDelete"`
		Status           int               `json:"status"`
		ModerationStatus string            `json:"moderationStatus"`
		Formatted        struct {
			ArticleTitle   string `json:"articleTitle"`
			ArticleContent string `json:"articleContent"`
		} `json:"_formatted"`
	}
	if err := json.Unmarshal(hit, &decoded); err != nil {
		return port.ArticleSearchHit{}, errors.Unavailable("search.decode", err)
	}
	return port.ArticleSearchHit{
		ArticleSearch: port.ArticleSearch{
			Id: decoded.ID, UserId: decoded.UserID, ArticleCover: decoded.ArticleCover,
			ArticleTitle: decoded.ArticleTitle, ArticleContent: decoded.ArticleContent,
			CategoryName: decoded.CategoryName, CreateTime: decoded.CreateTime, UpdateTime: decoded.UpdateTime,
			Author: decoded.Author, IsDelete: decoded.IsDelete, Status: decoded.Status,
			ModerationStatus: decoded.ModerationStatus,
		},
		HighlightedTitle:   decoded.Formatted.ArticleTitle,
		HighlightedContent: decoded.Formatted.ArticleContent,
	}, nil
}

func (s *MeiliSearcher) Upsert(ctx context.Context, documents []port.ArticleSearch) error {
	if len(documents) == 0 {
		return nil
	}
	if s == nil {
		return errors.Unavailable("search.index", fmt.Errorf("search client is not configured"))
	}
	if s.baseURL != "" {
		if err := s.ensureHTTPIndex(ctx); err != nil {
			return err
		}
		return s.writeHTTPDocuments(ctx, http.MethodPost, "/indexes/articles/documents", documents)
	}
	if s.client == nil {
		return errors.Unavailable("search.index", fmt.Errorf("search client is not configured"))
	}
	if _, err := s.client.Index("articles").AddDocumentsWithContext(ctx, documents, nil); err != nil {
		return errors.Unavailable("search.index", err)
	}
	return nil
}

func (s *MeiliSearcher) Delete(ctx context.Context, articleIDs []int) error {
	if len(articleIDs) == 0 {
		return nil
	}
	if s == nil {
		return errors.Unavailable("search.index", fmt.Errorf("search client is not configured"))
	}
	identifiers := make([]string, 0, len(articleIDs))
	for _, id := range articleIDs {
		if id > 0 {
			identifiers = append(identifiers, fmt.Sprint(id))
		}
	}
	if len(identifiers) == 0 {
		return nil
	}
	if s.baseURL != "" {
		if err := s.ensureHTTPIndex(ctx); err != nil {
			return err
		}
		return s.writeHTTPDocuments(ctx, http.MethodPost, "/indexes/articles/documents/delete-batch", identifiers)
	}
	if s.client == nil {
		return errors.Unavailable("search.index", fmt.Errorf("search client is not configured"))
	}
	if _, err := s.client.Index("articles").DeleteDocumentsWithContext(ctx, identifiers, nil); err != nil {
		return errors.Unavailable("search.index", err)
	}
	return nil
}

// Reconcile makes the public document set match the database snapshot. It
// deletes stale identifiers first and then upserts the authoritative rows.
func (s *MeiliSearcher) Reconcile(ctx context.Context, documents []port.ArticleSearch) error {
	if s == nil {
		return errors.Unavailable("search.reconcile", fmt.Errorf("search client is not configured"))
	}
	if s.baseURL != "" {
		if err := s.ensureHTTPIndex(ctx); err != nil {
			return err
		}
		if err := s.configureHTTPIndex(ctx); err != nil {
			return err
		}
		currentIDs, err := s.listHTTPDocumentIDs(ctx)
		if err != nil {
			return err
		}
		wanted := make(map[int]struct{}, len(documents))
		for _, document := range documents {
			if document.Id > 0 {
				wanted[document.Id] = struct{}{}
			}
		}
		stale := make([]int, 0)
		for id := range currentIDs {
			if _, ok := wanted[id]; !ok {
				stale = append(stale, id)
			}
		}
		if err := s.Delete(ctx, stale); err != nil {
			return err
		}
		return s.Upsert(ctx, documents)
	}
	if s.client == nil {
		return errors.Unavailable("search.reconcile", fmt.Errorf("search client is not configured"))
	}
	currentIDs, err := s.listSDKDocumentIDs(ctx)
	if err != nil {
		return err
	}
	wanted := make(map[int]struct{}, len(documents))
	for _, document := range documents {
		if document.Id > 0 {
			wanted[document.Id] = struct{}{}
		}
	}
	stale := make([]int, 0)
	for id := range currentIDs {
		if _, ok := wanted[id]; !ok {
			stale = append(stale, id)
		}
	}
	if err := s.Delete(ctx, stale); err != nil {
		return err
	}
	return s.Upsert(ctx, documents)
}

func (s *MeiliSearcher) ensureHTTPIndex(ctx context.Context) error {
	payload := map[string]string{"uid": "articles", "primaryKey": "id"}
	response, err := s.doHTTP(ctx, http.MethodPost, "/indexes", payload)
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusConflict {
		response.Body.Close()
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return errors.Unavailable("search.index", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body))))
	}
	return nil
}

func (s *MeiliSearcher) configureHTTPIndex(ctx context.Context) error {
	settings := []struct {
		path    string
		payload []string
	}{
		{"/indexes/articles/settings/searchable-attributes", []string{"articleTitle", "articleContent"}},
		{"/indexes/articles/settings/filterable-attributes", []string{"isDelete", "status", "moderationStatus"}},
		{"/indexes/articles/settings/sortable-attributes", []string{"createTime"}},
	}
	for _, setting := range settings {
		response, err := s.doHTTP(ctx, http.MethodPut, setting.path, setting.payload)
		if err != nil {
			return err
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			response.Body.Close()
			return errors.Unavailable("search.settings", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body))))
		}
		response.Body.Close()
	}
	return nil
}

func (s *MeiliSearcher) writeHTTPDocuments(ctx context.Context, method, path string, payload any) error {
	response, err := s.doHTTP(ctx, method, path, payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return errors.Unavailable("search.index", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body))))
	}
	return nil
}

func (s *MeiliSearcher) doHTTP(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, errors.Unavailable("search.encode", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, body)
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
	return response, nil
}

func (s *MeiliSearcher) listHTTPDocumentIDs(ctx context.Context) (map[int]struct{}, error) {
	ids := make(map[int]struct{})
	const pageSize = 1000
	for offset := 0; ; offset += pageSize {
		query := url.Values{}
		query.Set("limit", fmt.Sprint(pageSize))
		query.Set("offset", fmt.Sprint(offset))
		query.Set("fields", "id")
		response, err := s.doHTTP(ctx, http.MethodGet, "/indexes/articles/documents?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			response.Body.Close()
			return nil, errors.Unavailable("search.documents", fmt.Errorf("meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body))))
		}
		var decoded struct {
			Results []struct {
				ID int `json:"id"`
			} `json:"results"`
			Total int `json:"total"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&decoded)
		response.Body.Close()
		if decodeErr != nil {
			return nil, errors.Unavailable("search.decode", decodeErr)
		}
		for _, row := range decoded.Results {
			if row.ID > 0 {
				ids[row.ID] = struct{}{}
			}
		}
		if len(decoded.Results) < pageSize || (decoded.Total > 0 && offset+len(decoded.Results) >= decoded.Total) {
			return ids, nil
		}
	}
}

func (s *MeiliSearcher) listSDKDocumentIDs(ctx context.Context) (map[int]struct{}, error) {
	ids := make(map[int]struct{})
	const pageSize = 1000
	for offset := int64(0); ; offset += pageSize {
		var result meilisearch.DocumentsResult
		if err := s.client.Index("articles").GetDocumentsWithContext(ctx, &meilisearch.DocumentsQuery{
			Limit: pageSize, Offset: offset, Fields: []string{"id"},
		}, &result); err != nil {
			return nil, errors.Unavailable("search.documents", err)
		}
		for _, hit := range result.Results {
			var row struct {
				ID int `json:"id"`
			}
			if err := hit.DecodeInto(&row); err != nil {
				return nil, errors.Unavailable("search.decode", err)
			}
			if row.ID > 0 {
				ids[row.ID] = struct{}{}
			}
		}
		if len(result.Results) < pageSize || offset+int64(len(result.Results)) >= result.Total {
			return ids, nil
		}
	}
}

var _ port.ArticleSearcher = (*MeiliSearcher)(nil)
var _ port.ArticleSearchIndexer = (*MeiliSearcher)(nil)
