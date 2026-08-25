package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"xorm.io/builder"
	"xorm.io/xorm"
)

type CategoryService interface {
	ListCategories() model.ResultVO
	ListCategoriesAdmin(c *gin.Context) model.ResultVO
	ListCategoriesAdminBySearch(c *gin.Context) model.ResultVO
	DeleteCategories(c *gin.Context) model.ResultVO
	SaveOrUpdateCategory(c *gin.Context) model.ResultVO
}

type MyCategoryService struct{}

func (c *MyCategoryService) ListCategories() model.ResultVO {
	return model.ResultOkWithData(categoryRepo.ListCategories())
}

func (c *MyCategoryService) ListCategoriesAdmin(ctx *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := ctx.ShouldBind(&conditionVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var count int64
	if conditionVO.Keywords != "" {
		count, err = engine.Where(builder.Like{"category_name", conditionVO.Keywords}).Count(&entity.TCategory{})
	} else {
		count, err = engine.Count(&entity.TCategory{})
	}
	if err != nil {
		zlog.Error(err.Error())
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	data := categoryRepo.ListCategoriesAdmin(conditionVO.Current, conditionVO.Size, &conditionVO)
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCategoryService) ListCategoriesAdminBySearch(ctx *gin.Context) model.ResultVO {
	var conditionVO model.ConditionVO
	err := ctx.ShouldBind(&conditionVO)
	if err != nil {
		zlog.Error(err.Error())
	}
	var categorys []entity.TCategory
	err = ormInit.GetEngine().Where(builder.Like{"category_name", conditionVO.Keywords}).OrderBy("id").Desc("id").Find(&categorys)
	if err != nil {
		zlog.Error(err.Error())
	}
	marshal, err := json.Marshal(categorys)
	if err != nil {
		zlog.Error(err.Error())
	}
	var categoryOptionDTOs []model.CategoryOptionDTO
	err = json.Unmarshal(marshal, &categoryOptionDTOs)
	if err != nil {
		zlog.Error(err.Error())
	}
	return model.ResultOkWithData(categoryOptionDTOs)
}

func (c *MyCategoryService) DeleteCategories(ctx *gin.Context) model.ResultVO {
	var iDs []int
	err := ctx.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	count, err := engine.In("category_id", iDs).Count(&entity.TArticle{})
	if err != nil {
		zlog.Error(err.Error())
	}
	if count > 0 {
		return model.ResultFailWithMessage("删除失败，该分类下存在文章")
	}
	_, err = engine.In("id", iDs).Delete(&entity.TCategory{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (c *MyCategoryService) SaveOrUpdateCategory(ctx *gin.Context) model.ResultVO {
	var categoryVO model.CategoryVO
	err := ctx.ShouldBind(&categoryVO)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var existCategory entity.TCategory
	_, err = engine.Select("id").Where("category_name = ?", categoryVO.CategoryName).Get(&existCategory)
	if err != nil {
		zlog.Error(err.Error())
	}
	if existCategory.CategoryName != "" && existCategory.Id != categoryVO.Id {
		return model.ResultFailWithMessage("分类名已存在")
	}
	category := entity.TCategory{Id: categoryVO.Id, CategoryName: categoryVO.CategoryName}
	if err := ormInit.WithTx(ctx.Request.Context(), func(session *xorm.Session) error {
		if categoryVO.Id != 0 {
			_, err = session.ID(categoryVO.Id).Update(&category)
		} else {
			_, err = session.Insert(&category)
		}
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
