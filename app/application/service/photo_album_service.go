package service

import (
	"benetnasch/app/application/support"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"context"
	"strconv"
)

type PhotoAlbumService interface {
	ListPhotoAlbums(ctx context.Context) port.ResultVO
	SavePhotoAlbumCover(c port.Request) port.ResultVO
	SaveOrUpdatePhotoAlbum(c port.Request) port.ResultVO
	ListPhotoAlbumBacks(c port.Request) port.ResultVO
	ListPhotoAlbumBackInfos(ctx context.Context) port.ResultVO
	GetPhotoAlbumBackById(c port.Request) port.ResultVO
	DeletePhotoAlbumById(c port.Request) port.ResultVO
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

func (p *MyPhotoAlbumService) ListPhotoAlbums(ctx context.Context) port.ResultVO {
	albums, err := p.photoAlbumRepository().ListPublic(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.PhotoAlbumDTO
	support.StructCopy(albums, &dtos)
	return port.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) SavePhotoAlbumCover(c port.Request) port.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	ref, err := uploadMultipart(c.Context(), p.storage, file, "photos/")
	if err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOkWithData(ref.URL)
}

func (p *MyPhotoAlbumService) SaveOrUpdatePhotoAlbum(c port.Request) port.ResultVO {
	var vo port.PhotoAlbumVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	duplicate, err := p.photoAlbumRepository().FindByName(c.Context(), vo.AlbumName)
	if err != nil {
		return port.ResultFromError(err)
	}
	if duplicate.Id != 0 && duplicate.Id != vo.Id {
		return port.ResultFailWithMessage("相册名已存在")
	}
	album := port.TPhotoAlbum{Id: vo.Id, AlbumName: vo.AlbumName, AlbumDesc: vo.AlbumDesc, AlbumCover: vo.AlbumCover, Status: vo.Status}
	if err := p.photoAlbumRepository().SaveOrUpdate(c.Context(), album); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoAlbumService) ListPhotoAlbumBacks(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	albums, count, err := p.photoAlbumRepository().ListAdmin(c.Context(), vo.Current, vo.Size, vo.Keywords)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{})
	}
	return port.ResultOkWithData(port.PageResultDTO{Records: albums, Count: int(count)})
}

func (p *MyPhotoAlbumService) ListPhotoAlbumBackInfos(ctx context.Context) port.ResultVO {
	albums, err := p.photoAlbumRepository().ListOptions(ctx)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dtos []port.PhotoAlbumDTO
	support.StructCopy(albums, &dtos)
	return port.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) GetPhotoAlbumBackById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return port.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().Get(c.Context(), id)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindNotFound) {
			return port.ResultFailWithMessage("相册不存在")
		}
		return port.ResultFromError(err)
	}
	_, count, err := p.photoRepository().List(c.Context(), 1, 1, id, support.False)
	if err != nil {
		return port.ResultFromError(err)
	}
	var dto port.PhotoAlbumAdminDTO
	support.StructCopy(album, &dto)
	dto.PhotoCount = int(count)
	return port.ResultOkWithData(dto)
}

func (p *MyPhotoAlbumService) DeletePhotoAlbumById(c port.Request) port.ResultVO {
	id, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoAlbumRepository().Delete(c.Context(), id); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

var _ PhotoAlbumService = (*MyPhotoAlbumService)(nil)
