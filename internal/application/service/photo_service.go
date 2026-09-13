package service

import (
	"container/list"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PhotoService interface {
	SavePhotosAlbumCover(c *gin.Context) model.ResultVO
	ListPhotos(c *gin.Context) model.ResultVO
	UpdatePhoto(c *gin.Context) model.ResultVO
	SavePhotos(c *gin.Context) model.ResultVO
	UpdatePhotosAlbum(c *gin.Context) model.ResultVO
	UpdatePhotoDelete(c *gin.Context) model.ResultVO
	DeletePhotos(c *gin.Context) model.ResultVO
	ListPhotosByAlbumId(c *gin.Context) model.ResultVO
}

type MyPhotoService struct {
	repo    port.PhotoRepository
	albums  port.PhotoAlbumRepository
	storage port.ObjectStorage
}

func NewPhotoService(deps PhotoServiceDeps) (*MyPhotoService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyPhotoService{repo: deps.Repo, albums: deps.Albums, storage: deps.Storage}, nil
}

func (p *MyPhotoService) photoRepository() port.PhotoRepository {
	return p.repo
}

func (p *MyPhotoService) photoAlbumRepository() port.PhotoAlbumRepository {
	return p.albums
}

func (p *MyPhotoService) SavePhotosAlbumCover(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	ref, err := uploadMultipart(c.Request.Context(), p.storage, file, "photos/")
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(ref.URL)
}

func (p *MyPhotoService) ListPhotos(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	photos, count, err := p.photoRepository().List(c.Request.Context(), vo.Current, vo.Size, vo.AlbumId, vo.IsDelete)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New()})
	}
	var dtos []model.PhotoAdminDTO
	StructCopy(photos, &dtos)
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (p *MyPhotoService) UpdatePhoto(c *gin.Context) model.ResultVO {
	var vo model.PhotoInfoVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().Update(c.Request.Context(), entity.TPhoto{Id: vo.Id, PhotoName: vo.PhotoName, PhotoDesc: vo.PhotoDesc}); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoService) SavePhotos(c *gin.Context) model.ResultVO {
	var vo model.PhotoVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	albumID, err := strconv.Atoi(vo.AlbumId)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	uuid := GetUUID()
	photos := make([]entity.TPhoto, 0, len(vo.PhotoUrls))
	for _, url := range vo.PhotoUrls {
		photos = append(photos, entity.TPhoto{AlbumId: albumID, PhotoName: uuid[:20], PhotoSrc: url})
	}
	if err := p.photoRepository().InsertMany(c.Request.Context(), photos); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoService) UpdatePhotosAlbum(c *gin.Context) model.ResultVO {
	var vo model.PhotoVO1
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().UpdateAlbum(c.Request.Context(), vo.PhotoIds, vo.AlbumId); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoService) UpdatePhotoDelete(c *gin.Context) model.ResultVO {
	var vo model.DeleteVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().UpdateDelete(c.Request.Context(), vo.Ids, vo.IsDelete); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoService) DeletePhotos(c *gin.Context) model.ResultVO {
	var ids []int
	if err := c.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().Delete(c.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoService) ListPhotosByAlbumId(c *gin.Context) model.ResultVO {
	albumID, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return model.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().Get(c.Request.Context(), albumID)
	if err != nil {
		return model.ResultFailWithMessage("相册不存在")
	}
	if album.IsDelete != False || album.Status != 1 {
		return model.ResultFailWithMessage("相册不存在")
	}
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	photos, err := p.photoRepository().ListPublicByAlbum(c.Request.Context(), albumID, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	if len(photos) == 0 {
		return model.ResultOkWithData(model.PhotoDTO{PhotoAlbumCover: album.AlbumCover, PhotoAlbumName: album.AlbumName, Photos: list.New()})
	}
	urls := make([]string, 0, len(photos))
	for _, photo := range photos {
		urls = append(urls, photo.PhotoSrc)
	}
	return model.ResultOkWithData(model.PhotoDTO{PhotoAlbumCover: album.AlbumCover, PhotoAlbumName: album.AlbumName, Photos: urls})
}

var _ PhotoService = (*MyPhotoService)(nil)
