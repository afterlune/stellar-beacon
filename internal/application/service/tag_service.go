package service

import (
	"context"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type TagService interface {
	ListTags(context.Context) ([]*port.Tag, error)
	ListTopTenTags(context.Context) ([]*port.Tag, error)
	ListTagsAdmin(context.Context, int, int, string) ([]*port.TagAdmin, int64, error)
	ListTagsAdminBySearch(context.Context, string) ([]*port.TagAdmin, error)
	SaveOrUpdateTag(context.Context, entity.TTag) error
	DeleteTag(context.Context, int, []int) error
}

type MyTagService struct{ repo port.TagRepository }

func NewTagService(repo port.TagRepository) *MyTagService {
	return &MyTagService{repo: repo}
}

func (t *MyTagService) tagRepository() port.TagRepository {
	if t.repo != nil {
		return t.repo
	}
	return tagRepo
}

func (t *MyTagService) ListTags(ctx context.Context) ([]*port.Tag, error) {
	return t.tagRepository().List(ctx)
}

func (t *MyTagService) ListTopTenTags(ctx context.Context) ([]*port.Tag, error) {
	return t.tagRepository().ListTopTen(ctx)
}

func (t *MyTagService) ListTagsAdmin(ctx context.Context, current, size int, keywords string) ([]*port.TagAdmin, int64, error) {
	filter := port.TagFilter{Keywords: keywords}
	count, err := t.tagRepository().CountAdmin(ctx, filter)
	if err != nil || count == 0 {
		return []*port.TagAdmin{}, count, err
	}
	data, err := t.tagRepository().ListAdmin(ctx, current, size, filter)
	return data, count, err
}

func (t *MyTagService) ListTagsAdminBySearch(ctx context.Context, keywords string) ([]*port.TagAdmin, error) {
	return t.tagRepository().Search(ctx, keywords)
}

func (t *MyTagService) SaveOrUpdateTag(ctx context.Context, tag entity.TTag) error {
	return t.tagRepository().SaveOrUpdate(ctx, tag)
}

func (t *MyTagService) DeleteTag(ctx context.Context, userID int, ids []int) error {
	return t.tagRepository().Delete(ctx, userID, ids)
}

var _ TagService = (*MyTagService)(nil)
