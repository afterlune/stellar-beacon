package service

import (
	"context"
	"errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"testing"
)

type fakeErrorLogRepository struct{ err error }

func (f *fakeErrorLogRepository) List(context.Context, int, int, string) ([]entity.TExceptionLog, int64, error) {
	return nil, 0, f.err
}
func (f *fakeErrorLogRepository) Delete(context.Context, []int) error { return f.err }

func TestErrorLogServiceReturnsUnavailableForRepositoryFailure(t *testing.T) {
	service := NewErrorLogService(&fakeErrorLogRepository{err: apperrors.Unavailable("error_log.list", context.Canceled)})
	_, _, err := service.ListErrorLogs(context.Background(), 1, 10, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected unavailable repository error, got %v", err)
	}
}

type fakeFriendLinkRepository struct{ links []entity.TFriendLink }

func (f *fakeFriendLinkRepository) ListPublic(context.Context) ([]entity.TFriendLink, error) {
	return f.links, nil
}
func (f *fakeFriendLinkRepository) ListAdmin(context.Context, int, int, string) ([]entity.TFriendLink, int64, error) {
	return f.links, int64(len(f.links)), nil
}
func (f *fakeFriendLinkRepository) SaveOrUpdate(context.Context, entity.TFriendLink) error {
	return nil
}
func (f *fakeFriendLinkRepository) Delete(context.Context, []int) error { return nil }

func (f *fakeFriendLinkRepository) CreateApplication(_ context.Context, link entity.TFriendLink) (int, error) {
	f.links = append(f.links, link)
	return len(f.links), nil
}
func (f *fakeFriendLinkRepository) Review(context.Context, []int, int) error { return nil }
func (f *fakeFriendLinkRepository) AddressExists(context.Context, string) (bool, error) {
	return false, nil
}

func TestFriendLinkServiceMapsDomainRecordsToPublicDTO(t *testing.T) {
	service := NewFriendLinkService(&fakeFriendLinkRepository{links: []entity.TFriendLink{{Id: 7, LinkName: "example"}}})
	result := service.ListFriendLinks()
	links, ok := result.Data.([]model.FriendLinkDTO)
	if !ok || len(links) != 1 || links[0].Id != 7 || links[0].LinkName != "example" {
		t.Fatalf("unexpected links: %#v", result.Data)
	}
}

type fakeMenuRepository struct{ menus []entity.TMenu }

func (f *fakeMenuRepository) List(context.Context, string) ([]entity.TMenu, error) {
	return f.menus, nil
}
func (f *fakeMenuRepository) ListOptions(context.Context) ([]entity.TMenu, error) {
	return f.menus, nil
}
func (f *fakeMenuRepository) ListByUserInfoID(context.Context, int) ([]entity.TMenu, error) {
	return f.menus, nil
}
func (f *fakeMenuRepository) SaveOrUpdate(context.Context, entity.TMenu) error { return nil }
func (f *fakeMenuRepository) UpdateHidden(context.Context, int, int) error     { return nil }
func (f *fakeMenuRepository) Delete(context.Context, int) error                { return nil }

func TestMenuServiceBuildsStableTreeFromPortRecords(t *testing.T) {
	service := NewMenuService(&fakeMenuRepository{menus: []entity.TMenu{
		{Id: 2, Name: "child", ParentId: 1, OrderNum: 2},
		{Id: 1, Name: "root", ParentId: 0, OrderNum: 1},
	}})
	menus, err := service.ListMenus(context.Background(), "")
	if err != nil || len(menus) != 2 || menus[0].Id != 2 || menus[1].Id != 1 {
		t.Fatalf("unexpected typed menu records: menus=%+v err=%v", menus, err)
	}
}

func TestMenuServicePreservesUserMenuPaths(t *testing.T) {
	service := NewMenuService(&fakeMenuRepository{menus: []entity.TMenu{
		{Id: 1, Name: "home", Path: "/", Component: "/home/Home.vue", ParentId: 0, OrderNum: 1},
		{Id: 2, Name: "articles", Path: "/article-submenu", Component: "Layout", ParentId: 0, OrderNum: 2},
		{Id: 3, Name: "article list", Path: "/article-list", Component: "/article/ArticleList.vue", ParentId: 2, OrderNum: 1},
	}})

	menus, err := service.ListUserMenus(context.Background(), 1)
	if err != nil || len(menus) != 3 || menus[0].Path != "/" || menus[2].Path != "/article-list" {
		t.Fatalf("user menu route records were changed: menus=%+v err=%v", menus, err)
	}
}

type fakeRoleRepository struct{ existing entity.TRole }

func (f *fakeRoleRepository) ListUserRoles(context.Context) ([]entity.TRole, error) { return nil, nil }
func (f *fakeRoleRepository) Count(context.Context, string) (int64, error)          { return 0, nil }
func (f *fakeRoleRepository) List(context.Context, int, int, string) ([]port.RoleView, error) {
	return nil, nil
}
func (f *fakeRoleRepository) FindByName(context.Context, string) (entity.TRole, error) {
	return f.existing, nil
}
func (f *fakeRoleRepository) SaveOrUpdate(context.Context, entity.TRole, []int, []int) error {
	return nil
}
func (f *fakeRoleRepository) Delete(context.Context, []int) error { return nil }
func (f *fakeRoleRepository) ListResourceRoles(context.Context) ([]port.ResourceRoleView, error) {
	return nil, nil
}
func (f *fakeRoleRepository) ListRolesByUserInfoID(context.Context, int) ([]string, error) {
	return nil, nil
}

func TestRoleServiceRejectsDuplicateRoleName(t *testing.T) {
	service := NewRoleService(&fakeRoleRepository{existing: entity.TRole{Id: 3, RoleName: "admin"}})
	err := service.SaveOrUpdateRole(context.Background(), 4, "admin", nil, nil)
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("expected duplicate role conflict, got %v", err)
	}
}
