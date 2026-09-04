package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"sort"
	"strconv"
)

type MenuService interface {
	ListMenus(c port.Request) port.ResultVO
	SaveOrUpdateMenu(c port.Request) port.ResultVO
	UpdateMenuIsHidden(c port.Request) port.ResultVO
	DeleteMenu(c port.Request) port.ResultVO
	ListMenuOptions(ctx context.Context) port.ResultVO
	ListUserMenus(ctx context.Context, userInfoId int) port.ResultVO
	listCatalogs(menus []port.TMenu) []port.TMenu
	getMenuMap(menus []port.TMenu) map[int][]port.TMenu
	convertUserMenuList(catalogs []port.TMenu, hm map[int][]port.TMenu) []port.UserMenuDTO
}

type MyMenuSService struct{ repo port.MenuRepository }

func NewMenuService(repo port.MenuRepository) *MyMenuSService { return &MyMenuSService{repo: repo} }

func (m *MyMenuSService) menuRepository() port.MenuRepository {
	if m.repo != nil {
		return m.repo
	}
	return menuRepo
}

func (m *MyMenuSService) ListMenus(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	menus, err := m.menuRepository().List(c.Context(), vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(m.menuDTOs(menus))
}

func (m *MyMenuSService) SaveOrUpdateMenu(c port.Request) port.ResultVO {
	var vo port.MenuVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	menu := port.TMenu{Id: vo.Id, Name: vo.Name, Path: vo.Path, Component: vo.Component, Icon: vo.Icon, OrderNum: vo.OrderNum, ParentId: vo.ParentId, IsHidden: vo.IsHidden}
	if err := m.menuRepository().SaveOrUpdate(c.Context(), menu); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (m *MyMenuSService) UpdateMenuIsHidden(c port.Request) port.ResultVO {
	var vo port.IsHiddenVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := m.menuRepository().UpdateHidden(c.Context(), vo.Id, vo.IsHidden); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (m *MyMenuSService) DeleteMenu(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("menuId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := m.menuRepository().Delete(c.Context(), id); err != nil {
		if apperrors.IsKind(err, apperrors.KindConflict) {
			return port.ResultFailWithMessage("菜单下有角色关联")
		}
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (m *MyMenuSService) ListMenuOptions(ctx context.Context) port.ResultVO {
	menus, err := m.menuRepository().ListOptions(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	catalogs := m.listCatalogs(menus)
	children := m.getMenuMap(menus)
	options := make([]port.LabelOptionDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		items := children[catalog.Id]
		sort.Slice(items, func(i, j int) bool { return items[i].OrderNum < items[j].OrderNum })
		childrenDTO := make([]port.LabelOptionDTO, 0, len(items))
		for _, item := range items {
			childrenDTO = append(childrenDTO, port.LabelOptionDTO{Id: item.Id, Label: item.Name})
		}
		options = append(options, port.LabelOptionDTO{Id: catalog.Id, Label: catalog.Name, Children: childrenDTO})
	}
	return port.ResultOkWithData(options)
}

func (m *MyMenuSService) ListUserMenus(ctx context.Context, userInfoID int) port.ResultVO {
	menus, err := m.menuRepository().ListByUserInfoID(ctx, userInfoID)
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(m.convertUserMenuList(m.listCatalogs(menus), m.getMenuMap(menus)))
}

func (m *MyMenuSService) listCatalogs(menus []port.TMenu) []port.TMenu {
	result := make([]port.TMenu, 0)
	for _, menu := range menus {
		if menu.ParentId == 0 {
			result = append(result, menu)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderNum < result[j].OrderNum })
	return result
}

func (m *MyMenuSService) getMenuMap(menus []port.TMenu) map[int][]port.TMenu {
	result := make(map[int][]port.TMenu)
	for _, menu := range menus {
		if menu.ParentId != 0 {
			result[menu.ParentId] = append(result[menu.ParentId], menu)
		}
	}
	return result
}

func (m *MyMenuSService) menuDTOs(menus []port.TMenu) []port.MenuDTO {
	catalogs := m.listCatalogs(menus)
	children := m.getMenuMap(menus)
	result := make([]port.MenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		var dto port.MenuDTO
		support.StructCopy(catalog, &dto)
		var childDTOs []port.MenuDTO
		support.StructCopy(children[catalog.Id], &childDTOs)
		sort.Slice(childDTOs, func(i, j int) bool { return childDTOs[i].OrderNum < childDTOs[j].OrderNum })
		dto.Children = childDTOs
		result = append(result, dto)
		delete(children, catalog.Id)
	}
	if len(children) > 0 {
		for _, childList := range children {
			var dtos []port.MenuDTO
			support.StructCopy(childList, &dtos)
			result = append(result, dtos...)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].OrderNum < result[j].OrderNum })
	return result
}

func (m *MyMenuSService) convertUserMenuList(catalogs []port.TMenu, hm map[int][]port.TMenu) []port.UserMenuDTO {
	result := make([]port.UserMenuDTO, 0, len(catalogs))
	for _, catalog := range catalogs {
		children := hm[catalog.Id]
		dto := port.UserMenuDTO{
			Name:   catalog.Name,
			Path:   catalog.Path,
			Icon:   catalog.Icon,
			Hidden: catalog.IsHidden == support.True,
		}
		if len(children) == 0 {
			dto.Component = support.Component
			dto.Children = []port.UserMenuDTO{{
				Name:      catalog.Name,
				Icon:      catalog.Icon,
				Component: catalog.Component,
				Path:      "",
			}}
		} else {
			sort.Slice(children, func(i, j int) bool { return children[i].OrderNum < children[j].OrderNum })
			for _, child := range children {
				dto.Children = append(dto.Children, port.UserMenuDTO{Name: child.Name, Path: child.Path, Icon: child.Icon, Component: child.Component, Hidden: child.IsHidden == support.True})
			}
		}
		result = append(result, dto)
	}
	return result
}
