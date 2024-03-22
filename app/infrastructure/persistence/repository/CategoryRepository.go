package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
)

func ListCategories() []model.CategoryDTO {
	engine := ormInit.GetEngine()
	var categorys []model.CategoryDTO
	err := engine.SQL(pgsql.ListCategories).Find(&categorys)
	if err != nil {
		zlog.Error(err.Error())
	}

	return categorys
}

func ListCategoriesAdmin(current, size int, vo *model.ConditionVO) []*model.CategoryAdminDTO {
	engine := ormInit.GetEngine()
	var categorysadmin []*model.CategoryAdminDTO
	s := ""
	if vo.Keywords != "" {
		s += " where category_name like '%" + vo.Keywords + "%'"
	}
	s = fmt.Sprintf(pgsql.ListCategoriesAdmin, s, size, (current-1)*size)
	err := engine.SQL(s).Find(&categorysadmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	return categorysadmin
}
