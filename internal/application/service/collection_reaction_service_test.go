package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
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

func collectionReactionContext(method, target, body string, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		c.Set("userInfo", model.UserDetailsDTO{Id: userID, UserInfoId: userID})
	}
	return c, recorder
}

func TestCollectionReactionServiceUsesExplicitState(t *testing.T) {
	repo := &fakeCollectionReactionRepository{}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodPut, "/auth/me/collection-reactions", `{"collectionId":1,"active":true}`, 7)
	result := service.ToggleCollectionReaction(c)
	if !result.Flag || !repo.active || repo.counts.LikeCount != 1 {
		t.Fatalf("unexpected like result: result=%+v repo=%+v", result, repo)
	}
	data, ok := result.Data.(model.CollectionReactionToggleDTO)
	if !ok || !data.Active || data.LikeCount != 1 {
		t.Fatalf("unexpected reaction response: %#v", result.Data)
	}
}

func TestCollectionReactionServiceReturnsCurrentState(t *testing.T) {
	repo := &fakeCollectionReactionRepository{states: map[int]port.CollectionReactionState{9: {Like: true, Favorite: true}}}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodGet, "/auth/me/collection-reactions/state?collectionId=9", "", 7)
	result := service.GetCollectionReactionState(c)
	data, ok := result.Data.(model.CollectionReactionStateDTO)
	if !result.Flag || !ok || data.CollectionId != 9 || !data.Like || !data.Favorite {
		t.Fatalf("unexpected state response: result=%+v data=%#v", result, result.Data)
	}
}

func TestCollectionReactionServiceListsFavorites(t *testing.T) {
	repo := &fakeCollectionReactionRepository{favorites: []int{9}}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo, Collections: fakeCollectionBatchReader{}})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodGet, "/auth/me/collection-reactions?reaction=favorite&current=1&size=10", "", 7)
	result := service.ListMyCollectionReactions(c)
	page, ok := result.Data.(model.PageResultDTO)
	if !result.Flag || !ok || page.Count != 1 || len(page.Records.([]*port.CollectionSummary)) != 1 {
		t.Fatalf("unexpected favorites response: result=%+v data=%#v", result, result.Data)
	}
}
