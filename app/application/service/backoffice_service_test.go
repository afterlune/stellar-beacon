package service

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeErrorLogRepository struct{ err error }

func (f *fakeErrorLogRepository) List(context.Context, int, int, string) ([]entity.TExceptionLog, int64, error) {
	return nil, 0, f.err
}
func (f *fakeErrorLogRepository) Delete(context.Context, []int) error { return f.err }

func TestErrorLogServiceReturnsUnavailableForRepositoryFailure(t *testing.T) {
	service := NewErrorLogService(&fakeErrorLogRepository{err: apperrors.Unavailable("error_log.list", context.Canceled)})
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/admin/exception/logs?current=1&size=10", nil)
	result := service.ListErrorLogs(c)
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected result: %+v", result)
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
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/admin/menus", nil)
	result := service.ListMenus(c)
	menus, ok := result.Data.([]model.MenuDTO)
	if !ok || len(menus) != 1 || menus[0].Id != 1 || len(menus[0].Children) != 1 || menus[0].Children[0].Id != 2 {
		t.Fatalf("unexpected menu tree: %#v", result.Data)
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
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/admin/role", strings.NewReader(`{"id":4,"roleName":"admin"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	result := service.SaveOrUpdateRole(c)
	if result.Flag || result.Message != "该角色存在" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJobLogStatusRejectsFractionalNumbers(t *testing.T) {
	if _, ok := jobLogStatus(float64(1.5)); ok {
		t.Fatal("fractional status must be rejected")
	}
}
