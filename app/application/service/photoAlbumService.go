package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/oss"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/repository"
	"benetnasch/app/infrastructure/shared"
	"benetnasch/app/infrastructure/zlog"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/builder"
)

func ListPhotoAlbums() model.ResultVO {
	data := repository.PhotoAlbums()
	return model.ResultOkWithData(data)
}

func SavePhotoAlbumCover(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "photos/")
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func SaveOrUpdatePhotoAlbum(c *gin.Context) model.ResultVO {
	var vo model.PhotoAlbumVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var album entity.TPhotoAlbum
	engine := ormInit.GetEngine()
	_, err = engine.Prepare().Select("id").Where(builder.Eq{"album_name": vo.AlbumName}).Get(&album)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if album.Id != 0 && album.Id != vo.Id {
		return model.ResultFailWithMessage("相册名已存在")
	}
	var photoAlbum entity.TPhotoAlbum
	shared.StructCopy(vo, &photoAlbum)
	session := engine.NewSession()
	session.Begin()
	defer session.Close()
	if photoAlbum.Id != 0 {
		_, err = session.Prepare().ID(photoAlbum.Id).Update(&photoAlbum)
	} else {
		_, err = session.Prepare().Insert(&photoAlbum)
	}
	if err != nil {
		zlog.Error(err.Error())
		session.Rollback()
		return model.ResultFail()
	}
	session.Commit()
	return model.ResultOk()
}

func ListPhotoAlbumBacks(c *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	err := c.ShouldBind(&vo)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	engine := ormInit.GetEngine()
	var count int64
	if vo.Keywords != "" {
		count, err = engine.Prepare().Where(builder.Like{"album_name", vo.Keywords}, builder.Eq{"is_delete": shared.FALSE}).Count(&entity.TPhotoAlbum{})

	} else {
		count, err = engine.Prepare().Where(builder.Eq{"is_delete": shared.FALSE}).Count(&entity.TPhotoAlbum{})
	}
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{})
	}
	data := repository.ListPhotoAlbumsAdmin(vo.Current, vo.Size, &vo)
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func ListPhotoAlbumBackInfos() model.ResultVO {
	var photoAlbums []entity.TPhotoAlbum
	err := ormInit.GetEngine().Prepare().Where(builder.Eq{"is_delete": shared.FALSE}).Find(&photoAlbums)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var dtos []model.PhotoAlbumDTO
	shared.StructCopy(photoAlbums, &dtos)
	return model.ResultOkWithData(dtos)
}

func GetPhotoAlbumBackById(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("albumId"))
	engine := ormInit.GetEngine()
	var pm entity.TPhotoAlbum
	_, err := engine.Prepare().ID(id).Get(&pm)
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	count, err := engine.Prepare().Where(builder.Eq{"album_id": id, "is_delete": shared.FALSE}).Count(&entity.TPhoto{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	var album model.PhotoAlbumAdminDTO
	shared.StructCopy(pm, &album)
	album.PhotoCount = int(count)
	return model.ResultOkWithData(album)
}

func DeletePhotoAlbumById(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("albumId"))
	_, err := ormInit.GetEngine().Prepare().ID(id).Delete(&entity.TPhotoAlbum{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
