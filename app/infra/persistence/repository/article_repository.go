package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"container/list"
)

type ArticleRepo = port.ArticleRepository

type MyArticleRepo struct{}

func (a *MyArticleRepo) attachTags(articleID int, tags *[]entity.TTag) {
	if err := ormInit.GetEngine().SQL(pgsql.ArticleTags, articleID).Find(tags); err != nil {
		zlog.Error("load article tags: " + err.Error())
	}
}

func setArticleTags(tags []entity.TTag) interface{} {
	if len(tags) == 0 {
		return list.New()
	}
	return tags
}

func (a *MyArticleRepo) ListTopAndFeaturedArticles() []*model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	if err := engine.SQL(pgsql.ListTopAndFeaturedArticles).Find(&articles); err != nil {
		zlog.Error("list top articles: " + err.Error())
	}
	for _, article := range articles {
		var tags []entity.TTag
		a.attachTags(article.Id, &tags)
		article.Tags = setArticleTags(tags)
	}
	return articles
}

func (a *MyArticleRepo) ListArticles(current, size int) []*model.ArticleCardDTO {
	limit, offset := pgsql.Page(current, size)
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	if err := engine.SQL(pgsql.ListArticles, limit, offset).Find(&articles); err != nil {
		zlog.Error("list articles: " + err.Error())
	}
	for _, article := range articles {
		var tags []entity.TTag
		a.attachTags(article.Id, &tags)
		article.Tags = setArticleTags(tags)
	}
	return articles
}

func (a *MyArticleRepo) GetArticlesByCategoryId(current, size int, categoryId int) []*model.ArticleCardDTO {
	limit, offset := pgsql.Page(current, size)
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	if err := engine.SQL(pgsql.GetArticlesByCategoryId, categoryId, limit, offset).Find(&articles); err != nil {
		zlog.Error("list articles by category: " + err.Error())
	}
	for _, article := range articles {
		var tags []entity.TTag
		a.attachTags(article.Id, &tags)
		article.Tags = setArticleTags(tags)
	}
	return articles
}

func (a *MyArticleRepo) GetArticleById(articleID int) model.ArticleDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleDTO
	if _, err := engine.SQL(pgsql.GetArticleById, articleID).Get(&article); err != nil {
		zlog.Error("get article: " + err.Error())
	}
	var tags []entity.TTag
	a.attachTags(article.Id, &tags)
	article.Tags = setArticleTags(tags)
	return article
}

func (a *MyArticleRepo) cardWithTags(article model.ArticleCardDTO) model.ArticleCardDTO {
	var tags []entity.TTag
	a.attachTags(article.Id, &tags)
	article.Tags = setArticleTags(tags)
	return article
}

func (a *MyArticleRepo) GetPreArticleById(articleID int) model.ArticleCardDTO {
	var article model.ArticleCardDTO
	get, err := ormInit.GetEngine().SQL(pgsql.GetPreArticleById, articleID).Get(&article)
	if err != nil {
		zlog.Error("get previous article: " + err.Error())
	}
	if !get {
		return model.ArticleCardDTO{}
	}
	return a.cardWithTags(article)
}

func (a *MyArticleRepo) GetNextArticleById(articleID int) model.ArticleCardDTO {
	var article model.ArticleCardDTO
	get, err := ormInit.GetEngine().SQL(pgsql.GetNextArticleById, articleID).Get(&article)
	if err != nil {
		zlog.Error("get next article: " + err.Error())
	}
	if !get {
		return model.ArticleCardDTO{}
	}
	return a.cardWithTags(article)
}

func (a *MyArticleRepo) firstOrLast(query string) model.ArticleCardDTO {
	var article model.ArticleCardDTO
	if _, err := ormInit.GetEngine().SQL(query).Get(&article); err != nil {
		zlog.Error("get article navigation: " + err.Error())
	}
	return a.cardWithTags(article)
}

func (a *MyArticleRepo) GetFirstArticle() model.ArticleCardDTO {
	return a.firstOrLast(pgsql.GetFirstArticle)
}

func (a *MyArticleRepo) GetLastArticle() model.ArticleCardDTO {
	return a.firstOrLast(pgsql.GetLastArticle)
}

func (a *MyArticleRepo) ListArticlesByTagId(current, size int, tagID int) []*model.ArticleCardDTO {
	limit, offset := pgsql.Page(current, size)
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	if err := engine.SQL(pgsql.ListArticlesByTagId, tagID, limit, offset).Find(&articles); err != nil {
		zlog.Error("list articles by tag: " + err.Error())
	}
	for _, article := range articles {
		var tags []entity.TTag
		a.attachTags(article.Id, &tags)
		article.Tags = setArticleTags(tags)
	}
	return articles
}

func (a *MyArticleRepo) ListArchives(current, size int) []model.ArticleCardDTO {
	limit, offset := pgsql.Page(current, size)
	var articles []model.ArticleCardDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListArchives, limit, offset).Find(&articles); err != nil {
		zlog.Error("list article archives: " + err.Error())
	}
	return articles
}

func articleAdminFilters(vo *model.ConditionVO) (string, []interface{}) {
	query := " WHERE a.is_delete = ?"
	args := []interface{}{vo.IsDelete}
	if vo.Keywords != "" {
		query += " AND a.article_title LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	if vo.Status != 0 {
		query += " AND a.status = ?"
		args = append(args, vo.Status)
	}
	if vo.CategoryId != 0 {
		query += " AND a.category_id = ?"
		args = append(args, vo.CategoryId)
	}
	if vo.Type != 0 {
		query += " AND a.type = ?"
		args = append(args, vo.Type)
	}
	if vo.TagId != 0 {
		query += " AND a.id IN (SELECT article_id FROM t_article_tag WHERE tag_id = ?)"
		args = append(args, vo.TagId)
	}
	return query, args
}

func (a *MyArticleRepo) CountArticleAdmins(vo *model.ConditionVO) (count int) {
	filters, args := articleAdminFilters(vo)
	query := "SELECT count(DISTINCT a.id) FROM t_article a LEFT JOIN t_article_tag at ON a.id = at.article_id" + filters
	if _, err := ormInit.GetEngine().SQL(query, args...).Get(&count); err != nil {
		zlog.Error("count admin articles: " + err.Error())
	}
	return count
}

func (a *MyArticleRepo) ListArticlesAdmin(current, size int, vo *model.ConditionVO) []*model.ArticleAdminDTO {
	limit, offset := pgsql.Page(current, size)
	filters, args := articleAdminFilters(vo)
	query := "SELECT a.id, a.article_cover, a.article_title, a.is_top, a.is_featured, a.is_delete, a.status, a.type, a.create_time, c.category_name FROM (SELECT id, article_cover, article_title, is_top, is_featured, is_delete, status, type, create_time, category_id FROM t_article a" + filters + " ORDER BY is_top DESC, is_featured DESC, id DESC LIMIT ? OFFSET ?) a LEFT JOIN t_category c ON a.category_id = c.id ORDER BY is_top DESC, is_featured DESC, a.id DESC"
	args = append(args, limit, offset)
	var articles []*model.ArticleAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&articles); err != nil {
		zlog.Error("list admin articles: " + err.Error())
	}
	for _, article := range articles {
		var tags []model.TagDTO
		if err := ormInit.GetEngine().SQL(pgsql.ArticlesAdminTags, article.Id).Find(&tags); err != nil {
			zlog.Error("load admin article tags: " + err.Error())
		}
		article.TagDTOs = tags
	}
	return articles
}

func (a *MyArticleRepo) ListArticleStatistics() []model.ArticleStatisticsDTO {
	var statistics []model.ArticleStatisticsDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListArticleStatistics).Find(&statistics); err != nil {
		zlog.Error("list article statistics: " + err.Error())
	}
	return statistics
}
