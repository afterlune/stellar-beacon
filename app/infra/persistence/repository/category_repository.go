package repository

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"context"

	"xorm.io/xorm"
)

var _ port.CategoryRepository = (*MyCategoryRepo)(nil)

type MyCategoryRepo struct {
	engine *xorm.Engine
}

func NewCategoryRepo(engine *xorm.Engine) *MyCategoryRepo {
	return &MyCategoryRepo{engine: engine}
}

func (c *MyCategoryRepo) categorySession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(c.engine, ctx, "category")
}

func categoryFilter(filter port.CategoryFilter) (string, []interface{}) {
	if filter.Keywords == "" {
		return "", nil
	}
	return " WHERE c.category_name LIKE ? ESCAPE '\\'", []interface{}{pgsql.ContainsPattern(filter.Keywords)}
}

func (c *MyCategoryRepo) List(ctx context.Context) ([]port.Category, error) {
	session, err := c.categorySession(ctx)
	if err != nil {
		return nil, err
	}
	var categories []port.Category
	if err := session.SQL(pgsql.ListCategories).Find(&categories); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "category.list", err)
	}
	return categories, nil
}

func (c *MyCategoryRepo) CountAdmin(ctx context.Context, filter port.CategoryFilter) (int64, error) {
	session, err := c.categorySession(ctx)
	if err != nil {
		return 0, err
	}
	where, args := categoryFilter(filter)
	var count int64
	if _, err := session.SQL("SELECT count(1) FROM t_category c"+where, args...).Get(&count); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "category.count_admin", err)
	}
	return count, nil
}

func (c *MyCategoryRepo) ListAdmin(ctx context.Context, current, size int, filter port.CategoryFilter) ([]*port.CategoryAdmin, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := c.categorySession(ctx)
	if err != nil {
		return nil, err
	}
	where, args := categoryFilter(filter)
	query := "SELECT c.id, c.category_name, count(a.id) AS article_count, c.create_time FROM t_category c LEFT JOIN t_article a ON c.id = a.category_id" + where + " GROUP BY c.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var categories []*port.CategoryAdmin
	if err := session.SQL(query, args...).Find(&categories); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "category.list_admin", err)
	}
	return categories, nil
}

func (c *MyCategoryRepo) Search(ctx context.Context, keywords string) ([]port.CategoryOption, error) {
	session, err := c.categorySession(ctx)
	if err != nil {
		return nil, err
	}
	var categories []port.CategoryOption
	if err := session.SQL("SELECT id, category_name FROM t_category WHERE category_name LIKE ? ESCAPE '\\' ORDER BY id DESC", pgsql.ContainsPattern(keywords)).Find(&categories); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "category.search", err)
	}
	return categories, nil
}

func (c *MyCategoryRepo) SaveOrUpdate(ctx context.Context, category entity.TCategory) error {
	session, err := c.categorySession(ctx)
	if err != nil {
		return err
	}
	var existing entity.TCategory
	found, err := session.Select("id").Where("category_name = ?", category.CategoryName).Get(&existing)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "category.check_name", err)
	}
	if found && existing.Id != category.Id {
		return apperrors.Conflict("category.save", "category name already exists")
	}
	return ormInit.WithEngineTx(c.engine, ctx, func(tx *xorm.Session) error {
		var err error
		if category.Id != 0 {
			_, err = tx.ID(category.Id).Update(&category)
		} else {
			_, err = tx.Insert(&category)
		}
		if err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "category.save", err)
		}
		return nil
	})
}

func (c *MyCategoryRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	session, err := c.categorySession(ctx)
	if err != nil {
		return err
	}
	var count int64
	if count, err = session.In("category_id", ids).Count(&entity.TArticle{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "category.check_delete", err)
	}
	if count > 0 {
		return apperrors.Conflict("category.delete", "category has articles")
	}
	if _, err := session.In("id", ids).Delete(&entity.TCategory{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "category.delete", err)
	}
	return nil
}
