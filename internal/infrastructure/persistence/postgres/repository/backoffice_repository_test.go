package repository

import (
	"context"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"testing"
)

func TestBackofficeRepositoriesRequireInjectedEngine(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		call func() error
	}{
		{name: "site", call: func() error { _, err := NewSiteInfoRepo(nil).CountArticles(ctx); return err }},
		{name: "friend link", call: func() error { _, err := NewFriendLinkRepo(nil).ListPublic(ctx); return err }},
		{name: "job", call: func() error { _, err := NewJobRepository(nil).Get(ctx, 1); return err }},
		{name: "job log", call: func() error { _, _, err := NewJobLogRepo(nil).List(ctx, 1, 10, structJobLogFilter()); return err }},
		{name: "error log", call: func() error { _, _, err := NewErrorLogRepo(nil).List(ctx, 1, 10, ""); return err }},
		{name: "operation log", call: func() error { _, _, err := NewOperationLogRepo(nil).List(ctx, 1, 10, ""); return err }},
		{name: "menu", call: func() error { _, err := NewMenuRepo(nil).List(ctx, ""); return err }},
		{name: "resource", call: func() error { _, err := NewResourceRepo(nil).List(ctx, ""); return err }},
		{name: "role", call: func() error { _, err := NewRoleRepository(nil).ListUserRoles(ctx); return err }},
		{name: "photo album", call: func() error { _, err := NewPhotoAlbumRepository(nil).ListPublic(ctx); return err }},
		{name: "photo", call: func() error { _, _, err := NewPhotoRepository(nil).List(ctx, 1, 10, 0, 0, ""); return err }},
		{name: "user info", call: func() error { _, err := NewUserInfoRepo(nil).GetByID(ctx, 1); return err }},
		{name: "article", call: func() error { _, _, err := NewArticleRepo(nil).ListArchives(ctx, 1, 10); return err }},
		{name: "article reaction", call: func() error { _, err := NewArticleReactionRepo(nil).Counts(ctx, []int{1}); return err }},
		{name: "series", call: func() error { _, err := NewSeriesRepo(nil).ListPublic(ctx); return err }},
		{name: "category", call: func() error { _, err := NewCategoryRepo(nil).List(ctx); return err }},
		{name: "comment", call: func() error { _, err := NewCommentRepo(nil).ListTopSixComments(ctx); return err }},
		{name: "tag", call: func() error { _, err := NewTagRepo(nil).List(ctx); return err }},
		{name: "talk", call: func() error { _, err := NewTalkRepo(nil).Count(ctx, port.TalkFilter{}); return err }},
		{name: "auth", call: func() error { _, err := NewUserAuthRepo(nil).FindByUsername(ctx, "missing"); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !apperrors.IsKind(test.call(), apperrors.KindUnavailable) {
				t.Fatalf("expected unavailable error")
			}
		})
	}
}

// Keep the test table readable without importing the application package.
func structJobLogFilter() port.JobLogFilter { return port.JobLogFilter{} }
