package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"container/list"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/xorm"
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

type MyPhotoService struct{}

func (p *MyPhotoService) SavePhotosAlbumCover(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "photos/")
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func (p *MyPhotoService) ListPhotos(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var photos []entity.TPhoto
	limit, offset := pgsql.Page(vo.Current, vo.Size)
	if vo.AlbumId != 0 {
		err = engine.Prepare().
			SQL("select * from t_photo where album_id = ? and is_delete = ? order by id, update_time desc limit ? offset ?",
				vo.AlbumId, vo.IsDelete, limit, offset).Find(&photos)
	} else {
		err = engine.Prepare().
			SQL("select * from t_photo where is_delete = ? order by id, update_time desc limit ? offset ?",
				vo.IsDelete, limit, offset).Find(&photos)
	}
	if err != nil {
		return model.ResultFail()
	}
	count, err := engine.Count(&entity.TPhoto{})
	if err != nil {
		return model.ResultFail()
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New()})
	}
	var dtos []model.PhotoAdminDTO
	shared.StructCopy(photos, &dtos)
	return model.ResultOkWithData(model.PageResultDTO{Records: dtos, Count: int(count)})
}

func (p *MyPhotoService) UpdatePhoto(c *gin.Context) model.ResultVO {
	var vo model.PhotoInfoVO
	err := c.ShouldBind(&vo)
	if err != nil {
		return model.ResultFail()
	}
	var photo entity.TPhoto
	shared.StructCopy(vo, &photo)
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.ID(photo.Id).Update(&photo)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (p *MyPhotoService) SavePhotos(c *gin.Context) model.ResultVO {
	var vo model.PhotoVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	uuid := shared.GetUUID()
	aId, _ := strconv.Atoi(vo.AlbumId)
	var photos []entity.TPhoto
	for _, v := range vo.PhotoUrls {
		photos = append(photos, entity.TPhoto{
			AlbumId:   aId,
			PhotoName: uuid[:20],
			PhotoSrc:  v,
		})
	}
	if len(photos) == 0 {
		return model.ResultOk()
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		_, err := session.Insert(&photos)
		return err
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (p *MyPhotoService) UpdatePhotosAlbum(c *gin.Context) model.ResultVO {
	var vo model.PhotoVO1
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var photos []entity.TPhoto
	for _, v := range vo.PhotoIds {
		photos = append(photos, entity.TPhoto{
			Id:      v,
			AlbumId: vo.AlbumId,
		})
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		for _, v := range photos {
			if _, err := session.ID(v.Id).Update(&v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (p *MyPhotoService) UpdatePhotoDelete(c *gin.Context) model.ResultVO {
	var vo model.DeleteVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var photos []entity.TPhoto
	for _, v := range vo.Ids {
		photos = append(photos, entity.TPhoto{
			Id:       v,
			IsDelete: vo.IsDelete,
		})
	}
	if err := ormInit.WithTx(c.Request.Context(), func(session *xorm.Session) error {
		for _, v := range photos {
			if _, err := session.ID(v.Id).MustCols("is_delete").Update(&v); err != nil {
				return err
			}
		}
		if vo.IsDelete == 0 && len(vo.Ids) > 0 {
			var photoList []entity.TPhoto
			if err := session.Select("album_id").In("id", vo.Ids).GroupBy("album_id").Find(&photoList); err != nil {
				return err
			}
			var photoAlbums []entity.TPhotoAlbum
			for _, v := range photoList {
				photoAlbums = append(photoAlbums, entity.TPhotoAlbum{Id: v.AlbumId, IsDelete: shared.FALSE})
			}
			for _, v := range photoAlbums {
				if _, err := session.ID(v.Id).Update(&v); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		if err != nil {
			zlog.Error(err.Error())
		}
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (p *MyPhotoService) DeletePhotos(c *gin.Context) model.ResultVO {
	var iDs []int
	err := c.ShouldBind(&iDs)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	_, err = ormInit.GetEngine().Prepare().In("id", iDs).Delete(&entity.TPhoto{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}

func (p *MyPhotoService) ListPhotosByAlbumId(c *gin.Context) model.ResultVO {
	current := c.Query("current")
	size := c.Query("size")
	albumID, parseErr := strconv.Atoi(c.Param("albumId"))
	if parseErr != nil {
		return model.ResultFailWithMessage("相册不存在")
	}
	engine := ormInit.GetEngine()
	var photoAlbum entity.TPhotoAlbum
	_, err := engine.Where("id = ? and is_delete = 0 and status = 1", albumID).Get(&photoAlbum)
	if err != nil {
		zlog.Error(err.Error())
	}

	if photoAlbum.Id == 0 {
		return model.ResultFailWithMessage("相册不存在")
	}
	var photos []entity.TPhoto
	cu, err := strconv.Atoi(current)
	if err != nil {
		cu = 1
	}

	si, err := strconv.Atoi(size)
	if err != nil {
		si = pgsql.DefaultPageSize
	}
	limit, offset := pgsql.Page(cu, si)

	err = engine.SQL("select photo_src from t_photo where album_id = ? and is_delete = 0 order by id desc limit ? offset ?",
		albumID, limit, offset).Find(&photos)
	if err != nil {
		zlog.Error(err.Error())
	}

	if len(photos) == 0 {
		return model.ResultOkWithData(model.PhotoDTO{
			PhotoAlbumCover: photoAlbum.AlbumCover,
			PhotoAlbumName:  photoAlbum.AlbumName,
			Photos:          list.New(),
		})
	}
	var psrc []string
	for _, v := range photos {
		psrc = append(psrc, v.PhotoSrc)
	}
	return model.ResultOkWithData(model.PhotoDTO{
		PhotoAlbumCover: photoAlbum.AlbumCover,
		PhotoAlbumName:  photoAlbum.AlbumName,
		Photos:          psrc,
	})
}
