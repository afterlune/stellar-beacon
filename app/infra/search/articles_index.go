package search

import "benetnasch/app/domain/port"

// The articles index is the legacy keyword-search contract. Keep its UID and
// field names stable while later milestones add versioned chunk indexes.
const (
	ArticlesIndexUID        = "articles"
	ArticlesIndexPrimaryKey = "id"
	ArticleIDField          = "id"
	ArticleTitleField       = "articleTitle"
	ArticleContentField     = "articleContent"
	ArticleIsDeleteField    = "isDelete"
	ArticleStatusField      = "status"
	ArticleSearchLimit      = 1000
	ArticleCropLength       = 50
)

var articleKeywordSearchableAttributes = []string{
	ArticleTitleField,
	ArticleContentField,
}

var articleKeywordRetrievableAttributes = []string{
	ArticleIDField,
	ArticleTitleField,
	ArticleContentField,
	ArticleIsDeleteField,
	ArticleStatusField,
}

func articleKeywordSearchableFields() []string {
	return append([]string(nil), articleKeywordSearchableAttributes...)
}

func articleKeywordRetrievableFields() []string {
	return append([]string(nil), articleKeywordRetrievableAttributes...)
}

// isPublicArticleHit is deliberately fail-closed. The legacy index is
// populated from rows where is_delete = 0 and status = 1; filtering again at
// the adapter boundary prevents stale or manually inserted private documents
// from reaching the public search API.
func isPublicArticleHit(isDelete, status int) bool {
	return port.IsPublicArticle(status, isDelete)
}
