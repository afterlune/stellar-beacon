package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
	"fmt"
)

type CategoryRepo interface {
	ListCategories() []model.CategoryDTO
	ListCategoriesAdmin(current, size int, vo *model.ConditionVO) []*model.CategoryAdminDTO
}

type MyCategoryRepo struct{}

func (c *MyCategoryRepo) ListCategories() []model.CategoryDTO {
	engine := ormInit.GetEngine()
	var categorys []model.CategoryDTO
	err := engine.SQL(pgsql.ListCategories).Find(&categorys)
	if err != nil {
		zlog.Error(err.Error())
	}

	return categorys
}

func (c *MyCategoryRepo) ListCategoriesAdmin(current, size int, vo *model.ConditionVO) []*model.CategoryAdminDTO {
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
