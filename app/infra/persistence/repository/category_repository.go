package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
)

type CategoryRepo interface {
	ListCategories() []model.CategoryDTO
	ListCategoriesAdmin(current, size int, vo *model.ConditionVO) []*model.CategoryAdminDTO
}

type MyCategoryRepo struct{}

func (c *MyCategoryRepo) ListCategories() []model.CategoryDTO {
	var categories []model.CategoryDTO
	if err := ormInit.GetEngine().SQL(pgsql.ListCategories).Find(&categories); err != nil {
		zlog.Error("list categories: " + err.Error())
	}
	return categories
}

func (c *MyCategoryRepo) ListCategoriesAdmin(current, size int, vo *model.ConditionVO) []*model.CategoryAdminDTO {
	limit, offset := pgsql.Page(current, size)
	query := "SELECT c.id, c.category_name, count(a.id) AS article_count, c.create_time FROM t_category c LEFT JOIN t_article a ON c.id = a.category_id"
	args := []interface{}{}
	if vo.Keywords != "" {
		query += " WHERE c.category_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	query += " GROUP BY c.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var categories []*model.CategoryAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&categories); err != nil {
		zlog.Error("list admin categories: " + err.Error())
	}
	return categories
}
