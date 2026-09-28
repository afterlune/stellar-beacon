package service

import (
	"context"
	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type PhotoAlbumService interface {
	ListPhotoAlbums() model.ResultVO
	ListAuthorAlbums(c *gin.Context) model.ResultVO
	ListAuthorPhotos(c *gin.Context) model.ResultVO
	ListStudioAlbums(c *gin.Context) model.ResultVO
	SaveStudioAlbum(c *gin.Context) model.ResultVO
	DeleteStudioAlbum(c *gin.Context) model.ResultVO
	ListStudioPhotos(c *gin.Context) model.ResultVO
	SaveStudioPhotos(c *gin.Context) model.ResultVO
	DeleteStudioPhotos(c *gin.Context) model.ResultVO
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

func (p *MyPhotoAlbumService) ListAuthorAlbums(c *gin.Context) model.ResultVO {
	albums, err := p.photoAlbumRepository().ListPublicByHandle(c.Request.Context(), c.Param("handle"))
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.PhotoAlbumDTO
	StructCopy(albums, &dtos)
	return model.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) ListAuthorPhotos(c *gin.Context) model.ResultVO {
	albumID, err := strconv.Atoi(c.Param("albumId"))
	if err != nil || albumID <= 0 {
		return model.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().GetPublicByHandle(c.Request.Context(), c.Param("handle"), albumID)
	if err != nil {
		return model.ResultFromError(err)
	}
	current, _ := strconv.Atoi(c.Query("current"))
	size, _ := strconv.Atoi(c.Query("size"))
	photos, err := p.photoRepository().ListPublicByAlbum(c.Request.Context(), album.Id, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	urls := make([]string, 0, len(photos))
	for _, photo := range photos {
		urls = append(urls, photo.PhotoSrc)
	}
	return model.ResultOkWithData(model.PhotoDTO{PhotoAlbumCover: album.AlbumCover, PhotoAlbumName: album.AlbumName, Photos: urls})
}

func (p *MyPhotoAlbumService) ListStudioAlbums(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	albums, err := p.photoAlbumRepository().ListOwned(c.Request.Context(), user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	var dtos []model.PhotoAlbumDTO
	StructCopy(albums, &dtos)
	return model.ResultOkWithData(dtos)
}

func (p *MyPhotoAlbumService) SaveStudioAlbum(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.PhotoAlbumVO
	if err := c.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if rawID := strings.TrimSpace(c.Param("albumId")); rawID != "" {
		albumID, err := strconv.Atoi(rawID)
		if err != nil || albumID <= 0 {
			return model.ResultFailWithMessage("相册不存在")
		}
		vo.Id = albumID
	}
	vo.AlbumName = strings.TrimSpace(vo.AlbumName)
	vo.AlbumDesc = strings.TrimSpace(vo.AlbumDesc)
	vo.AlbumCover = strings.TrimSpace(vo.AlbumCover)
	if vo.AlbumName == "" || len([]rune(vo.AlbumName)) > 20 || len([]rune(vo.AlbumDesc)) > 50 || len(vo.AlbumCover) > 255 || (vo.Status != 1 && vo.Status != 2) {
		return model.ResultFailWithMessage("相册信息不符合要求")
	}
	if err := p.photoAlbumRepository().SaveOwned(c.Request.Context(), entity.TPhotoAlbum{
		Id: vo.Id, AlbumName: vo.AlbumName, AlbumDesc: vo.AlbumDesc,
		AlbumCover: vo.AlbumCover, Status: vo.Status,
	}, user.UserInfoId); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoAlbumService) DeleteStudioAlbum(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	albumID, err := strconv.Atoi(c.Param("albumId"))
	if err != nil || albumID <= 0 {
		return model.ResultFailWithMessage("相册不存在")
	}
	if err := p.photoAlbumRepository().DeleteOwned(c.Request.Context(), albumID, user.UserInfoId); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (p *MyPhotoAlbumService) ListStudioPhotos(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	albumID, err := strconv.Atoi(c.Param("albumId"))
	if err != nil || albumID <= 0 {
		return model.ResultFailWithMessage("相册不存在")
	}
	album, err := p.photoAlbumRepository().GetOwned(c.Request.Context(), albumID, user.UserInfoId)
	if err != nil {
		return model.ResultFromError(err)
	}
	photos, err := p.photoRepository().ListOwnedByAlbum(c.Request.Context(), user.UserInfoId, albumID)
	if err != nil {
		return model.ResultFromError(err)
	}
	result := model.StudioAlbumPhotosDTO{Photos: make([]model.StudioPhotoDTO, 0, len(photos))}
	StructCopy(album, &result.Album)
	for _, photo := range photos {
		result.Photos = append(result.Photos, model.StudioPhotoDTO{Id: photo.Id, PhotoName: photo.PhotoName, PhotoDesc: photo.PhotoDesc, PhotoSrc: photo.PhotoSrc})
	}
	return model.ResultOkWithData(result)
}

func (p *MyPhotoAlbumService) SaveStudioPhotos(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioPhotosVO
	if err := c.ShouldBindJSON(&vo); err != nil || vo.AlbumId <= 0 || len(vo.PhotoUrls) == 0 || len(vo.PhotoUrls) > 50 {
		return model.ResultFailWithMessage("照片参数不符合要求")
	}
	photos := make([]entity.TPhoto, 0, len(vo.PhotoUrls))
	for _, photoURL := range vo.PhotoUrls {
		photoURL = strings.TrimSpace(photoURL)
		if !validStudioWebsite(photoURL) || len(photoURL) > 255 {
			return model.ResultFailWithMessage("照片地址必须是有效的 HTTP(S) 地址")
		}
		photos = append(photos, entity.TPhoto{PhotoName: GetUUID()[:20], PhotoSrc: photoURL})
	}
	if err := p.photoRepository().InsertOwned(c.Request.Context(), user.UserInfoId, vo.AlbumId, photos); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(map[string]int{"saved": len(photos)})
}

func (p *MyPhotoAlbumService) DeleteStudioPhotos(c *gin.Context) model.ResultVO {
	user, ok := currentUser(c)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.StudioDeletePhotosVO
	if err := c.ShouldBindJSON(&vo); err != nil || len(vo.Ids) == 0 || len(vo.Ids) > 200 {
		return model.ResultFailWithMessage("照片参数不符合要求")
	}
	if err := p.photoRepository().DeleteOwned(c.Request.Context(), user.UserInfoId, vo.Ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
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
	_, count, err := p.photoRepository().List(c.Request.Context(), 1, 1, id, False, "")
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
	_, activeCount, err := p.photoRepository().List(c.Request.Context(), 1, 1, id, False, "")
	if err != nil {
		return model.ResultFromError(err)
	}
	_, deletedCount, err := p.photoRepository().List(c.Request.Context(), 1, 1, id, True, "")
	if err != nil {
		return model.ResultFromError(err)
	}
	if activeCount+deletedCount > 0 {
		return model.ResultFailWithMessage("相册中还有照片，请先移走并清理")
	}
	if err := p.photoAlbumRepository().Delete(c.Request.Context(), id); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

var _ PhotoAlbumService = (*MyPhotoAlbumService)(nil)
