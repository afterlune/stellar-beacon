package service

import (
	"context"
	"testing"

	"github.com/afterlune/stellar-beacon/internal/domain/port"
)

type fakeCollectionReactionRepository struct {
	active    bool
	reaction  string
	counts    port.CollectionReactionCounts
	states    map[int]port.CollectionReactionState
	favorites []int
}

func (f *fakeCollectionReactionRepository) Set(_ context.Context, _ int, _ int, reaction string, active bool) (bool, port.CollectionReactionCounts, error) {
	f.active = active
	f.reaction = reaction
	if active {
		if reaction == port.ReactionFavorite {
			f.counts.FavoriteCount = 1
		} else {
			f.counts.LikeCount = 1
		}
	} else {
		f.counts = port.CollectionReactionCounts{}
	}
	return f.active, f.counts, nil
}

func (f *fakeCollectionReactionRepository) Counts(context.Context, []int) (map[int]port.CollectionReactionCounts, error) {
	return map[int]port.CollectionReactionCounts{1: f.counts}, nil
}

func (f *fakeCollectionReactionRepository) States(context.Context, int, []int) (map[int]port.CollectionReactionState, error) {
	return f.states, nil
}

func (f *fakeCollectionReactionRepository) ListFavoriteCollectionIDsByUser(context.Context, int, int, int) ([]int, int, error) {
	return f.favorites, len(f.favorites), nil
}

type fakeCollectionBatchReader struct{}

func (fakeCollectionBatchReader) ListPublicByIDs(context.Context, []int) ([]*port.CollectionSummary, error) {
	return []*port.CollectionSummary{{ID: 9, Slug: "saved-list", Title: "Saved"}}, nil
}

func TestCollectionReactionServiceUsesExplicitState(t *testing.T) {
	repo := &fakeCollectionReactionRepository{}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ToggleCollectionReaction(context.Background(), 7, 1, "", true)
	if err != nil || !repo.active || repo.reaction != port.ReactionLike || repo.counts.LikeCount != 1 || !result.Active || result.LikeCount != 1 {
		t.Fatalf("unexpected like result: result=%+v repo=%+v err=%v", result, repo, err)
	}
}

func TestCollectionReactionServiceReturnsCurrentState(t *testing.T) {
	repo := &fakeCollectionReactionRepository{states: map[int]port.CollectionReactionState{9: {Like: true, Favorite: true}}}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.GetCollectionReactionState(context.Background(), 7, 9)
	if err != nil || !state.Like || !state.Favorite {
		t.Fatalf("unexpected state response: state=%#v err=%v", state, err)
	}
}

func TestCollectionReactionServiceListsFavorites(t *testing.T) {
	repo := &fakeCollectionReactionRepository{favorites: []int{9}}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	records, count, err := service.ListMyCollectionReactions(context.Background(), 7, 1, 10, port.ReactionFavorite)
	if err != nil || count != 1 || len(records) != 1 {
		t.Fatalf("unexpected favorites response: records=%#v count=%d err=%v", records, count, err)
	}
}
