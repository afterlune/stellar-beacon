package port

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// Series is the application-facing read model for one ordered collection.
type Series struct {
	Id           int       `json:"id"`
	SeriesName   string    `json:"seriesName"`
	SeriesDesc   string    `json:"seriesDesc"`
	Cover        string    `json:"cover"`
	ArticleCount int       `json:"articleCount"`
	UpdateTime   time.Time `json:"updateTime"`
}

// SeriesRepository stores article collections. Soft-deleted series keep their
// name reserved so a later re-creation does not silently adopt old articles.
type SeriesRepository interface {
	ListPublic(ctx context.Context) ([]*Series, error)
	ListAdmin(ctx context.Context, current, size int, keywords string) ([]*Series, int64, error)
	ListOptions(ctx context.Context) ([]*Series, error)
	Get(ctx context.Context, seriesID int) (entity.TSeries, error)
	SaveOrUpdate(ctx context.Context, series entity.TSeries) (entity.TSeries, error)
	Delete(ctx context.Context, seriesID int) error
}
