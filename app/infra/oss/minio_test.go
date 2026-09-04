package oss

import (
	"benetnasch/app/domain/errors"
	"benetnasch/app/infra/config"
	"context"
	stderrors "errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestMinioStorageRefBuildsPublicURL(t *testing.T) {
	storage := &MinioStorage{publicURL: "http://127.0.0.1:19000/test/"}
	got := storage.ref("articles/image.png")
	if got.Key != "articles/image.png" || got.URL != "http://127.0.0.1:19000/test/articles/image.png" {
		t.Fatalf("unexpected object reference: %#v", got)
	}
}

func TestMinioEndpoint(t *testing.T) {
	for _, test := range []struct {
		name   string
		raw    string
		host   string
		secure bool
	}{
		{name: "plain", raw: "minio:9000", host: "minio:9000"},
		{name: "http", raw: "http://minio:9000", host: "minio:9000"},
		{name: "https", raw: "https://minio:9000", host: "minio:9000", secure: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			host, secure, err := minioEndpoint(test.raw)
			if err != nil || host != test.host || secure != test.secure {
				t.Fatalf("minioEndpoint(%q) = %q, %t, %v", test.raw, host, secure, err)
			}
		})
	}
}

func TestMinioStoragePutRequiresConfiguredClient(t *testing.T) {
	_, err := (&MinioStorage{}).Put(context.Background(), "test.txt", strings.NewReader("test"))
	if !errors.IsKind(err, errors.KindUnavailable) {
		t.Fatalf("Put() error kind = %v, want %v", errors.KindOf(err), errors.KindUnavailable)
	}
}

func TestMinioStoragePreservesContextErrors(t *testing.T) {
	server := httptest.NewServer(nil)
	defer server.Close()
	client, err := minio.New(server.Listener.Addr().String(), &minio.Options{
		Creds: credentials.NewStaticV4("access", "secret", ""),
	})
	if err != nil {
		t.Fatal(err)
	}
	storage := &MinioStorage{client: client, bucket: "bucket"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := storage.Put(ctx, "test.txt", strings.NewReader("test")); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v, want context canceled", err)
	}
}

func TestNewObjectStorageRejectsUnknownProvider(t *testing.T) {
	_, err := NewObjectStorage(&config.Oss{Provider: "unknown"})
	if err == nil {
		t.Fatal("NewObjectStorage() unexpectedly accepted unknown provider")
	}
}
