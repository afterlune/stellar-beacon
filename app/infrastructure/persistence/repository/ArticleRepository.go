package repository

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"container/list"
	"fmt"
	"strconv"
)

func ListTopAndFeaturedArticles() []*model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	err := engine.SQL(pgsql.ListTopAndFeaturedArticles).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range articles {
		var tags []entity.TTag
		s := fmt.Sprintf(pgsql.ArticleTags, v.Id)
		err := engine.SQL(s).Find(&tags)
		if err != nil {
			zlog.Error(err.Error())
		}
		if len(tags) != 0 {
			v.Tags = tags
			continue
		}
		v.Tags = list.New()
	}
	return articles
}

func ListArticles(current, size int) []*model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	s1 := fmt.Sprintf(pgsql.ListArticles, size, (current-1)*size)
	err := engine.SQL(s1).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range articles {
		var tags []entity.TTag
		s2 := fmt.Sprintf(pgsql.ArticleTags, v.Id)
		err := engine.SQL(s2).Find(&tags)
		if err != nil {
			zlog.Error(err.Error())
		}
		if len(tags) != 0 {
			v.Tags = tags
			continue
		}
		v.Tags = list.New()
	}
	return articles
}

func GetArticlesByCategoryId(current, size int, categoryId int) []*model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	s1 := fmt.Sprintf(pgsql.GetArticlesByCategoryId, categoryId, size, (current-1)*size)
	err := engine.SQL(s1).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range articles {
		var tags []entity.TTag
		s2 := fmt.Sprintf(pgsql.ArticleTags, v.Id)
		err := engine.SQL(s2).Find(&tags)
		if err != nil {
			zlog.Error(err.Error())
		}
		if len(tags) != 0 {
			v.Tags = tags
			continue
		}
		v.Tags = list.New()
	}
	return articles
}

func GetArticleById(articleId int) model.ArticleDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleDTO
	s1 := fmt.Sprintf(pgsql.GetArticleById, articleId)
	_, err := engine.SQL(s1).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
	}

	var tags []entity.TTag
	s2 := fmt.Sprintf(pgsql.ArticleTags, article.Id)
	err = engine.SQL(s2).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	if len(tags) != 0 {
		article.Tags = tags
	} else {
		article.Tags = list.New()
	}
	return article
}

func GetPreArticleById(articleId int) model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleCardDTO
	s1 := fmt.Sprintf(pgsql.GetPreArticleById, articleId)
	get, err := engine.SQL(s1).Get(&article)
	if !get {
		return model.ArticleCardDTO{}
	}
	if err != nil {
		zlog.Error(err.Error())
	}

	var tags []entity.TTag
	s2 := fmt.Sprintf(pgsql.ArticleTags, article.Id)
	err = engine.SQL(s2).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	if len(tags) != 0 {
		article.Tags = tags
	} else {
		article.Tags = list.New()
	}
	return article
}

func GetNextArticleById(articleId int) model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleCardDTO
	s1 := fmt.Sprintf(pgsql.GetNextArticleById, articleId)
	get, err := engine.SQL(s1).Get(&article)
	if !get {
		return model.ArticleCardDTO{}
	}
	if err != nil {
		zlog.Error(err.Error())
	}

	var tags []entity.TTag
	s2 := fmt.Sprintf(pgsql.ArticleTags, article.Id)
	err = engine.SQL(s2).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	if len(tags) != 0 {
		article.Tags = tags
	} else {
		article.Tags = list.New()
	}
	return article
}

func GetFirstArticle() model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleCardDTO
	_, err := engine.SQL(pgsql.GetFirstArticle).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
	}

	var tags []entity.TTag
	s := fmt.Sprintf(pgsql.ArticleTags, article.Id)
	err = engine.SQL(s).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	if len(tags) != 0 {
		article.Tags = tags
	} else {
		article.Tags = list.New()
	}
	return article
}

func GetLastArticle() model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var article model.ArticleCardDTO
	_, err := engine.SQL(pgsql.GetLastArticle).Get(&article)
	if err != nil {
		zlog.Error(err.Error())
	}

	var tags []entity.TTag
	s := fmt.Sprintf(pgsql.ArticleTags, article.Id)
	err = engine.SQL(s).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}
	if len(tags) != 0 {
		article.Tags = tags
	} else {
		article.Tags = list.New()
	}
	return article
}

func ListArticlesByTagId(current, size int, tagId int) []*model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []*model.ArticleCardDTO
	s1 := fmt.Sprintf(pgsql.ListArticlesByTagId, tagId, size, (current-1)*size)
	err := engine.SQL(s1).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range articles {
		var tags []entity.TTag
		s2 := fmt.Sprintf(pgsql.ArticleTags, v.Id)
		err := engine.SQL(s2).Find(&tags)
		if err != nil {
			zlog.Error(err.Error())
		}
		if len(tags) != 0 {
			v.Tags = tags
			continue
		}
		v.Tags = list.New()
	}
	return articles
}

func ListArchives(current, size int) []model.ArticleCardDTO {
	engine := ormInit.GetEngine()
	var articles []model.ArticleCardDTO
	s := fmt.Sprintf(pgsql.ListArchives, size, (current-1)*size)
	err := engine.SQL(s).Find(&articles)
	if err != nil {
		zlog.Error(err.Error())
	}

	return articles
}

func CountArticleAdmins(vo *model.ConditionVO) (count int) {
	s := fmt.Sprintf("where is_delete = %d", vo.IsDelete)
	if vo.Keywords != "" {
		s += " and article_title like '%" + vo.Keywords + "%'"
	}
	if vo.Status != 0 {
		s += " and status = " + strconv.Itoa(vo.Status)
	}
	if vo.CategoryId != 0 {
		s += " and category_id = " + strconv.Itoa(vo.CategoryId)
	}
	if vo.Type != 0 {
		s += " and type = " + strconv.Itoa(vo.Type)
	}
	if vo.TagId != 0 {
		s += " and at.tag_id = " + strconv.Itoa(vo.TagId)
	}
	engine := ormInit.GetEngine()
	s = fmt.Sprintf(pgsql.CountArticleAdmins, s)
	_, err := engine.SQL(s).Get(&count)
	if err != nil {
		zlog.Error(err.Error())
	}

	return count
}

func ListArticlesAdmin(current, size int, vo *model.ConditionVO) []*model.ArticleAdminDTO {
	s := fmt.Sprintf("where is_delete = %d", vo.IsDelete)
	if vo.Keywords != "" {
		s += " and article_title like '%" + vo.Keywords + "%'"
	}
	if vo.Status != 0 {
		s += " and status = " + strconv.Itoa(vo.Status)
	}
	if vo.CategoryId != 0 {
		s += " and category_id = " + strconv.Itoa(vo.CategoryId)
	}
	if vo.Type != 0 {
		s += " and type = " + strconv.Itoa(vo.Type)
	}
	if vo.TagId != 0 {
		s += " and id in (select article_id from t_article_tag where tag_id = " + strconv.Itoa(vo.TagId) + ")"
	}
	engine := ormInit.GetEngine()
	var articlesAdmin []*model.ArticleAdminDTO
	s = fmt.Sprintf(pgsql.ListArticlesAdmin, s, size, (current-1)*size)
	err := engine.SQL(s).Find(&articlesAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	for _, v := range articlesAdmin {
		var tagDTOs []model.TagDTO
		s2 := fmt.Sprintf(pgsql.ArticlesAdminTags, v.Id)
		err := engine.SQL(s2).Find(&tagDTOs)
		if err != nil {
			zlog.Error(err.Error())
		}

		v.TagDTOs = tagDTOs
	}
	return articlesAdmin
}

func ListArticleStatistics() []model.ArticleStatisticsDTO {
	engine := ormInit.GetEngine()
	var articleStatistics []model.ArticleStatisticsDTO
	err := engine.SQL(pgsql.ListArticleStatistics).Find(&articleStatistics)
	if err != nil {
		zlog.Error(err.Error())
	}

	return articleStatistics
}
