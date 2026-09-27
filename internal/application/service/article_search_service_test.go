package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
)

type recordingArticleSearchIndexer struct {
	upsertCalls    [][]port.ArticleSearch
	deleteCalls    [][]int
	reconcileCalls [][]port.ArticleSearch
	upsertErr      error
	deleteErr      error
	reconcileErr   error
}

func (f *recordingArticleSearchIndexer) Upsert(_ context.Context, documents []port.ArticleSearch) error {
	f.upsertCalls = append(f.upsertCalls, append([]port.ArticleSearch(nil), documents...))
	return f.upsertErr
}

func (f *recordingArticleSearchIndexer) Delete(_ context.Context, ids []int) error {
	f.deleteCalls = append(f.deleteCalls, append([]int(nil), ids...))
	return f.deleteErr
}

func (f *recordingArticleSearchIndexer) Reconcile(_ context.Context, documents []port.ArticleSearch) error {
	f.reconcileCalls = append(f.reconcileCalls, append([]port.ArticleSearch(nil), documents...))
	return f.reconcileErr
}

func TestArticleSearchServiceSyncsOnlyPublicDocuments(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{
		Id: 41, UserId: 7, ArticleTitle: "Public article", ArticleContent: "Searchable body",
		Status: 1, IsDelete: 0, ModerationStatus: "visible",
	}}
	indexer := &recordingArticleSearchIndexer{}
	search, err := NewArticleSearchService(repo, indexer)
	if err != nil {
		t.Fatal(err)
	}

	if err := search.Sync(context.Background(), 41, 41, 0, -1); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(indexer.upsertCalls) != 1 || len(indexer.upsertCalls[0]) != 1 || indexer.upsertCalls[0][0].Id != 41 {
		t.Fatalf("public article was not upserted once: %#v", indexer.upsertCalls)
	}
	if len(indexer.deleteCalls) != 0 {
		t.Fatalf("public article was unexpectedly deleted: %#v", indexer.deleteCalls)
	}

	repo.record.ModerationStatus = "hidden"
	if err := search.Sync(context.Background(), 41); err != nil {
		t.Fatalf("Sync(hidden) error = %v", err)
	}
	if len(indexer.deleteCalls) != 1 || len(indexer.deleteCalls[0]) != 1 || indexer.deleteCalls[0][0] != 41 {
		t.Fatalf("hidden article was not removed from the index: %#v", indexer.deleteCalls)
	}
	if len(indexer.upsertCalls) != 1 {
		t.Fatalf("hidden article was unexpectedly upserted: %#v", indexer.upsertCalls)
	}
}

func TestArticleSearchServiceReconcileUsesDatabaseSnapshot(t *testing.T) {
	repo := &fakeArticleRepository{record: entity.TArticle{
		Id: 52, UserId: 9, ArticleTitle: "Authoritative article", Status: 1, ModerationStatus: "visible",
	}}
	indexer := &recordingArticleSearchIndexer{}
	search, err := NewArticleSearchService(repo, indexer)
	if err != nil {
		t.Fatal(err)
	}

	if err := search.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if len(indexer.reconcileCalls) != 1 || len(indexer.reconcileCalls[0]) != 1 || indexer.reconcileCalls[0][0].Id != 52 {
		t.Fatalf("database snapshot was not passed to the indexer: %#v", indexer.reconcileCalls)
	}
}

func TestArticleSearchServiceReturnsIndexerFailures(t *testing.T) {
	indexErr := errors.New("index unavailable")
	repo := &fakeArticleRepository{record: entity.TArticle{
		Id: 63, UserId: 12, ArticleTitle: "Public article", Status: 1, ModerationStatus: "visible",
	}}
	indexer := &recordingArticleSearchIndexer{upsertErr: indexErr}
	search, err := NewArticleSearchService(repo, indexer)
	if err != nil {
		t.Fatal(err)
	}
	if err := search.Sync(context.Background(), 63); !errors.Is(err, indexErr) {
		t.Fatalf("Sync() error = %v, want wrapped index failure", err)
	}

	indexer.reconcileErr = indexErr
	if err := search.Reconcile(context.Background()); !errors.Is(err, indexErr) {
		t.Fatalf("Reconcile() error = %v, want index failure", err)
	}
}
