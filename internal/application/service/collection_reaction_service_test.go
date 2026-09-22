package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

type fakeCollectionReactionRepository struct {
	active    bool
	likeCount int
	states    map[int]bool
}

func (f *fakeCollectionReactionRepository) Set(_ context.Context, _ int, _ int, active bool) (bool, int, error) {
	f.active = active
	if active {
		f.likeCount = 1
	} else {
		f.likeCount = 0
	}
	return f.active, f.likeCount, nil
}

func (f *fakeCollectionReactionRepository) Counts(context.Context, []int) (map[int]int, error) {
	return map[int]int{1: f.likeCount}, nil
}

func (f *fakeCollectionReactionRepository) States(context.Context, int, []int) (map[int]bool, error) {
	return f.states, nil
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
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodPut, "/auth/me/collection-reactions", `{"collectionId":1,"active":true}`, 7)
	result := service.ToggleCollectionReaction(c)
	if !result.Flag || !repo.active || repo.likeCount != 1 {
		t.Fatalf("unexpected like result: result=%+v repo=%+v", result, repo)
	}
	data, ok := result.Data.(model.CollectionReactionToggleDTO)
	if !ok || !data.Active || data.LikeCount != 1 {
		t.Fatalf("unexpected reaction response: %#v", result.Data)
	}
}

func TestCollectionReactionServiceReturnsCurrentState(t *testing.T) {
	repo := &fakeCollectionReactionRepository{states: map[int]bool{9: true}}
	service, err := NewCollectionReactionService(CollectionReactionServiceDeps{Repo: repo})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodGet, "/auth/me/collection-reactions/state?collectionId=9", "", 7)
	result := service.GetCollectionReactionState(c)
	data, ok := result.Data.(model.CollectionReactionStateDTO)
	if !result.Flag || !ok || data.CollectionId != 9 || !data.Like {
		t.Fatalf("unexpected state response: result=%+v data=%#v", result, result.Data)
	}
}
