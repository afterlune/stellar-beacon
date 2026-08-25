package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
)

type TagRepo interface {
	ListTags() []*model.TagDTO
	ListTopTenTags() []*model.TagDTO
	ListTagNamesByArticleId(articleId int) (s []string)
	ListTagsAdmin(current, size int, vo *model.ConditionVO) []*model.TagAdminDTO
}

type MyTagRepo struct{}

func (t *MyTagRepo) ListTags() []*model.TagDTO {
	var tags []*model.TagDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListTags).Find(&tags); err != nil {
		zlog.Error("list tags: " + err.Error())
	}
	return tags
}

func (t *MyTagRepo) ListTopTenTags() []*model.TagDTO {
	var tags []*model.TagDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListTopTenTags).Find(&tags); err != nil {
		zlog.Error("list top tags: " + err.Error())
	}
	return tags
}

func (t *MyTagRepo) ListTagNamesByArticleId(articleID int) []string {
	var names []string
	if err := ormInit.GetEngine().SQL("SELECT tag_name FROM t_tag t JOIN t_article_tag at ON t.id = at.tag_id WHERE article_id = ?", articleID).Find(&names); err != nil {
		zlog.Error("list article tag names: " + err.Error())
	}
	return names
}

func (t *MyTagRepo) ListTagsAdmin(current, size int, vo *model.ConditionVO) []*model.TagAdminDTO {
	limit, offset := pgsql.Page(current, size)
	query := "SELECT t.id, tag_name, COUNT(at.article_id) AS article_count, t.create_time FROM t_tag t LEFT JOIN t_article_tag at ON t.id = at.tag_id"
	args := []interface{}{}
	if vo.Keywords != "" {
		query += " WHERE t.tag_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	query += " GROUP BY t.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var tags []*model.TagAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&tags); err != nil {
		zlog.Error("list admin tags: " + err.Error())
	}
	return tags
}
