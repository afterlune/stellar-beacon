package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type MenuService interface {
	ListMenus(ctx context.Context, keywords string) ([]entity.TMenu, error)
	SaveOrUpdateMenu(ctx context.Context, menu entity.TMenu) error
	UpdateMenuIsHidden(ctx context.Context, id, isHidden int) error
	DeleteMenu(ctx context.Context, id int) error
	ListMenuOptions(ctx context.Context) ([]entity.TMenu, error)
	ListUserMenus(ctx context.Context, userInfoID int) ([]entity.TMenu, error)
}

type MyMenuSService struct{ repo port.MenuRepository }

func NewMenuService(repo port.MenuRepository) *MyMenuSService { return &MyMenuSService{repo: repo} }

func (m *MyMenuSService) menuRepository() port.MenuRepository {
	if m.repo != nil {
		return m.repo
	}
	return menuRepo
}

func (m *MyMenuSService) ListMenus(ctx context.Context, keywords string) ([]entity.TMenu, error) {
	return m.menuRepository().List(ctx, keywords)
}

func (m *MyMenuSService) SaveOrUpdateMenu(ctx context.Context, menu entity.TMenu) error {
	return m.menuRepository().SaveOrUpdate(ctx, menu)
}

func (m *MyMenuSService) UpdateMenuIsHidden(ctx context.Context, id, isHidden int) error {
	return m.menuRepository().UpdateHidden(ctx, id, isHidden)
}

func (m *MyMenuSService) DeleteMenu(ctx context.Context, id int) error {
	return m.menuRepository().Delete(ctx, id)
}

func (m *MyMenuSService) ListMenuOptions(ctx context.Context) ([]entity.TMenu, error) {
	return m.menuRepository().ListOptions(ctx)
}

func (m *MyMenuSService) ListUserMenus(ctx context.Context, userInfoID int) ([]entity.TMenu, error) {
	return m.menuRepository().ListByUserInfoID(ctx, userInfoID)
}
