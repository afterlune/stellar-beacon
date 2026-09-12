package service

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"context"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MenuService interface {
	ListMenus(c *gin.Context) model.ResultVO
	SaveOrUpdateMenu(c *gin.Context) model.ResultVO
	UpdateMenuIsHidden(c *gin.Context) model.ResultVO
	DeleteMenu(c *gin.Context) model.ResultVO
	ListMenuOptions() model.ResultVO
	ListUserMenus(userInfoId int) model.ResultVO
	listCatalogs(menus []entity.TMenu) []entity.TMenu
	getMenuMap(menus []entity.TMenu) map[int][]entity.TMenu
	convertUserMenuList(catalogs []entity.TMenu, hm map[int][]entity.TMenu) []model.UserMenuDTO
}

type MyMenuSService struct{ repo port.MenuRepository }

func NewMenuService(repo port.MenuRepository) *MyMenuSService { return &MyMenuSService{repo: repo} }

func (m *MyMenuSService) menuRepository() port.MenuRepository {
	if m.repo != nil {
		return m.repo
	}
	return menuRepo
}

func (m *MyMenuSService) ListMenus(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	menus, err := m.menuRepository().List(c.Request.Context(), vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(m.menuDTOs(menus))
}

func (m *MyMenuSService) SaveOrUpdateMenu(c *gin.Context) model.ResultVO {
	var vo model.MenuVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	menu := entity.TMenu{Id: vo.Id, Name: vo.Name, Path: vo.Path, Component: vo.Component, Icon: vo.Icon, OrderNum: vo.OrderNum, ParentId: vo.ParentId, IsHidden: vo.IsHidden}
	if err := m.menuRepository().SaveOrUpdate(c.Request.Context(), menu); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (m *MyMenuSService) UpdateMenuIsHidden(c *gin.Context) model.ResultVO {
	var vo model.IsHiddenVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := m.menuRepository().UpdateHidden(c.Request.Context(), vo.Id, vo.IsHidden); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (m *MyMenuSService) DeleteMenu(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("menuId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := m.menuRepository().Delete(c.Request.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return model.ResultFailWithMessage("菜单下有角色关联")
		}
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (m *MyMenuSService) ListMenuOptions() model.ResultVO {
	menus, err := m.menuRepository().ListOptions(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	catalogs := m.listCatalogs(menus)
	children := m.getMenuMap(menus)
	options := make([]model.LabelOptionDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		items := children[catalog.Id]
		sort.Slice(items, func(i, j int) bool { return items[i].OrderNum < items[j].OrderNum })
		childrenDTO := make([]model.LabelOptionDTO, 0, len(items))
		for _, item := range items {
			childrenDTO = append(childrenDTO, model.LabelOptionDTO{Id: item.Id, Label: item.Name})
		}
		options = append(options, model.LabelOptionDTO{Id: catalog.Id, Label: catalog.Name, Children: childrenDTO})
	}
	return model.ResultOkWithData(options)
}

func (m *MyMenuSService) ListUserMenus(userInfoID int) model.ResultVO {
	menus, err := m.menuRepository().ListByUserInfoID(context.Background(), userInfoID)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(m.convertUserMenuList(m.listCatalogs(menus), m.getMenuMap(menus)))
}

func (m *MyMenuSService) listCatalogs(menus []entity.TMenu) []entity.TMenu {
	result := make([]entity.TMenu, 0)
	for _, menu := range menus {
		if menu.ParentId == 0 {
			result = append(result, menu)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderNum < result[j].OrderNum })
	return result
}

func (m *MyMenuSService) getMenuMap(menus []entity.TMenu) map[int][]entity.TMenu {
	result := make(map[int][]entity.TMenu)
	for _, menu := range menus {
		if menu.ParentId != 0 {
			result[menu.ParentId] = append(result[menu.ParentId], menu)
		}
	}
	return result
}

func (m *MyMenuSService) menuDTOs(menus []entity.TMenu) []model.MenuDTO {
	catalogs := m.listCatalogs(menus)
	children := m.getMenuMap(menus)
	result := make([]model.MenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		var dto model.MenuDTO
		StructCopy(catalog, &dto)
		var childDTOs []model.MenuDTO
		StructCopy(children[catalog.Id], &childDTOs)
		sort.Slice(childDTOs, func(i, j int) bool { return childDTOs[i].OrderNum < childDTOs[j].OrderNum })
		dto.Children = childDTOs
		result = append(result, dto)
		delete(children, catalog.Id)
	}
	if len(children) > 0 {
		for _, childList := range children {
			var dtos []model.MenuDTO
			StructCopy(childList, &dtos)
			result = append(result, dtos...)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderNum < result[j].OrderNum })
	return result
}

func (m *MyMenuSService) convertUserMenuList(catalogs []entity.TMenu, hm map[int][]entity.TMenu) []model.UserMenuDTO {
	result := make([]model.UserMenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		children := hm[catalog.Id]
		dto := model.UserMenuDTO{
			Name:   catalog.Name,
			Path:   catalog.Path,
			Icon:   catalog.Icon,
			Hidden: catalog.IsHidden == True,
		}
		if len(children) == 0 {
			dto.Component = Component
			dto.Children = []model.UserMenuDTO{{
				Name:      catalog.Name,
				Icon:      catalog.Icon,
				Component: catalog.Component,
				Path:      "",
			}}
		} else {
			sort.Slice(children, func(i, j int) bool { return children[i].OrderNum < children[j].OrderNum })
			for _, child := range children {
				dto.Children = append(dto.Children, model.UserMenuDTO{Name: child.Name, Path: child.Path, Icon: child.Icon, Component: child.Component, Hidden: child.IsHidden == True})
			}
		}
		result = append(result, dto)
	}
	return result
}
