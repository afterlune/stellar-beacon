package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
)

func ListTags() []*model.TagDTO {
	engine := ormInit.GetEngine()
	var tags []*model.TagDTO
	err := engine.SQL(pgsql.ListTags).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}

	return tags
}

func ListTopTenTags() []*model.TagDTO {
	engine := ormInit.GetEngine()
	var tags []*model.TagDTO
	err := engine.SQL(pgsql.ListTopTenTags).Find(&tags)
	if err != nil {
		zlog.Error(err.Error())
	}

	return tags
}

func ListTagNamesByArticleId(articleId int) (s []string) {
	engine := ormInit.GetEngine()
	sql := fmt.Sprintf(pgsql.ListTagNamesByArticleId, articleId)
	err := engine.SQL(sql).Find(&s)
	if err != nil {
		zlog.Error(err.Error())
	}

	return s
}

func ListTagsAdmin(current, size int, vo *model.ConditionVO) []*model.TagAdminDTO {
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
