package search

import (
	"fmt"
	"strconv"
	"strings"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
)

const (
	ArticleChunkIDField                 = "id"
	ArticleChunkArticleIDField          = "articleId"
	ArticleChunkTitleField              = "articleTitle"
	ArticleChunkTextField               = "text"
	ArticleChunkCategoryField           = "category"
	ArticleChunkTagsField               = "tags"
	ArticleChunkURLField                = "articleUrl"
	ArticleChunkPublishedAtField        = "publishedAt"
	ArticleChunkPublishedAtUnixField    = "publishedAtUnix"
	ArticleChunkIndexField              = "chunkIndex"
	ArticleChunkStatusField             = "status"
	ArticleChunkIsDeleteField           = "isDelete"
	ArticleChunkEmbeddingModelField     = "embeddingModel"
	ArticleChunkEmbeddingVersionField   = "embeddingVersion"
	ArticleChunkEmbeddingDimensionField = "embeddingDimension"
)

var articleChunkSchemaFields = []string{
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

func ArticleChunkSchemaFields() []string {
	return append([]string(nil), articleChunkSchemaFields...)
}

// ArticleChunkID is deterministic across retries and re-indexes. The worker
// can therefore replace a chunk idempotently without generating duplicates.
func ArticleChunkID(articleID, chunkIndex int) (string, error) {
	if articleID <= 0 {
		return "", apperrors.Invalid("search.article_chunk.id", "article id must be positive")
	}
	if chunkIndex < 0 {
		return "", apperrors.Invalid("search.article_chunk.id", "chunk index must not be negative")
	}
	return "article-" + strconv.Itoa(articleID) + "-chunk-" + strconv.Itoa(chunkIndex), nil
}

// NormalizeArticleChunkDocument validates the schema and returns a copy with
// stable whitespace, first-seen tag de-duplication and UTC timestamps. It does not clean
// Markdown or alter the body; that is intentionally M2-04's responsibility.
func NormalizeArticleChunkDocument(document port.ArticleChunkDocument) (port.ArticleChunkDocument, error) {
	document.ID = strings.TrimSpace(document.ID)
	document.ArticleTitle = strings.TrimSpace(document.ArticleTitle)
	document.Text = strings.TrimSpace(document.Text)
	document.Category = strings.TrimSpace(document.Category)
	document.ArticleURL = strings.TrimSpace(document.ArticleURL)
	document.PublishedAt = document.PublishedAt.UTC()
	document.PublishedAtUnix = document.PublishedAt.Unix()

	if document.ID == "" {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "chunk id is required")
	}
	if document.ArticleID <= 0 {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "article id must be positive")
	}
	if document.ArticleTitle == "" {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "article title is required")
	}
	if document.Text == "" {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "chunk text is required")
	}
	if document.ChunkIndex < 0 {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "chunk index must not be negative")
	}
	if document.PublishedAt.IsZero() {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "published time is required")
	}
	if !document.IsPublic() {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "only public articles may be indexed")
	}

	tags := make([]string, 0, len(document.Tags))
	seen := make(map[string]struct{}, len(document.Tags))
	for _, tag := range document.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", "tags must not contain empty values")
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	document.Tags = tags

	expectedID, err := ArticleChunkID(document.ArticleID, document.ChunkIndex)
	if err != nil {
		return port.ArticleChunkDocument{}, err
	}
	if document.ID != expectedID {
		return port.ArticleChunkDocument{}, apperrors.Invalid("search.article_chunk.document", fmt.Sprintf("chunk id %q does not match article and chunk index", document.ID))
	}
	return document, nil
}
