package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
)

type fakeCommentReactionRepository struct {
	active    bool
	likeCount int
}

func (f *fakeCommentReactionRepository) Set(_ context.Context, _ int, _ int, active bool) (bool, int, error) {
	f.active = active
	if active {
		f.likeCount = 1
	} else {
		f.likeCount = 0
	}
	return f.active, f.likeCount, nil
}

func TestCommentReactionServiceUsesExplicitState(t *testing.T) {
	repo := &fakeCommentReactionRepository{}
	service, err := NewCommentReactionService(CommentReactionServiceDeps{Repo: repo})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := collectionReactionContext(http.MethodPut, "/auth/me/comment-reactions", `{"commentId":42,"active":true}`, 7)
	result := service.ToggleCommentReaction(c)
	data, ok := result.Data.(model.CommentReactionToggleDTO)
	if !result.Flag || !ok || !data.Active || data.LikeCount != 1 {
		t.Fatalf("unexpected comment reaction: result=%+v data=%#v", result, result.Data)
	}
}
