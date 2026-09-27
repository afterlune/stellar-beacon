package service

import (
	"context"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

// SeriesService owns ordered article collections: public reads plus the admin
// CRUD that backs the series management page.
type SeriesService interface {
	ListPublicSeries(ctx context.Context) ([]*port.Series, error)
	GetPublicSeries(ctx context.Context, seriesID int) (SeriesDetail, error)
	ListAdminSeries(ctx context.Context, filter port.SeriesFilter) ([]*port.Series, int64, error)
	ListSeriesOptions(ctx context.Context) ([]*port.Series, error)
	SaveOrUpdateSeries(ctx context.Context, userID int, input SeriesSaveInput) (entity.TSeries, error)
	DeleteSeries(ctx context.Context, userID int, ids []int) error
}

type MySeriesService struct {
	repo     port.SeriesRepository
	articles port.ArticleRepository
}

func NewSeriesService(deps SeriesServiceDeps) (*MySeriesService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MySeriesService{repo: deps.Repo, articles: deps.Articles}, nil
}

type SeriesDetail struct {
	Series   port.Series         `json:"series"`
	Articles []*port.ArticleCard `json:"articles"`
}

type SeriesSaveInput struct {
	ID          int
	Name        string
	Description string
	Cover       string
}

func (s *MySeriesService) ListPublicSeries(ctx context.Context) ([]*port.Series, error) {
	return s.repo.ListPublic(ctx)
}

func (s *MySeriesService) GetPublicSeries(ctx context.Context, seriesID int) (SeriesDetail, error) {
	series, err := s.repo.Get(ctx, seriesID)
	if err != nil {
		return SeriesDetail{}, err
	}
	if series.Status != 1 || series.ModerationStatus == "hidden" {
		return SeriesDetail{}, apperrors.NotFound("series.public.visibility")
	}
	articles, err := s.articles.ListArticleCardsBySeries(ctx, seriesID)
	if err != nil {
		return SeriesDetail{}, err
	}
	return SeriesDetail{
		Series:   port.Series{Id: series.Id, SeriesName: series.SeriesName, SeriesDesc: series.SeriesDesc, Cover: series.Cover, ArticleCount: len(articles)},
		Articles: articles,
	}, nil
}

func (s *MySeriesService) ListAdminSeries(ctx context.Context, filter port.SeriesFilter) ([]*port.Series, int64, error) {
	return s.repo.ListAdmin(ctx, filter)
}

func (s *MySeriesService) ListSeriesOptions(ctx context.Context) ([]*port.Series, error) {
	return s.repo.ListOptions(ctx)
}

func (s *MySeriesService) SaveOrUpdateSeries(ctx context.Context, userID int, input SeriesSaveInput) (entity.TSeries, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 50 {
		return entity.TSeries{}, apperrors.Invalid("series.save", "series name is required")
	}
	if len([]rune(input.Description)) > 255 {
		return entity.TSeries{}, apperrors.Invalid("series.save", "series description is too long")
	}
	if input.ID != 0 {
		existing, err := s.repo.Get(ctx, input.ID)
		if err != nil {
			return entity.TSeries{}, err
		}
		if existing.UserId != userID {
			return entity.TSeries{}, apperrors.New(apperrors.KindForbidden, "series.save", nil)
		}
	}
	return s.repo.SaveOrUpdate(ctx, entity.TSeries{
		Id: input.ID, UserId: userID, SeriesName: name, SeriesDesc: strings.TrimSpace(input.Description),
		Cover: strings.TrimSpace(input.Cover), Status: 1,
	})
}

func (s *MySeriesService) DeleteSeries(ctx context.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return apperrors.Invalid("series.delete", "series id is required")
	}
	for _, id := range ids {
		existing, err := s.repo.Get(ctx, id)
		if err != nil {
			if apperrors.IsKind(err, apperrors.KindNotFound) {
				continue
			}
			return err
		}
		if existing.UserId != userID {
			return apperrors.New(apperrors.KindForbidden, "series.delete", nil)
		}
	}
	for _, id := range ids {
		if err := s.repo.Delete(ctx, id); err != nil && !apperrors.IsKind(err, apperrors.KindNotFound) {
			return err
		}
	}
	return nil
}

var _ SeriesService = (*MySeriesService)(nil)
