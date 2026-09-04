package port

import "context"

// ArticleIndexSource is the complete current source record needed to build a
// searchable article projection. It is separate from ArticleAdmin, whose
// intentionally small response must not be used for a full content rebuild.
type ArticleIndexSource struct {
	Article      TArticle
	CategoryName string
	Tags         []string
}

// ArticleIndexSourceRepository exposes a keyset-paginated public snapshot for
// explicit index backfills. afterArticleID is exclusive, so a caller can
// resume without offset drift when unrelated articles are inserted.
type ArticleIndexSourceRepository interface {
	ListPublicArticleIndexSources(ctx context.Context, afterArticleID, limit int) ([]ArticleIndexSource, error)
}
