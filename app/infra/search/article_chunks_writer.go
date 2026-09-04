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

// MeiliArticleChunkIndex writes only the versioned article_chunks projection.
// The adapter uses the stable HTTP document API so it remains compatible with
// the Meilisearch server already used by this project; SDK-specific index
// provisioning remains an explicit operation in article_chunks_index.go.
type MeiliArticleChunkIndex struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	spec       ArticleChunksIndexSpec
}

type meiliArticleChunkDocument struct {
	ID                 string               `json:"id"`
	ArticleID          int                  `json:"articleId"`
	ArticleTitle       string               `json:"articleTitle"`
	Text               string               `json:"text"`
	Category           string               `json:"category"`
	Tags               []string             `json:"tags"`
	ArticleURL         string               `json:"articleUrl"`
	PublishedAt        time.Time            `json:"publishedAt"`
	PublishedAtUnix    int64                `json:"publishedAtUnix"`
	ChunkIndex         int                  `json:"chunkIndex"`
	Status             int                  `json:"status"`
	IsDelete           int                  `json:"isDelete"`
	EmbeddingModel     string               `json:"embeddingModel"`
	EmbeddingVersion   string               `json:"embeddingVersion"`
	EmbeddingDimension int                  `json:"embeddingDimension"`
	Vectors            map[string][]float32 `json:"_vectors"`
}

var _ port.ArticleChunkIndex = (*MeiliArticleChunkIndex)(nil)

func NewMeiliArticleChunkIndex(conf *config.MeiliSearch, spec ArticleChunksIndexSpec) (*MeiliArticleChunkIndex, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if conf == nil {
		return &MeiliArticleChunkIndex{spec: spec}, nil
	}
	return &MeiliArticleChunkIndex{
		baseURL:    strings.TrimRight(strings.TrimSpace(conf.URL), "/"),
		apiKey:     strings.TrimSpace(conf.ApiKey),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		spec:       spec,
	}, nil
}

// NewMeiliArticleChunkIndexWithHTTPClient is useful for bounded tests and for
// callers that already own an HTTP transport. It does not contact Meili.
func NewMeiliArticleChunkIndexWithHTTPClient(baseURL, apiKey string, client *http.Client, spec ArticleChunksIndexSpec) (*MeiliArticleChunkIndex, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &MeiliArticleChunkIndex{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: client,
		spec:       spec,
	}, nil
}

func (i *MeiliArticleChunkIndex) UpsertChunks(ctx context.Context, chunks []port.IndexedArticleChunk) error {
	if i == nil {
		return apperrors.Unavailable("search.article_chunks.upsert", nil)
	}
	if err := i.spec.Validate(); err != nil {
		return err
	}
	if len(chunks) == 0 {
		return nil
	}
	documents := make([]meiliArticleChunkDocument, 0, len(chunks))
	for index, chunk := range chunks {
		document, err := NormalizeArticleChunkDocument(chunk.Document)
		if err != nil {
			return apperrors.Wrap(apperrors.KindValidation, fmt.Sprintf("search.article_chunks.upsert.%d", index), err)
		}
		if err := i.spec.ValidateVector(chunk.Embedding); err != nil {
			return err
		}
		model := strings.TrimSpace(chunk.EmbeddingModel)
		version := strings.TrimSpace(chunk.EmbeddingVersion)
		if model != i.spec.Model || version != i.spec.ModelVersion {
			return apperrors.Invalid("search.article_chunks.embedding_contract", "embedding model or version does not match the index specification")
		}
		dimension := chunk.EmbeddingDimension
		if dimension == 0 {
			dimension = len(chunk.Embedding)
		}
		if dimension != i.spec.Dimension {
			return apperrors.Invalid("search.article_chunks.embedding_contract", "embedding dimension does not match the index specification")
		}
		documents = append(documents, meiliArticleChunkDocument{
			ID:                 document.ID,
			ArticleID:          document.ArticleID,
			ArticleTitle:       document.ArticleTitle,
			Text:               document.Text,
			Category:           document.Category,
			Tags:               append([]string(nil), document.Tags...),
			ArticleURL:         document.ArticleURL,
			PublishedAt:        document.PublishedAt,
			PublishedAtUnix:    document.PublishedAtUnix,
			ChunkIndex:         document.ChunkIndex,
			Status:             document.Status,
			IsDelete:           document.IsDelete,
			EmbeddingModel:     model,
			EmbeddingVersion:   version,
			EmbeddingDimension: dimension,
			Vectors:            map[string][]float32{"default": append([]float32(nil), chunk.Embedding...)},
		})
	}
	body, err := json.Marshal(documents)
	if err != nil {
		return apperrors.Unavailable("search.article_chunks.encode", err)
	}
	endpoint := i.documentsEndpoint()
	query := url.Values{}
	query.Set("primaryKey", ArticleChunksIndexPrimaryKey)
	return i.doJSON(ctx, http.MethodPost, endpoint+"?"+query.Encode(), body, "search.article_chunks.upsert")
}

func (i *MeiliArticleChunkIndex) DeleteArticle(ctx context.Context, articleID int) error {
	if i == nil {
		return apperrors.Unavailable("search.article_chunks.delete", nil)
	}
	if err := i.spec.Validate(); err != nil {
		return err
	}
	if articleID <= 0 {
		return apperrors.Invalid("search.article_chunks.delete", "article id must be positive")
	}
	body, err := json.Marshal(map[string]string{
		"filter": ArticleChunkArticleIDField + " = " + strconv.Itoa(articleID),
	})
	if err != nil {
		return apperrors.Unavailable("search.article_chunks.delete.encode", err)
	}
	return i.doJSON(ctx, http.MethodPost, i.documentsEndpoint()+"/delete", body, "search.article_chunks.delete")
}

func (i *MeiliArticleChunkIndex) documentsEndpoint() string {
	return i.baseURL + "/indexes/" + url.PathEscape(i.spec.UID) + "/documents"
}

func (i *MeiliArticleChunkIndex) doJSON(ctx context.Context, method, endpoint string, body []byte, operation string) error {
	if strings.TrimSpace(i.baseURL) == "" {
		return apperrors.Unavailable(operation, fmt.Errorf("Meilisearch URL is not configured"))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return apperrors.Unavailable(operation+".request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if i.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+i.apiKey)
	}
	client := i.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return apperrors.Unavailable(operation+".request", err)
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		return apperrors.Unavailable(operation+".response", readErr)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return apperrors.Unavailable(operation, fmt.Errorf("Meilisearch returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	return nil
}
