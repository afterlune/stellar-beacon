package service

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCategoryRepository struct {
	err        error
	saved      entity.TCategory
	deleteUser int
	deleteIDs  []int
}

func (f *fakeCategoryRepository) List(context.Context) ([]port.Category, error) { return nil, f.err }
func (f *fakeCategoryRepository) CountAdmin(context.Context, port.CategoryFilter) (int64, error) {
	return 0, f.err
}
func (f *fakeCategoryRepository) ListAdmin(context.Context, int, int, port.CategoryFilter) ([]*port.CategoryAdmin, error) {
	return nil, nil
}
func (f *fakeCategoryRepository) Search(context.Context, string) ([]port.CategoryOption, error) {
	return nil, nil
}
func (f *fakeCategoryRepository) SaveOrUpdate(_ context.Context, category entity.TCategory) error {
	f.saved = category
	return f.err
}
func (f *fakeCategoryRepository) Delete(_ context.Context, userID int, ids []int) error {
	f.deleteUser = userID
	f.deleteIDs = append([]int(nil), ids...)
	return f.err
}

type fakeTagRepository struct {
	err        error
	saved      entity.TTag
	deleteUser int
	deleteIDs  []int
}

func (f *fakeTagRepository) List(context.Context) ([]*port.Tag, error) { return nil, f.err }
func (f *fakeTagRepository) ListTopTen(context.Context) ([]*port.Tag, error) {
	return nil, f.err
}
func (f *fakeTagRepository) ListNamesByArticleID(context.Context, int) ([]string, error) {
	return nil, nil
}
func (f *fakeTagRepository) CountAdmin(context.Context, port.TagFilter) (int64, error) {
	return 0, f.err
}
func (f *fakeTagRepository) ListAdmin(context.Context, int, int, port.TagFilter) ([]*port.TagAdmin, error) {
	return nil, nil
}
func (f *fakeTagRepository) Search(context.Context, string) ([]*port.TagAdmin, error) {
	return nil, nil
}
func (f *fakeTagRepository) SaveOrUpdate(_ context.Context, tag entity.TTag) error {
	f.saved = tag
	return f.err
}
func (f *fakeTagRepository) Delete(_ context.Context, userID int, ids []int) error {
	f.deleteUser = userID
	f.deleteIDs = append([]int(nil), ids...)
	return f.err
}

func categoryTagTestContext(method, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, path, nil)
	return c
}

func TestCategoryServicePreservesConflictMessage(t *testing.T) {
	c := categoryTagTestContext(http.MethodGet, "/admin/categories?id=0&categoryName=test")
	c.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	result := NewCategoryService(&fakeCategoryRepository{err: apperrors.Conflict("category.save", "duplicate")}).SaveOrUpdateCategory(c)
	if result.Flag || result.Message != "分类名已存在" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTagServiceMapsRepositoryFailure(t *testing.T) {
	c := categoryTagTestContext(http.MethodGet, "/admin/tags?current=1&size=10")
	result := NewTagService(&fakeTagRepository{err: apperrors.Unavailable("tag.count", testServiceError("secret"))}).ListTagsAdmin(c)
	if result.Flag || result.Message != "系统繁忙，请稍后再试" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func categoryTagJSONContext(method, path, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestCategoryAndTagWritesUseCurrentOwner(t *testing.T) {
	categoryRepo := &fakeCategoryRepository{}
	categoryCtx := categoryTagJSONContext(http.MethodPost, "/v1/admin/categories", `{"categoryName":"私人分类"}`)
	categoryCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := NewCategoryService(categoryRepo).SaveOrUpdateCategory(categoryCtx); !result.Flag || categoryRepo.saved.UserId != 7 {
		t.Fatalf("category write must use the authenticated owner: result=%+v saved=%+v", result, categoryRepo.saved)
	}

	tagRepo := &fakeTagRepository{}
	tagCtx := categoryTagJSONContext(http.MethodPost, "/v1/admin/tags", `{"tagName":"私人标签"}`)
	tagCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := NewTagService(tagRepo).SaveOrUpdateTag(tagCtx); !result.Flag || tagRepo.saved.UserId != 7 {
		t.Fatalf("tag write must use the authenticated owner: result=%+v saved=%+v", result, tagRepo.saved)
	}
}

func TestCategoryAndTagDeletePassOwnerScope(t *testing.T) {
	categoryRepo := &fakeCategoryRepository{}
	categoryCtx := categoryTagJSONContext(http.MethodDelete, "/v1/admin/categories", `[3,4]`)
	categoryCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := NewCategoryService(categoryRepo).DeleteCategories(categoryCtx); !result.Flag || categoryRepo.deleteUser != 7 || len(categoryRepo.deleteIDs) != 2 {
		t.Fatalf("category delete must pass owner scope: result=%+v user=%d ids=%v", result, categoryRepo.deleteUser, categoryRepo.deleteIDs)
	}

	tagRepo := &fakeTagRepository{}
	tagCtx := categoryTagJSONContext(http.MethodDelete, "/v1/admin/tags", `[5,6]`)
	tagCtx.Set("userInfo", model.UserDetailsDTO{UserInfoId: 7})
	if result := NewTagService(tagRepo).DeleteTag(tagCtx); !result.Flag || tagRepo.deleteUser != 7 || len(tagRepo.deleteIDs) != 2 {
		t.Fatalf("tag delete must pass owner scope: result=%+v user=%d ids=%v", result, tagRepo.deleteUser, tagRepo.deleteIDs)
	}
}
