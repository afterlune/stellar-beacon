package port

import "context"

type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	CountAdmin(ctx context.Context, filter CategoryFilter) (int64, error)
	ListAdmin(ctx context.Context, current, size int, filter CategoryFilter) ([]*CategoryAdmin, error)
	Search(ctx context.Context, keywords string) ([]CategoryOption, error)
	SaveOrUpdate(ctx context.Context, category TCategory) error
	Delete(ctx context.Context, ids []int) error
}
