package service

import (
	"benetnasch/internal/domain/entity"
	apperrors "benetnasch/internal/domain/errors"
	"benetnasch/internal/domain/port"
	"benetnasch/internal/interfaces/http/model"
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PhotoAlbumService interface {
	ListPhotoAlbums() model.ResultVO
	SavePhotoAlbumCover(c *gin.Context) model.ResultVO
	SaveOrUpdatePhotoAlbum(c *gin.Context) model.ResultVO
	ListPhotoAlbumBacks(c *gin.Context) model.ResultVO
	ListPhotoAlbumBackInfos() model.ResultVO
	GetPhotoAlbumBackById(c *gin.Context) model.ResultVO
	DeletePhotoAlbumById(c *gin.Context) model.ResultVO
}

type MyPhotoAlbumService struct {
	repo    port.PhotoAlbumRepository
	photos  port.PhotoRepository
	storage port.ObjectStorage
}

func NewPhotoAlbumService(deps PhotoAlbumServiceDeps) (*MyPhotoAlbumService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyPhotoAlbumService{repo: deps.Repo, photos: deps.Photos, storage: deps.Storage}, nil
}

func (p *MyPhotoAlbumService) photoAlbumRepository() port.PhotoAlbumRepository {
	return p.repo
}

func (p *MyPhotoAlbumService) photoRepository() port.PhotoRepository {
	return p.photos
}

func (p *MyPhotoAlbumService) ListPhotoAlbums() model.ResultVO {
	albums, err := p.photoAlbumRepository().ListPublic(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.PhotoAlbumDTO
	StructCopy(albums, &dtos)
	return model.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) SavePhotoAlbumCover(c *gin.Context) model.ResultVO {
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

func (p *MyPhotoAlbumService) SaveOrUpdatePhotoAlbum(c *gin.Context) model.ResultVO {
	var vo model.PhotoAlbumVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	duplicate, err := p.photoAlbumRepository().FindByName(c.Request.Context(), vo.AlbumName)
	if err != nil {
		return model.ResultFromError(err)
	}
	if duplicate.Id != 0 && duplicate.Id != vo.Id {
		return model.ResultFailWithMessage("相册名已存在")
	}
	album := entity.TPhotoAlbum{Id: vo.Id, AlbumName: vo.AlbumName, AlbumDesc: vo.AlbumDesc, AlbumCover: vo.AlbumCover, Status: vo.Status}
	if err := p.photoAlbumRepository().SaveOrUpdate(c.Request.Context(), album); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoAlbumService) ListPhotoAlbumBacks(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := c.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	albums, count, err := p.photoAlbumRepository().ListAdmin(c.Request.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: albums, Count: int(count)})
}

func (p *MyPhotoAlbumService) ListPhotoAlbumBackInfos() model.ResultVO {
	albums, err := p.photoAlbumRepository().ListOptions(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.PhotoAlbumDTO
	StructCopy(albums, &dtos)
	return model.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) GetPhotoAlbumBackById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return model.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().Get(c.Request.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return model.ResultFailWithMessage("相册不存在")
		}
		return model.ResultFromError(err)
	}
	_, count, err := p.photoRepository().List(c.Request.Context(), 1, 1, id, False)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dto model.PhotoAlbumAdminDTO
	StructCopy(album, &dto)
	dto.PhotoCount = int(count)
	return model.ResultOkWithData(dto)
}

func (p *MyPhotoAlbumService) DeletePhotoAlbumById(c *gin.Context) model.ResultVO {
	id, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoAlbumRepository().Delete(c.Request.Context(), id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

var _ PhotoAlbumService = (*MyPhotoAlbumService)(nil)
