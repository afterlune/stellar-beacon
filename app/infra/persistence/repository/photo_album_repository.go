package repository

import (
	"benetnasch/app/facade/model"
	"benetnasch/app/infra/persistence/ormInit"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/zlog"
)

type PhotoAlbumRepo interface {
	ListPhotoAlbumsAdmin(current, size int, vo *model.ConditionVO) []*model.PhotoAlbumAdminDTO
	PhotoAlbums() []*model.PhotoAlbumDTO
}

type MyPhotoAlbumRepo struct{}

func (p *MyPhotoAlbumRepo) ListPhotoAlbumsAdmin(current, size int, vo *model.ConditionVO) []*model.PhotoAlbumAdminDTO {
	limit, offset := pgsql.Page(current, size)
	query := "SELECT pa.id, album_name, album_desc, album_cover, COUNT(a.id) AS photo_count, status FROM (SELECT id, album_name, album_desc, album_cover, status FROM t_photo_album WHERE is_delete = 0"
	args := []interface{}{}
	if vo.Keywords != "" {
		query += " AND album_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(vo.Keywords))
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?) pa LEFT JOIN (SELECT id, album_id FROM t_photo WHERE is_delete = 0) a ON pa.id = a.album_id GROUP BY pa.id, album_name, album_desc, album_cover, status"
	args = append(args, limit, offset)
	var albums []*model.PhotoAlbumAdminDTO
	if err := ormInit.GetEngine().SQL(query, args...).Find(&albums); err != nil {
		zlog.Error("list admin photo albums: " + err.Error())
	}
	return albums
}

func (p *MyPhotoAlbumRepo) PhotoAlbums() []*model.PhotoAlbumDTO {
	var albums []*model.PhotoAlbumDTO
	query := "SELECT id, album_name, album_desc, album_cover FROM t_photo_album WHERE status = 1 AND is_delete = 0 ORDER BY id DESC"
	if err := ormInit.GetEngine().SQL(query).Find(&albums); err != nil {
		zlog.Error("list photo albums: " + err.Error())
	}
	return albums
}
