package oss

import (
	"benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"context"
	"strings"
	"testing"
)

func TestAliyunStorageRefBuildsPublicURL(t *testing.T) {
	storage := &AliyunStorage{publicURL: "https://cdn.example.com/"}

	got := storage.ref("photos/avatar.png")
	want := port.ObjectRef{Key: "photos/avatar.png", URL: "https://cdn.example.com/photos/avatar.png"}
	if got != want {
		t.Fatalf("ref() = %#v, want %#v", got, want)
	}
}

func TestAliyunStoragePutRequiresConfiguredClient(t *testing.T) {
	_, err := (&AliyunStorage{}).Put(context.Background(), "photos/avatar.png", strings.NewReader("image"))
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Put() error kind = %v, want %v", errors.KindOf(err), errors.KindUnavailable)
	}
}
