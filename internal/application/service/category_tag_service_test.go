package service

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"testing"
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

func TestCategoryServicePreservesConflictMessage(t *testing.T) {
	err := NewCategoryService(&fakeCategoryRepository{err: apperrors.Conflict("category.save", "duplicate")}).SaveOrUpdateCategory(context.Background(), entity.TCategory{UserId: 7, CategoryName: "test"})
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("expected category conflict, got %v", err)
	}
}

func TestTagServiceMapsRepositoryFailure(t *testing.T) {
	_, _, err := NewTagService(&fakeTagRepository{err: apperrors.Unavailable("tag.count", testServiceError("secret"))}).ListTagsAdmin(context.Background(), 1, 10, "")
	if !apperrors.IsKind(err, apperrors.KindUnavailable) {
		t.Fatalf("expected unavailable repository error, got %v", err)
	}
}

func TestCategoryAndTagWritesUseCurrentOwner(t *testing.T) {
	categoryRepo := &fakeCategoryRepository{}
	err := NewCategoryService(categoryRepo).SaveOrUpdateCategory(context.Background(), entity.TCategory{UserId: 7, CategoryName: "私人分类"})
	if err != nil || categoryRepo.saved.UserId != 7 {
		t.Fatalf("category write must use the authenticated owner: err=%v saved=%+v", err, categoryRepo.saved)
	}

	tagRepo := &fakeTagRepository{}
	err = NewTagService(tagRepo).SaveOrUpdateTag(context.Background(), entity.TTag{UserId: 7, TagName: "私人标签"})
	if err != nil || tagRepo.saved.UserId != 7 {
		t.Fatalf("tag write must use the authenticated owner: err=%v saved=%+v", err, tagRepo.saved)
	}
}

func TestCategoryAndTagDeletePassOwnerScope(t *testing.T) {
	categoryRepo := &fakeCategoryRepository{}
	err := NewCategoryService(categoryRepo).DeleteCategories(context.Background(), 7, []int{3, 4})
	if err != nil || categoryRepo.deleteUser != 7 || len(categoryRepo.deleteIDs) != 2 {
		t.Fatalf("category delete must pass owner scope: err=%v user=%d ids=%v", err, categoryRepo.deleteUser, categoryRepo.deleteIDs)
	}

	tagRepo := &fakeTagRepository{}
	err = NewTagService(tagRepo).DeleteTag(context.Background(), 7, []int{5, 6})
	if err != nil || tagRepo.deleteUser != 7 || len(tagRepo.deleteIDs) != 2 {
		t.Fatalf("tag delete must pass owner scope: err=%v user=%d ids=%v", err, tagRepo.deleteUser, tagRepo.deleteIDs)
	}
}
