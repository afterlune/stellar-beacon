package service

import (
	"benetnasch/app/domain/entity"
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/oss"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/shared"
	"benetnasch/app/infra/zlog"
	"github.com/gin-gonic/gin"
	"strconv"
	"xorm.io/builder"
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

type MyPhotoAlbumService struct{}

func (p *MyPhotoAlbumService) ListPhotoAlbums() model.ResultVO {
	data := photoAlbumRepo.PhotoAlbums()
	return model.ResultOkWithData(data)
}

func (p *MyPhotoAlbumService) SavePhotoAlbumCover(c *gin.Context) model.ResultVO {
	file, err := c.FormFile("file")
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	fileUri := oss.Upload(file, "photos/")
	return model.ResultOkWithData(shared.FILEURL + fileUri)
}

func (p *MyPhotoAlbumService) SaveOrUpdatePhotoAlbum(c *gin.Context) model.ResultVO {
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

func (p *MyPhotoAlbumService) ListPhotoAlbumBacks(c *gin.Context) model.ResultVO {
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
	data := photoAlbumRepo.ListPhotoAlbumsAdmin(vo.Current, vo.Size, &vo)
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (p *MyPhotoAlbumService) ListPhotoAlbumBackInfos() model.ResultVO {
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

func (p *MyPhotoAlbumService) GetPhotoAlbumBackById(c *gin.Context) model.ResultVO {
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

func (p *MyPhotoAlbumService) DeletePhotoAlbumById(c *gin.Context) model.ResultVO {
	id, _ := strconv.Atoi(c.Param("albumId"))
	_, err := ormInit.GetEngine().Prepare().ID(id).Delete(&entity.TPhotoAlbum{})
	if err != nil {
		zlog.Error(err.Error())
		return model.ResultFail()
	}
	return model.ResultOk()
}
