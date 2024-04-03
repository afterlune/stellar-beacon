package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"fmt"
)

type TagRepo interface {
	ListTags() []*model.TagDTO
	ListTopTenTags() []*model.TagDTO
	ListTagNamesByArticleId(articleId int) (s []string)
	ListTagsAdmin(current, size int, vo *model.ConditionVO) []*model.TagAdminDTO
}

type MyTagRepo struct {
}

func (t *MyTagRepo) ListTags() []*model.TagDTO {
	engine := ormInit.GetEngine()
	var tags []*model.TagDTO
	err := engine.SQL(pgsql.ListTags).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}

	return tags
}

func (t *MyTagRepo) ListTopTenTags() []*model.TagDTO {
	engine := ormInit.GetEngine()
	var tags []*model.TagDTO
	err := engine.SQL(pgsql.ListTopTenTags).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}

	return tags
}

func (t *MyTagRepo) ListTagNamesByArticleId(articleId int) (s []string) {
	engine := ormInit.GetEngine()
	sql := fmt.Sprintf(pgsql.ListTagNamesByArticleId, articleId)
	err := engine.SQL(sql).Find(&s)
	if err != nil {
		zlog.Error(err.Error())
	}

	return s
}

func (t *MyTagRepo) ListTagsAdmin(current, size int, vo *model.ConditionVO) []*model.TagAdminDTO {
	s := ""
	if vo.Keywords != "" {
		s += " where tag_name like '%" + vo.Keywords + "%'"
	}
	s = fmt.Sprintf(pgsql.ListTagsAdmin, s, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var tagsAdmin []*model.TagAdminDTO
	err := engine.SQL(s).Find(&tagsAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	return tagsAdmin
}
