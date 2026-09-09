package service

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCategoryRepository struct {
	err error
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
func (f *fakeCategoryRepository) SaveOrUpdate(context.Context, entity.TCategory) error { return f.err }
func (f *fakeCategoryRepository) Delete(context.Context, []int) error                  { return f.err }

type fakeTagRepository struct {
	err error
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
func (f *fakeTagRepository) SaveOrUpdate(context.Context, entity.TTag) error { return f.err }
func (f *fakeTagRepository) Delete(context.Context, []int) error             { return f.err }

func categoryTagTestContext(method, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, path, nil)
	return c
}

func TestCategoryServicePreservesConflictMessage(t *testing.T) {
	c := categoryTagTestContext(http.MethodGet, "/admin/categories?id=0&categoryName=test")
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
