package port

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
)

type TalkRepository interface {
	Count(ctx context.Context, filter TalkFilter) (int, error)
	List(ctx context.Context, current, size int) ([]*Talk, error)
	Get(ctx context.Context, id int) (Talk, error)
	ListAdmin(ctx context.Context, current, size int, filter TalkFilter) ([]*TalkAdmin, error)
	GetAdmin(ctx context.Context, id int) (TalkAdmin, error)
	SaveOrUpdate(ctx context.Context, talk entity.TTalk) error
	Delete(ctx context.Context, ids []int) error
}
