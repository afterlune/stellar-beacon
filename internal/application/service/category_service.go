package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type CategoryService interface {
	ListCategories(context.Context) ([]port.Category, error)
	ListCategoriesAdmin(context.Context, int, int, string) ([]*port.CategoryAdmin, int64, error)
	ListCategoriesAdminBySearch(context.Context, string) ([]port.CategoryOption, error)
	DeleteCategories(context.Context, int, []int) error
	SaveOrUpdateCategory(context.Context, entity.TCategory) error
}

type MyCategoryService struct{ repo port.CategoryRepository }

func NewCategoryService(repo port.CategoryRepository) *MyCategoryService {
	return &MyCategoryService{repo: repo}
}

func (c *MyCategoryService) categoryRepository() port.CategoryRepository {
	if c.repo != nil {
		return c.repo
	}
	return categoryRepo
}

func (c *MyCategoryService) ListCategories(ctx context.Context) ([]port.Category, error) {
	return c.categoryRepository().List(ctx)
}

func (c *MyCategoryService) ListCategoriesAdmin(ctx context.Context, current, size int, keywords string) ([]*port.CategoryAdmin, int64, error) {
	filter := port.CategoryFilter{Keywords: keywords}
	count, err := c.categoryRepository().CountAdmin(ctx, filter)
	if err != nil || count == 0 {
		return []*port.CategoryAdmin{}, count, err
	}
	data, err := c.categoryRepository().ListAdmin(ctx, current, size, filter)
	return data, count, err
}

func (c *MyCategoryService) ListCategoriesAdminBySearch(ctx context.Context, keywords string) ([]port.CategoryOption, error) {
	return c.categoryRepository().Search(ctx, keywords)
}

func (c *MyCategoryService) DeleteCategories(ctx context.Context, userID int, ids []int) error {
	return c.categoryRepository().Delete(ctx, userID, ids)
}

func (c *MyCategoryService) SaveOrUpdateCategory(ctx context.Context, category entity.TCategory) error {
	return c.categoryRepository().SaveOrUpdate(ctx, category)
}

var _ CategoryService = (*MyCategoryService)(nil)
