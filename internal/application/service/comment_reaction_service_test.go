package service

import (
	"context"
	"testing"
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
	active, count, err := service.ToggleCommentReaction(context.Background(), 42, 7, true)
	if err != nil || !active || count != 1 {
		t.Fatalf("unexpected comment reaction: active=%t count=%d err=%v", active, count, err)
	}
}
