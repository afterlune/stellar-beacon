package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"container/list"
	"context"

	"github.com/gin-gonic/gin"
)

type CategoryService interface {
	ListCategories() model.ResultVO
	ListCategoriesAdmin(c *gin.Context) model.ResultVO
	ListCategoriesAdminBySearch(c *gin.Context) model.ResultVO
	DeleteCategories(c *gin.Context) model.ResultVO
	SaveOrUpdateCategory(c *gin.Context) model.ResultVO
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

func (c *MyCategoryService) ListCategories() model.ResultVO {
	data, err := c.categoryRepository().List(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (c *MyCategoryService) ListCategoriesAdmin(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CategoryFilter{Keywords: vo.Keywords}
	count, err := c.categoryRepository().CountAdmin(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	data, err := c.categoryRepository().ListAdmin(ctx.Request.Context(), vo.Current, vo.Size, filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCategoryService) ListCategoriesAdminBySearch(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, err := c.categoryRepository().Search(ctx.Request.Context(), vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (c *MyCategoryService) DeleteCategories(ctx *gin.Context) model.ResultVO {
	var ids []int
	if err := ctx.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.categoryRepository().Delete(ctx.Request.Context(), ids); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("删除失败，该分类下存在文章")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCategoryService) SaveOrUpdateCategory(ctx *gin.Context) model.ResultVO {
	var vo model.CategoryVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	category := entity.TCategory{Id: vo.Id, CategoryName: vo.CategoryName}
	if err := c.categoryRepository().SaveOrUpdate(ctx.Request.Context(), category); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("分类名已存在")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

var _ CategoryService = (*MyCategoryService)(nil)
