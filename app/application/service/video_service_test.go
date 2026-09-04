package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/facade/model"
)

type videoRepositoryFake struct {
	created []port.Video
	deleted []string
	rows    []port.Video
	err     error
}

func (f *videoRepositoryFake) Create(_ context.Context, video port.Video) error {
	f.created = append(f.created, video)
	return f.err
}

func (f *videoRepositoryFake) ListPublic(context.Context, int, int) ([]port.Video, int, error) {
	return f.rows, len(f.rows), f.err
}

func (f *videoRepositoryFake) Delete(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return f.err
}

type videoStorageFake struct {
	key  string
	body []byte
	err  error
}

func (f *videoStorageFake) Put(_ context.Context, key string, reader io.Reader) (port.ObjectRef, error) {
	f.key = key
	if f.err != nil {
		return port.ObjectRef{}, f.err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return port.ObjectRef{}, err
	}
	f.body = data
	return port.ObjectRef{Key: key, URL: "https://cdn.example/" + key}, nil
}

func videoFileHeader(t *testing.T, filename, contentType string, body []byte) *multipart.FileHeader {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/admin/videos/upload", &bytes.Buffer{})
	requestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(requestBody)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request.Body = io.NopCloser(bytes.NewReader(requestBody.Bytes()))
	request.ContentLength = int64(requestBody.Len())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatal(err)
	}
	file, header := request.MultipartForm.File["file"][0], request.MultipartForm.File["file"][0]
	if contentType != "" {
		file.Header.Set("Content-Type", contentType)
	}
	return header
}

func TestVideoServiceValidatesSignatureStoresObjectAndCreatesMetadata(t *testing.T) {
	repository := &videoRepositoryFake{}
	storage := &videoStorageFake{}
	service, err := NewVideoService(VideoServiceDeps{
		Repository: repository,
		Storage:    storage,
		Now:        func() time.Time { return time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC) },
		Enabled:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	file := videoFileHeader(t, "night.mp4", "video/mp4", []byte{0, 0, 0, 24, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'})
	result := service.Upload(context.Background(), VideoUploadInput{File: file, Description: "a safe clip"})
	if !result.Flag || len(repository.created) != 1 || len(storage.body) == 0 {
		t.Fatalf("upload result=%+v created=%+v stored=%d", result, repository.created, len(storage.body))
	}
	created := repository.created[0]
	if created.Source != port.VideoSourceLocal || created.MIMEType != "video/mp4" || created.SizeBytes != int64(len(storage.body)) || !strings.HasPrefix(storage.key, "videos/") {
		t.Fatalf("created video=%+v key=%q", created, storage.key)
	}
	if result.Data.(model.VideoDTO).URL != "https://cdn.example/"+storage.key {
		t.Fatalf("unexpected DTO=%+v", result.Data)
	}
}

func TestVideoServiceRejectsOversizeAndMismatchedContent(t *testing.T) {
	service, err := NewVideoService(VideoServiceDeps{Repository: &videoRepositoryFake{}, Storage: &videoStorageFake{}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	oversized := &multipart.FileHeader{Filename: "large.mp4", Size: port.MaxVideoBytes + 1}
	if result := service.Upload(context.Background(), VideoUploadInput{File: oversized}); result.Flag || result.Code != 51000 {
		t.Fatalf("oversized result=%+v", result)
	}
	textFile := videoFileHeader(t, "fake.mp4", "video/mp4", []byte("not a video"))
	if result := service.Upload(context.Background(), VideoUploadInput{File: textFile}); result.Flag || result.Code != 51000 {
		t.Fatalf("mismatched result=%+v", result)
	}
}

func TestVideoServiceRequiresExactHTTPSExternalOrigin(t *testing.T) {
	repository := &videoRepositoryFake{}
	service, err := NewVideoService(VideoServiceDeps{Repository: repository, Storage: &videoStorageFake{}, AllowedOrigins: []string{"https://video.example"}, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	result := service.CreateExternal(context.Background(), ExternalVideoInput{Title: "远程影像", URL: "https://video.example/embed/abc?autoplay=0"})
	if !result.Flag || len(repository.created) != 1 {
		t.Fatalf("external result=%+v", result)
	}
	if result := service.CreateExternal(context.Background(), ExternalVideoInput{Title: "坏链接", URL: "https://evil.example/embed/abc"}); result.Flag || result.Code != 40300 {
		t.Fatalf("unallowlisted result=%+v", result)
	}
}
