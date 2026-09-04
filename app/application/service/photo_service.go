package service

import (
	"benetnasch/app/application/support"
	"benetnasch/app/domain/port"
	"container/list"
	"strconv"
)

type PhotoService interface {
	SavePhotosAlbumCover(c port.Request) port.ResultVO
	ListPhotos(c port.Request) port.ResultVO
	UpdatePhoto(c port.Request) port.ResultVO
	SavePhotos(c port.Request) port.ResultVO
	UpdatePhotosAlbum(c port.Request) port.ResultVO
	UpdatePhotoDelete(c port.Request) port.ResultVO
	DeletePhotos(c port.Request) port.ResultVO
	ListPhotosByAlbumId(c port.Request) port.ResultVO
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

func (p *MyPhotoService) SavePhotosAlbumCover(c port.Request) port.ResultVO {
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

func (p *MyPhotoService) ListPhotos(c port.Request) port.ResultVO {
	var vo port.ConditionVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	photos, count, err := p.photoRepository().List(c.Context(), vo.Current, vo.Size, vo.AlbumId, vo.IsDelete)
	if err != nil {
		return port.ResultFromError(err)
	}
	if count == 0 {
		return port.ResultOkWithData(port.PageResultDTO{Records: list.New()})
	}
	var dtos []port.PhotoAdminDTO
	support.StructCopy(photos, &dtos)
	return port.ResultOkWithData(port.PageResultDTO{Records: dtos, Count: int(count)})
}

func (p *MyPhotoService) UpdatePhoto(c port.Request) port.ResultVO {
	var vo port.PhotoInfoVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().Update(c.Context(), port.TPhoto{Id: vo.Id, PhotoName: vo.PhotoName, PhotoDesc: vo.PhotoDesc}); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoService) SavePhotos(c port.Request) port.ResultVO {
	var vo port.PhotoVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	albumID, err := strconv.Atoi(vo.AlbumId)
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	uuid := support.GetUUID()
	photos := make([]port.TPhoto, 0, len(vo.PhotoUrls))
	for _, url := range vo.PhotoUrls {
		photos = append(photos, port.TPhoto{AlbumId: albumID, PhotoName: uuid[:20], PhotoSrc: url})
	}
	if err := p.photoRepository().InsertMany(c.Context(), photos); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoService) UpdatePhotosAlbum(c port.Request) port.ResultVO {
	var vo port.PhotoVO1
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().UpdateAlbum(c.Context(), vo.PhotoIds, vo.AlbumId); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoService) UpdatePhotoDelete(c port.Request) port.ResultVO {
	var vo port.DeleteVO
	if err := c.Bind(&vo); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().UpdateDelete(c.Context(), vo.Ids, vo.IsDelete); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoService) DeletePhotos(c port.Request) port.ResultVO {
	var ids []int
	if err := c.Bind(&ids); err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	if err := p.photoRepository().Delete(c.Context(), ids); err != nil {
		return port.ResultFromError(err)
	}
	return port.ResultOk()
}

func (p *MyPhotoService) ListPhotosByAlbumId(c port.Request) port.ResultVO {
	albumID, err := strconv.Atoi(c.Param("albumId"))
	if err != nil {
		return port.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().Get(c.Context(), albumID)
	if err != nil {
		return port.ResultFailWithMessage("相册不存在")
	}
	if album.IsDelete != support.False || album.Status != 1 {
		return port.ResultFailWithMessage("相册不存在")
	}
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		return port.ResultFailWithMessage("参数格式不正确")
	}
	photos, err := p.photoRepository().ListPublicByAlbum(c.Context(), albumID, current, size)
	if err != nil {
		return port.ResultFromError(err)
	}
	if len(photos) == 0 {
		return port.ResultOkWithData(port.PhotoDTO{PhotoAlbumCover: album.AlbumCover, PhotoAlbumName: album.AlbumName, Photos: list.New()})
	}
	urls := make([]string, 0, len(photos))
	for _, photo := range photos {
		urls = append(urls, photo.PhotoSrc)
	}
	return port.ResultOkWithData(port.PhotoDTO{PhotoAlbumCover: album.AlbumCover, PhotoAlbumName: album.AlbumName, Photos: urls})
}

var _ PhotoService = (*MyPhotoService)(nil)
