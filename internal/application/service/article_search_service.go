package service

import (
	"context"
	"errors"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

// ArticleSearchMaintainer is the application-facing seam used by article,
// moderation and scheduled-publish flows to keep search documents current.
type ArticleSearchMaintainer interface {
	Sync(context.Context, ...int) error
	Reconcile(context.Context) error
}

type MyArticleSearchService struct {
	repo    port.ArticleRepository
	indexer port.ArticleSearchIndexer
}

func NewArticleSearchService(repo port.ArticleRepository, indexer port.ArticleSearchIndexer) (*MyArticleSearchService, error) {
	if repo == nil || indexer == nil {
		return nil, errors.New("article search service dependencies are incomplete")
	}
	return &MyArticleSearchService{repo: repo, indexer: indexer}, nil
}

// Sync reads the authoritative state after a database mutation. A document
// that is missing, private, recycled, scheduled or hidden is removed from the
// public index instead of leaving stale searchable content behind.
func (s *MyArticleSearchService) Sync(ctx context.Context, articleIDs ...int) error {
	if s == nil || s.repo == nil || s.indexer == nil || len(articleIDs) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(articleIDs))
	upserts := make([]port.ArticleSearch, 0, len(articleIDs))
	deletes := make([]int, 0, len(articleIDs))
	var syncErr error
	for _, articleID := range articleIDs {
		if articleID <= 0 {
			continue
		}
		if _, ok := seen[articleID]; ok {
			continue
		}
		seen[articleID] = struct{}{}
		document, public, err := s.repo.GetArticleSearchDocument(ctx, articleID)
		if err != nil {
			syncErr = errors.Join(syncErr, err)
			continue
		}
		if public {
			upserts = append(upserts, document)
		} else {
			deletes = append(deletes, articleID)
		}
	}
	if len(deletes) > 0 {
		syncErr = errors.Join(syncErr, s.indexer.Delete(ctx, deletes))
	}
	if len(upserts) > 0 {
		syncErr = errors.Join(syncErr, s.indexer.Upsert(ctx, upserts))
	}
	return syncErr
}

// Reconcile compares the complete public database snapshot with the index and
// removes stale documents through the provider's atomic reconciliation path.
func (s *MyArticleSearchService) Reconcile(ctx context.Context) error {
	if s == nil || s.repo == nil || s.indexer == nil {
		return nil
	}
	documents, err := s.repo.ListPublicArticleSearchDocuments(ctx)
	if err != nil {
		return err
	}
	return s.indexer.Reconcile(ctx, documents)
}

var _ ArticleSearchMaintainer = (*MyArticleSearchService)(nil)
