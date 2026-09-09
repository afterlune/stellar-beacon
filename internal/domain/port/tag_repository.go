package port

import (
	"benetnasch/internal/domain/entity"
	"context"
)

type TagRepository interface {
	List(ctx context.Context) ([]*Tag, error)
	ListTopTen(ctx context.Context) ([]*Tag, error)
	ListNamesByArticleID(ctx context.Context, articleID int) ([]string, error)
	CountAdmin(ctx context.Context, filter TagFilter) (int64, error)
	ListAdmin(ctx context.Context, current, size int, filter TagFilter) ([]*TagAdmin, error)
	Search(ctx context.Context, keywords string) ([]*TagAdmin, error)
	SaveOrUpdate(ctx context.Context, tag entity.TTag) error
	Delete(ctx context.Context, ids []int) error
}
