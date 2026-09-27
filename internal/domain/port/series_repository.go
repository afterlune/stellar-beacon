package port

import (
	"context"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
)

// Series is the application-facing read model for one ordered collection.
type Series struct {
	Id               int        `json:"id"`
	UserId           int        `json:"userId"`
	SeriesName       string     `json:"seriesName"`
	SeriesDesc       string     `json:"seriesDesc"`
	Cover            string     `json:"cover"`
	Status           int        `json:"status"`
	ModerationStatus string     `json:"moderationStatus"`
	ModerationReason string     `json:"moderationReason,omitempty"`
	ModeratedBy      int        `json:"moderatedBy,omitempty"`
	ModeratedAt      *time.Time `json:"moderatedAt,omitempty"`
	AuthorHandle     string     `json:"authorHandle"`
	AuthorNickname   string     `json:"authorNickname"`
	AuthorAvatar     string     `json:"authorAvatar"`
	ArticleCount     int        `json:"articleCount"`
	UpdateTime       time.Time  `json:"updateTime"`
}

type SeriesFilter struct {
	Current          int
	Size             int
	Keywords         string
	Status           int
	ModerationStatus string
}

// SeriesRepository stores article collections. Soft-deleted series keep their
// name reserved so a later re-creation does not silently adopt old articles.
type SeriesRepository interface {
	ListPublic(ctx context.Context) ([]*Series, error)
	ListAdmin(ctx context.Context, filter SeriesFilter) ([]*Series, int64, error)
	ListOptions(ctx context.Context) ([]*Series, error)
	Get(ctx context.Context, seriesID int) (entity.TSeries, error)
	SaveOrUpdate(ctx context.Context, series entity.TSeries) (entity.TSeries, error)
	Delete(ctx context.Context, seriesID int) error
}
