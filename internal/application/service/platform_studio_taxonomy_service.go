package service

import (
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

func (s *MyPlatformService) ListOwnedCategories(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	categories, err := s.platformRepo().ListOwnedCategories(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(categories)
}

func (s *MyPlatformService) SaveOwnedCategory(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTaxonomyVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	category, err := s.platformRepo().SaveOwnedCategory(c.Request.Context(), entity.TCategory{
		Id: vo.Id, UserId: user.UserInfoId, CategoryName: strings.TrimSpace(vo.Name),
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(category)
}

func (s *MyPlatformService) DeleteOwnedCategory(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "categoryId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedCategory(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (s *MyPlatformService) ListOwnedTags(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	tags, err := s.platformRepo().ListOwnedTags(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(tags)
}

func (s *MyPlatformService) SaveOwnedTag(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioTaxonomyVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	tag, err := s.platformRepo().SaveOwnedTag(c.Request.Context(), entity.TTag{
		Id: vo.Id, UserId: user.UserInfoId, TagName: strings.TrimSpace(vo.Name),
	})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(tag)
}

func (s *MyPlatformService) DeleteOwnedTag(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	id, err := pathID(c, "tagId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := s.platformRepo().DeleteOwnedTag(c.Request.Context(), user.UserInfoId, id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}
