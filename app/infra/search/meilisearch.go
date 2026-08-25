package search

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/config"
	"context"
	"encoding/json"
	"fmt"

	"github.com/meilisearch/meilisearch-go"
)

const (
	preTag  = "<mark>"
	postTag = "</mark>"
)

type MeiliSearcher struct {
	client meilisearch.ServiceManager
}

func NewMeiliSearcher(conf *config.MeiliSearch) *MeiliSearcher {
	return &MeiliSearcher{client: meilisearch.New(conf.URL, meilisearch.WithAPIKey(conf.ApiKey))}
}

func NewMeiliSearcherWithClient(client meilisearch.ServiceManager) *MeiliSearcher {
	return &MeiliSearcher{client: client}
}

func (s *MeiliSearcher) Search(ctx context.Context, keywords string) ([]port.ArticleSearchHit, error) {
	if s == nil || s.client == nil {
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

var _ port.ArticleSearcher = (*MeiliSearcher)(nil)
