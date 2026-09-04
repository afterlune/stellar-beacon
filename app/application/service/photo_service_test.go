package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/domain/port"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type photoAlbumLookupRepository struct{}

func (photoAlbumLookupRepository) ListPublic(context.Context) ([]entity.TPhotoAlbum, error) {
	return nil, nil
}
func (photoAlbumLookupRepository) FindByName(context.Context, string) (entity.TPhotoAlbum, error) {
	return entity.TPhotoAlbum{}, nil
}
func (photoAlbumLookupRepository) ListAdmin(context.Context, int, int, string) ([]port.PhotoAlbumAdmin, int64, error) {
	return nil, 0, nil
}
func (photoAlbumLookupRepository) ListOptions(context.Context) ([]entity.TPhotoAlbum, error) {
	return nil, nil
}
func (photoAlbumLookupRepository) Get(context.Context, int) (entity.TPhotoAlbum, error) {
	return entity.TPhotoAlbum{Id: 1, AlbumName: "album", Status: 1}, nil
}
func (photoAlbumLookupRepository) SaveOrUpdate(context.Context, entity.TPhotoAlbum) error { return nil }
func (photoAlbumLookupRepository) Delete(context.Context, int) error                      { return nil }

type photoListProbeRepository struct {
	called bool
}

func (photoListProbeRepository) List(context.Context, int, int, int, int) ([]entity.TPhoto, int64, error) {
	return nil, 0, nil
}
func (photoListProbeRepository) Update(context.Context, entity.TPhoto) error { return nil }
func (photoListProbeRepository) InsertMany(context.Context, []entity.TPhoto) error {
	return nil
}
func (photoListProbeRepository) UpdateAlbum(context.Context, []int, int) error { return nil }
func (photoListProbeRepository) UpdateDelete(context.Context, []int, int) error {
	return nil
}
func (photoListProbeRepository) Delete(context.Context, []int) error { return nil }
func (p *photoListProbeRepository) ListPublicByAlbum(context.Context, int, int, int) ([]entity.TPhoto, error) {
	p.called = true
	return nil, nil
}

func TestPhotoServiceRejectsInvalidPublicPagination(t *testing.T) {
	for _, query := range []string{"current=invalid&size=10", "current=1&size=invalid"} {
		repository := &photoListProbeRepository{}
		service, err := NewPhotoService(PhotoServiceDeps{
			Repo:    repository,
			Albums:  photoAlbumLookupRepository{},
			Storage: fakeServiceStorage{},
		})
		if err != nil {
			t.Fatal(err)
		}
		gin.SetMode(gin.TestMode)
		request := httptest.NewRequest(http.MethodGet, "/photos/albums/1?"+query, nil)
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = request
		context.Params = gin.Params{{Key: "albumId", Value: "1"}}

		result := service.ListPhotosByAlbumId(serviceTestRequest{ginContextForServiceTest: context})
		if result.Flag || result.Message != "参数格式不正确" {
			t.Fatalf("query %q: unexpected result: %+v", query, result)
		}
		if repository.called {
			t.Fatalf("query %q reached repository with invalid pagination", query)
		}
	}
}
