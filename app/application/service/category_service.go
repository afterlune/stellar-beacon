package service

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"container/list"
	"context"
)

type CategoryService interface {
	ListCategories(ctx context.Context) port.ResultVO
	ListCategoriesAdmin(c port.Request) port.ResultVO
	ListCategoriesAdminBySearch(c port.Request) port.ResultVO
	DeleteCategories(c port.Request) port.ResultVO
	SaveOrUpdateCategory(c port.Request) port.ResultVO
}

type MyCategoryService struct {
	repo port.CategoryRepository
}

func NewCategoryService(repo port.CategoryRepository) *MyCategoryService {
	return &MyCategoryService{repo: repo}
}

func (c *MyCategoryService) categoryRepository() port.CategoryRepository {
	if c.repo != nil {
		return c.repo
	}
	return categoryRepo
}

func (c *MyCategoryService) ListCategories(ctx context.Context) port.ResultVO {
	data, err := c.categoryRepository().List(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (c *MyCategoryService) ListCategoriesAdmin(ctx port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CategoryFilter{Keywords: vo.Keywords}
	count, err := c.categoryRepository().CountAdmin(ctx.Context(), filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := c.categoryRepository().ListAdmin(ctx.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCategoryService) ListCategoriesAdminBySearch(ctx port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	data, err := c.categoryRepository().Search(ctx.Context(), vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(data)
}

func (c *MyCategoryService) DeleteCategories(ctx port.Request) port.ResultVO {
	var ids []int
	if err := ctx.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.categoryRepository().Delete(ctx.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("删除失败，该分类下存在文章")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (c *MyCategoryService) SaveOrUpdateCategory(ctx port.Request) port.ResultVO {
	var vo port.CategoryVO
	if err := ctx.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	category := port.TCategory{Id: vo.Id, CategoryName: vo.CategoryName}
	if err := c.categoryRepository().SaveOrUpdate(ctx.Context(), category); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("分类名已存在")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

var _ CategoryService = (*MyCategoryService)(nil)
