package port

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// ArticleRepository is the application-facing contract for article reads.
// It owns domain read models and propagates persistence failures instead of
// turning them into empty result sets.
type ArticleRepository interface {
	ListTopAndFeaturedArticles(ctx context.Context) ([]*ArticleCard, error)
	ListArticles(ctx context.Context, current, size int) ([]*ArticleCard, int, error)
	GetArticlesByCategoryID(ctx context.Context, current, size, categoryID int) ([]*ArticleCard, int, error)
	GetArticleByID(ctx context.Context, articleID int) (Article, error)
	GetPreArticleByID(ctx context.Context, articleID int) (ArticleCard, error)
	GetNextArticleByID(ctx context.Context, articleID int) (ArticleCard, error)
	GetFirstArticle(ctx context.Context) (ArticleCard, error)
	GetLastArticle(ctx context.Context) (ArticleCard, error)
	ListArticlesByTagID(ctx context.Context, current, size, tagID int) ([]*ArticleCard, int, error)
	ListArchives(ctx context.Context, current, size int) ([]ArticleCard, int, error)
	CountArticleAdmins(ctx context.Context, filter ArticleFilter) (int, error)
	ListArticlesAdmin(ctx context.Context, filter ArticleFilter) ([]*ArticleAdmin, error)
	ListArticleStatistics(ctx context.Context) ([]ArticleStatistics, error)
	GetArticleRecord(ctx context.Context, articleID int) (entity.TArticle, error)
	SaveOrUpdate(ctx context.Context, article entity.TArticle, categoryName string, tagNames []string) (entity.TArticle, error)
	UpdateTopAndFeatured(ctx context.Context, articleID, isTop, isFeatured int) (entity.TArticle, error)
	UpdateDelete(ctx context.Context, ids []int, isDelete int) error
	Delete(ctx context.Context, ids []int) error
	GetAdminArticle(ctx context.Context, articleID int) (entity.TArticle, string, []string, error)
	Export(ctx context.Context, ids []int) ([]entity.TArticle, error)
}
