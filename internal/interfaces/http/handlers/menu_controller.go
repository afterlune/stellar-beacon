package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ListMenus
// @Summary         菜单模块
// @Description    查看菜单列表
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/menus [GET]
func ListMenus(c *gin.Context) {
	var query model.ConditionVO
	if err := c.ShouldBind(&query); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	menus, err := menuService.ListMenus(c.Request.Context(), query.Keywords)
	if err != nil {
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(menuDTOs(menus)))
}

// SaveOrUpdateMenu
// @Summary         菜单模块
// @Description    新增或修改菜单
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/menus [POST]
func SaveOrUpdateMenu(c *gin.Context) {
	var request model.MenuVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	menu := entity.TMenu{Id: request.Id, Name: request.Name, Path: request.Path, Component: request.Component, Icon: request.Icon, OrderNum: request.OrderNum, ParentId: request.ParentId, IsHidden: request.IsHidden}
	if err := menuService.SaveOrUpdateMenu(c.Request.Context(), menu); err != nil {
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// UpdateMenuIsHidden
// @Summary         菜单模块
// @Description    修改目录是否隐藏
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/menus/{menuId}/visibility [PUT]
func UpdateMenuIsHidden(c *gin.Context) {
	var request model.IsHiddenVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := menuService.UpdateMenuIsHidden(c.Request.Context(), request.Id, request.IsHidden); err != nil {
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// DeleteMenu
// @Summary         菜单模块
// @Description    删除菜单
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/menus/{menuId} [DELETE]
func DeleteMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("menuId"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	if err := menuService.DeleteMenu(c.Request.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			c.JSON(http.StatusOK, model.ResultFailWithMessage("菜单下有角色关联"))
			return
		}
		writeMenuError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.ResultOk())
}

// ListMenuOptions
// @Summary         菜单模块
// @Description    查看角色菜单选项
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/roles/menu-options [GET]
func ListMenuOptions(c *gin.Context) {
	menus, err := menuService.ListMenuOptions(c.Request.Context())
	if err != nil {
		writeMenuError(c, err)
		return
	}
	catalogs, children := menuTree(menus)
	options := make([]model.LabelOptionDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		items := children[catalog.Id]
		sort.Slice(items, func(i, j int) bool { return items[i].OrderNum < items[j].OrderNum })
		option := model.LabelOptionDTO{Id: catalog.Id, Label: catalog.Name, Children: make([]model.LabelOptionDTO, 0, len(items))}
		for _, item := range items {
			option.Children = append(option.Children, model.LabelOptionDTO{Id: item.Id, Label: item.Name})
		}
		options = append(options, option)
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(options))
}

// ListUserMenus
// @Summary         菜单模块
// @Description    查看当前用户菜单
// @Success        200 {object} model.ResultVO
// @Router         /v1/admin/me/menu [GET]
func ListUserMenus(c *gin.Context) {
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithStatus(model.NO_LOGIN))
		return
	}
	menus, err := menuService.ListUserMenus(c.Request.Context(), userID)
	if err != nil {
		writeMenuError(c, err)
		return
	}
	catalogs, children := menuTree(menus)
	result := make([]model.UserMenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		items := children[catalog.Id]
		dto := model.UserMenuDTO{Name: catalog.Name, Path: catalog.Path, Icon: catalog.Icon, Hidden: catalog.IsHidden == 1}
		if len(items) == 0 {
			dto.Component = "Layout"
			dto.Children = []model.UserMenuDTO{{Name: catalog.Name, Icon: catalog.Icon, Component: catalog.Component}}
		} else {
			sort.Slice(items, func(i, j int) bool { return items[i].OrderNum < items[j].OrderNum })
			for _, item := range items {
				dto.Children = append(dto.Children, model.UserMenuDTO{Name: item.Name, Path: item.Path, Icon: item.Icon, Component: item.Component, Hidden: item.IsHidden == 1})
			}
		}
		result = append(result, dto)
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(result))
}

func menuTree(menus []entity.TMenu) ([]entity.TMenu, map[int][]entity.TMenu) {
	catalogs := make([]entity.TMenu, 0)
	children := make(map[int][]entity.TMenu)
	for _, menu := range menus {
		if menu.ParentId == 0 {
			catalogs = append(catalogs, menu)
		} else {
			children[menu.ParentId] = append(children[menu.ParentId], menu)
		}
	}
	sort.Slice(catalogs, func(i, j int) bool { return catalogs[i].OrderNum < catalogs[j].OrderNum })
	return catalogs, children
}

func menuDTOs(menus []entity.TMenu) []model.MenuDTO {
	catalogs, children := menuTree(menus)
	result := make([]model.MenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		dto := menuDTO(catalog)
		items := children[catalog.Id]
		sort.Slice(items, func(i, j int) bool { return items[i].OrderNum < items[j].OrderNum })
		for _, item := range items {
			dto.Children = append(dto.Children, menuDTO(item))
		}
		result = append(result, dto)
		delete(children, catalog.Id)
	}
	for _, items := range children {
		for _, item := range items {
			result = append(result, menuDTO(item))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderNum < result[j].OrderNum })
	return result
}

func menuDTO(menu entity.TMenu) model.MenuDTO {
	return model.MenuDTO{Id: menu.Id, Name: menu.Name, Path: menu.Path, Component: menu.Component, Icon: menu.Icon, CreateTime: menu.CreateTime, OrderNum: menu.OrderNum, IsHidden: menu.IsHidden}
}

func writeMenuError(c *gin.Context, err error) {
	c.JSON(http.StatusOK, model.ResultFromError(err))
}
