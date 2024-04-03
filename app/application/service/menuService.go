package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/repository"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"sort"
	"strconv"
	"xorm.io/builder"
)

type MenuService interface {
	ListMenus(c *gin.Context) model.ResultVO
	SaveOrUpdateMenu(c *gin.Context) model.ResultVO
	UpdateMenuIsHidden(c *gin.Context) model.ResultVO
	DeleteMenu(c *gin.Context) model.ResultVO
	ListMenuOptions() model.ResultVO
	ListUserMenus(userInfoId int) model.ResultVO
	listCatalogs(menus []*entity.TMenu) []*entity.TMenu
	getMenuMap(menus []*entity.TMenu) map[int][]*entity.TMenu
	convertUserMenuList(catalogs []*entity.TMenu, hm map[int][]*entity.TMenu) []model.UserMenuDTO
}

type MyMenuSService struct{}

func (m *MyMenuSService) ListMenus(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	engine := ormInit.GetEngine()
	var menus []*entity.TMenu
	if vo.Keywords != "" {
		err = engine.Prepare().Where(builder.Like{"name", vo.Keywords}).Find(&menus)
	} else {
		err = engine.Prepare().Find(&menus)
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	catalogs := m.listCatalogs(menus)
	childrenMap := m.getMenuMap(menus)

	var dtos []model.MenuDTO
	for _, v := range catalogs {
		var dto model.MenuDTO
		marshal, err := json.Marshal(v)
		err = json.Unmarshal(marshal, &dto)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		var list []model.MenuDTO
		bytes, err := json.Marshal(childrenMap[v.Id])
		err = json.Unmarshal(bytes, &list)
		if err != nil {
			zlog.Error(err.Error())
			return model.ResultFail()
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].OrderNum < list[j].OrderNum
		})
		dto.Children = list
		delete(childrenMap, v.Id)
		dtos = append(dtos, dto)
	}
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].OrderNum < dtos[j].OrderNum
	})
	if len(childrenMap) != 0 {
		var childrenList []*entity.TMenu
		for _, v := range childrenMap {
			childrenList = append(childrenList, v...)
		}
		var childrenDTOList []model.MenuDTO
		for _, v := range childrenList {
			var menuDTO model.MenuDTO
			marshal, err := json.Marshal(v)
			err = json.Unmarshal(marshal, &menuDTO)
			if err != nil {
				zlog.Error(err.Error())
				return model.ResultFail()
			}
			childrenDTOList = append(childrenDTOList, menuDTO)
		}
		sort.Slice(childrenDTOList, func(i, j int) bool {
			return childrenDTOList[i].OrderNum < childrenDTOList[j].OrderNum
		})
		dtos = append(dtos, childrenDTOList...)
	}
	return model.ResultOkWithData(dtos)
}

func (m *MyMenuSService) SaveOrUpdateMenu(c *gin.Context) model.ResultVO {
	var vo model.MenuVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
	}
	var menu entity.TMenu
	shared.StructCopy(vo, &menu)
	session := ormInit.GetEngine().NewSession()
	session.Begin()
	defer session.Close()
	if menu.Id != 0 {
		_, err = session.Prepare().ID(menu.Id).Update(&menu)
	} else {
		_, err = session.Prepare().Insert(&menu)
	}
	if err != nil {
		zlog.Error(err.Error())
		zlog.Unwrap(session.Rollback())
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func (m *MyMenuSService) UpdateMenuIsHidden(c *gin.Context) model.ResultVO {
	var vo model.IsHiddenVO
	zlog.Unwrap(c.ShouldBind(&vo))

	var menu entity.TMenu
	shared.StructCopy(vo, &menu)
	_, err := ormInit.GetEngine().ID(menu.Id).MustCols("is_hidden").Update(&menu)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (m *MyMenuSService) DeleteMenu(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("menuId"))
	engine := ormInit.GetEngine()
	count, err := engine.Prepare().Where(fmt.Sprintf("menu_id = %d", id)).Count(&entity.TRoleMenu{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if count > 0 {
		return model.ResultFailWithMessage("菜单下有角色关联")
	}
	var iDs []int
	err = engine.Prepare().Select("id").Where(fmt.Sprintf("parent_id = %d", id)).Find(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	iDs = append(iDs, id)
	_, err = engine.Prepare().In("id", iDs).Delete(&entity.TMenu{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (m *MyMenuSService) ListMenuOptions() model.ResultVO {
	engine := ormInit.GetEngine()
	var menus []*entity.TMenu
	err := engine.Select("id, name, parent_id, order_num").Find(&menus)
	if err != nil {
		zlog.Error(err.Error())
	}
	catalogs := m.listCatalogs(menus)
	childrenMap := m.getMenuMap(menus)
	var labelOptionDTOs []model.LabelOptionDTO
	for _, v := range catalogs {
		var dtos []model.LabelOptionDTO
		children := childrenMap[v.Id]
		if len(children) != 0 {
			sort.Slice(children, func(i, j int) bool {
				return children[i].OrderNum < children[j].OrderNum
			})
			for _, va := range children {
				dtos = append(dtos, model.LabelOptionDTO{
					Id:    va.Id,
					Label: va.Name,
				})
			}
		}
		labelOptionDTOs = append(labelOptionDTOs, model.LabelOptionDTO{
			Id:       v.Id,
			Label:    v.Name,
			Children: dtos,
		})
	}
	return model.ResultOkWithData(labelOptionDTOs)
}

func (m *MyMenuSService) ListUserMenus(userInfoId int) model.ResultVO {

	menus := repository.ListMenusByUserInfoId(userInfoId)
	catalogs := m.listCatalogs(menus)
	childrenMap := m.getMenuMap(menus)
	return model.ResultOkWithData(m.convertUserMenuList(catalogs, childrenMap))
}

func (m *MyMenuSService) listCatalogs(menus []*entity.TMenu) []*entity.TMenu {
	var mens []*entity.TMenu
	for _, item := range menus {
		if item.ParentId == 0 {
			mens = append(mens, item)
		}
	}
	sort.Slice(mens, func(i, j int) bool {
		return mens[i].OrderNum < mens[j].OrderNum
	})
	return mens
}

func (m *MyMenuSService) getMenuMap(menus []*entity.TMenu) map[int][]*entity.TMenu {
	hm := make(map[int][]*entity.TMenu)
	for _, item := range menus {
		if item.ParentId != 0 {
			hm[item.ParentId] = append(hm[item.ParentId], item)
		}
	}
	return hm
}

func (m *MyMenuSService) convertUserMenuList(catalogs []*entity.TMenu, hm map[int][]*entity.TMenu) []model.UserMenuDTO {
	var dtos []model.UserMenuDTO
	for i := 0; i < len(catalogs); i++ {
		var userMenuDTO model.UserMenuDTO
		var userMenuDTOs []model.UserMenuDTO
		children := hm[catalogs[i].Id]
		if len(children) != 0 {
			marshal, err := json.Marshal(catalogs[i])
			if err != nil {
				zlog.Error(err.Error())
			}
			err = json.Unmarshal(marshal, &userMenuDTO)
			if err != nil {
				zlog.Error(err.Error())
			}
			sort.Slice(children, func(i, j int) bool {
				return children[i].OrderNum < children[j].OrderNum
			})
			for i := 0; i < len(children); i++ {
				var dto model.UserMenuDTO
				bytes, err := json.Marshal(children[i])
				if err != nil {
					zlog.Error(err.Error())
				}
				err = json.Unmarshal(bytes, &dto)
				if err != nil {
					zlog.Error(err.Error())
				}
				dto.Hidden = children[i].IsHidden == shared.TRUE
				userMenuDTOs = append(userMenuDTOs, dto)
			}
		} else {
			userMenuDTO.Path = catalogs[i].Path
			userMenuDTO.Component = shared.COMPONENT
			userMenuDTOs = append(userMenuDTOs, model.UserMenuDTO{
				Path:      "",
				Name:      catalogs[i].Name,
				Icon:      catalogs[i].Icon,
				Component: catalogs[i].Component,
			})
		}
		userMenuDTO.Hidden = catalogs[i].IsHidden == shared.TRUE
		userMenuDTO.Children = userMenuDTOs
		dtos = append(dtos, userMenuDTO)
	}
	return dtos
}
