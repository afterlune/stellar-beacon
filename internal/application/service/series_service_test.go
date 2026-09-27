package service

import (
	"context"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type fakeSeriesRepository struct {
	items   []*port.Series
	record  entity.TSeries
	saved   entity.TSeries
	deleted []int
}

func (f *fakeSeriesRepository) ListPublic(context.Context) ([]*port.Series, error) {
	return f.items, nil
}
func (f *fakeSeriesRepository) ListAdmin(context.Context, port.SeriesFilter) ([]*port.Series, int64, error) {
	return f.items, int64(len(f.items)), nil
}
func (f *fakeSeriesRepository) ListOptions(context.Context) ([]*port.Series, error) {
	return f.items, nil
}
func (f *fakeSeriesRepository) Get(context.Context, int) (entity.TSeries, error) {
	if f.record.Id == 0 {
		return entity.TSeries{}, apperrors.NotFound("series.get")
	}
	return f.record, nil
}
func (f *fakeSeriesRepository) SaveOrUpdate(_ context.Context, series entity.TSeries) (entity.TSeries, error) {
	f.saved = series
	if series.Id == 0 {
		series.Id = 12
	}
	return series, nil
}
func (f *fakeSeriesRepository) Delete(_ context.Context, seriesID int) error {
	f.deleted = append(f.deleted, seriesID)
	return nil
}

type seriesArticles struct {
	fakeArticleRepository
	cards []*port.ArticleCard
}

func (f *seriesArticles) ListArticleCardsBySeries(context.Context, int) ([]*port.ArticleCard, error) {
	return f.cards, nil
}

func mustSeriesService(t *testing.T, repo port.SeriesRepository, articles port.ArticleRepository) *MySeriesService {
	t.Helper()
	svc, err := NewSeriesService(SeriesServiceDeps{Repo: repo, Articles: articles})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestSeriesPublicDetailReturnsOrderedArticles(t *testing.T) {
	repo := &fakeSeriesRepository{record: entity.TSeries{Id: 3, SeriesName: "渲染管线", Status: 1, ModerationStatus: "visible"}}
	articles := &seriesArticles{cards: []*port.ArticleCard{{Id: 1, ArticleTitle: "第一篇"}, {Id: 2, ArticleTitle: "第二篇"}}}
	svc := mustSeriesService(t, repo, articles)

	detail, err := svc.GetPublicSeries(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Series.SeriesName != "渲染管线" || detail.Series.ArticleCount != 2 {
		t.Fatalf("unexpected series payload: %+v", detail.Series)
	}
	if len(detail.Articles) != 2 || detail.Articles[0].Id != 1 {
		t.Fatalf("articles must keep the authored order: %+v", detail.Articles)
	}
}

func TestSeriesPublicDetailReportsHiddenSeries(t *testing.T) {
	repo := &fakeSeriesRepository{record: entity.TSeries{Id: 9, Status: 2, ModerationStatus: "visible"}}
	svc := mustSeriesService(t, repo, &seriesArticles{})
	_, err := svc.GetPublicSeries(context.Background(), 9)
	if !apperrors.IsKind(err, apperrors.KindNotFound) || apperrors.Op(err) != "series.public.visibility" {
		t.Fatalf("hidden series should be unavailable: %v", err)
	}
}

func TestSeriesSaveValidatesAndNormalizesInput(t *testing.T) {
	repo := &fakeSeriesRepository{}
	svc := mustSeriesService(t, repo, &seriesArticles{})
	if _, err := svc.SaveOrUpdateSeries(context.Background(), 7, SeriesSaveInput{Name: "  "}); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("an empty series name must fail validation: %v", err)
	}
	created, err := svc.SaveOrUpdateSeries(context.Background(), 7, SeriesSaveInput{Name: " 渲染管线 ", Description: " desc ", Cover: " cover "})
	if err != nil || created.Id != 12 {
		t.Fatalf("unexpected save result: series=%+v err=%v", created, err)
	}
	if repo.saved.UserId != 7 || repo.saved.SeriesName != "渲染管线" || repo.saved.SeriesDesc != "desc" || repo.saved.Cover != "cover" {
		t.Fatalf("series input must be normalized and owner-scoped: %+v", repo.saved)
	}
}

func TestSeriesDeleteRequiresIDsAndIgnoresAlreadyMissingRows(t *testing.T) {
	repo := &fakeSeriesRepository{}
	svc := mustSeriesService(t, repo, &seriesArticles{})
	if err := svc.DeleteSeries(context.Background(), 7, nil); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("empty ids must fail: %v", err)
	}
	if err := svc.DeleteSeries(context.Background(), 7, []int{4}); err != nil || len(repo.deleted) != 1 || repo.deleted[0] != 4 {
		t.Fatalf("missing row should preserve idempotent delete behavior: deleted=%v err=%v", repo.deleted, err)
	}
}

func TestSeriesServiceRejectsMissingDependencies(t *testing.T) {
	if _, err := NewSeriesService(SeriesServiceDeps{}); err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSeriesServiceRejectsNonOwnerMutations(t *testing.T) {
	repo := &fakeSeriesRepository{record: entity.TSeries{Id: 4, UserId: 2, SeriesName: "owned by another user", Status: 1}}
	svc := mustSeriesService(t, repo, &seriesArticles{})

	if _, err := svc.SaveOrUpdateSeries(context.Background(), 7, SeriesSaveInput{ID: 4, Name: "forbidden"}); !apperrors.IsKind(err, apperrors.KindForbidden) || repo.saved.Id != 0 {
		t.Fatalf("non-owner series save must be forbidden: err=%v saved=%+v", err, repo.saved)
	}
	if err := svc.DeleteSeries(context.Background(), 7, []int{4}); !apperrors.IsKind(err, apperrors.KindForbidden) || len(repo.deleted) != 0 {
		t.Fatalf("non-owner series delete must be forbidden: err=%v deleted=%v", err, repo.deleted)
	}
}
