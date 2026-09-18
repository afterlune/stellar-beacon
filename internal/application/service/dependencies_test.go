package service

import (
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"testing"
)

func TestServiceConstructorsRejectMissingDependencies(t *testing.T) {
	tests := []struct {
		name      string
		construct func() error
	}{
		{name: "article", construct: func() error {
			_, err := NewArticleService(ArticleServiceDeps{})
			return err
		}},
		{name: "stellar beacon info", construct: func() error {
			_, err := NewStellarBeaconInfoService(StellarBeaconInfoServiceDeps{})
			return err
		}},
		{name: "user info", construct: func() error {
			_, err := NewUserInfoService(UserInfoServiceDeps{})
			return err
		}},
		{name: "comment", construct: func() error {
			_, err := NewCommentService(CommentServiceDeps{})
			return err
		}},
		{name: "photo album", construct: func() error {
			_, err := NewPhotoAlbumService(PhotoAlbumServiceDeps{})
			return err
		}},
		{name: "photo", construct: func() error {
			_, err := NewPhotoService(PhotoServiceDeps{})
			return err
		}},
		{name: "talk", construct: func() error {
			_, err := NewTalkService(TalkServiceDeps{})
			return err
		}},
		{name: "user auth", construct: func() error {
			_, err := NewUserAuthService(UserAuthServiceDeps{})
			return err
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.construct()
			if err == nil {
				t.Fatal("expected missing dependency error")
			}
			if !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Fatalf("expected validation error, got %T: %v", err, err)
			}
		})
	}
}

func TestArticleServiceRejectsMissingSearchDependency(t *testing.T) {
	_, err := NewArticleService(ArticleServiceDeps{
		Repo:             &fakeArticleRepository{},
		ContentAnalytics: fakeContentAnalyticsRepository{},
		Cache:            fakeServiceCache{},
		Storage:          fakeServiceStorage{},
	})
	if err == nil || !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
