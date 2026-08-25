package port

import "benetnasch/app/facade/model"

// ArticleRepository is the application-facing contract for article reads.
// The concrete xorm adapter lives under infra/persistence/repository; keeping
// this contract in domain prevents the application layer from depending on
// that adapter while the remaining repositories are migrated incrementally.
type ArticleRepository interface {
	ListTopAndFeaturedArticles() []*model.ArticleCardDTO
	ListArticles(current, size int) []*model.ArticleCardDTO
	GetArticlesByCategoryId(current, size, categoryId int) []*model.ArticleCardDTO
	GetArticleById(articleId int) model.ArticleDTO
	GetPreArticleById(articleId int) model.ArticleCardDTO
	GetNextArticleById(articleId int) model.ArticleCardDTO
	GetFirstArticle() model.ArticleCardDTO
	GetLastArticle() model.ArticleCardDTO
	ListArticlesByTagId(current, size, tagId int) []*model.ArticleCardDTO
	ListArchives(current, size int) []model.ArticleCardDTO
	CountArticleAdmins(vo *model.ConditionVO) int
	ListArticlesAdmin(current, size int, vo *model.ConditionVO) []*model.ArticleAdminDTO
	ListArticleStatistics() []model.ArticleStatisticsDTO
}
