package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infrastructure/persistence/ormInit"
	"benetnasch/app/infrastructure/persistence/pgsql"
	"benetnasch/app/infrastructure/zlog"
	"fmt"
)

func ListPhotoAlbumsAdmin(current, size int, vo *model.ConditionVO) []*model.PhotoAlbumAdminDTO {
	s := ""
	if vo.Keywords != "" {
		s += " and album_name like '%" + vo.Keywords + "%'"
	}
	s = fmt.Sprintf(pgsql.ListPhotoAlbumsAdmin, s, size, (current-1)*size)
	engine := ormInit.GetEngine()
	var photosAlbumAdmin []*model.PhotoAlbumAdminDTO
	err := engine.SQL(s).Find(&photosAlbumAdmin)
	if err != nil {
		zlog.Error(err.Error())
	}

	return photosAlbumAdmin
}

func PhotoAlbums() []*model.PhotoAlbumDTO {
	engine := ormInit.GetEngine()
	var photoAlbums []*model.PhotoAlbumDTO
	err := engine.SQL("select id, album_name, album_desc, album_cover from t_photo_album where status = 1 and is_delete = 0 order by id desc").Find(&photoAlbums)
	if err != nil {
		zlog.Error(err.Error())
	}

	return photoAlbums
}
