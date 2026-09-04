package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/row"
	"context"
	"strings"

	"xorm.io/xorm"
)

var _ port.PhotoAlbumRepository = (*MyPhotoAlbumRepository)(nil)
var _ port.PhotoRepository = (*MyPhotoRepository)(nil)

type MyPhotoAlbumRepository struct{ engine *xorm.Engine }

func NewPhotoAlbumRepository(engine *xorm.Engine) *MyPhotoAlbumRepository {
	return &MyPhotoAlbumRepository{engine: engine}
}

func (r *MyPhotoAlbumRepository) ListPublic(ctx context.Context) ([]port.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.public")
	if err != nil {
		return nil, err
	}
	var albums []row.TPhotoAlbum
	if err := session.Where("status = ? AND is_delete = ?", 1, 0).OrderBy("id DESC").Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.public", err)
	}
	return row.FromPhotoAlbums(albums), nil
}

func (r *MyPhotoAlbumRepository) FindByName(ctx context.Context, name string) (port.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.find_name")
	if err != nil {
		return port.TPhotoAlbum{}, err
	}
	var album row.TPhotoAlbum
	found, err := session.Select("id, album_name").Where("album_name = ?", name).Get(&album)
	if err != nil {
		return port.TPhotoAlbum{}, apperrors.Unavailable("photo_album.find_name", err)
	}
	if !found {
		return port.TPhotoAlbum{}, nil
	}
	return row.FromPhotoAlbum(album), nil
}

func (r *MyPhotoAlbumRepository) ListAdmin(ctx context.Context, current, size int, keywords string) ([]port.PhotoAlbumAdmin, int64, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.admin")
	if err != nil {
		return nil, 0, err
	}
	where := " WHERE pa.is_delete = 0"
	args := []interface{}{}
	if strings.TrimSpace(keywords) != "" {
		where += " AND pa.album_name LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(keywords))
	}
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_photo_album pa"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("photo_album.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	query := "SELECT pa.id, pa.album_name, pa.album_desc, pa.album_cover, COUNT(p.id) AS photo_count, pa.status FROM t_photo_album pa LEFT JOIN t_photo p ON p.album_id = pa.id AND p.is_delete = 0" + where + " GROUP BY pa.id, pa.album_name, pa.album_desc, pa.album_cover, pa.status ORDER BY pa.id DESC LIMIT ? OFFSET ?"
	var albums []port.PhotoAlbumAdmin
	if err := session.SQL(query, args...).Find(&albums); err != nil {
		return nil, 0, apperrors.Unavailable("photo_album.admin", err)
	}
	return albums, count, nil
}

func (r *MyPhotoAlbumRepository) ListOptions(ctx context.Context) ([]port.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.options")
	if err != nil {
		return nil, err
	}
	var albums []row.TPhotoAlbum
	if err := session.Where("is_delete = ?", 0).OrderBy("id DESC").Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.options", err)
	}
	return row.FromPhotoAlbums(albums), nil
}

func (r *MyPhotoAlbumRepository) Get(ctx context.Context, id int) (port.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.get")
	if err != nil {
		return port.TPhotoAlbum{}, err
	}
	var album row.TPhotoAlbum
	found, err := session.ID(id).Get(&album)
	if err != nil {
		return port.TPhotoAlbum{}, apperrors.Unavailable("photo_album.get", err)
	}
	if !found {
		return port.TPhotoAlbum{}, apperrors.NotFound("photo_album.get")
	}
	return row.FromPhotoAlbum(album), nil
}

func (r *MyPhotoAlbumRepository) SaveOrUpdate(ctx context.Context, album port.TPhotoAlbum) error {
	return repoTx(r.engine, ctx, "photo_album.save", func(session *xorm.Session) error {
		albumRow := row.ToPhotoAlbum(album)
		if album.Id == 0 {
			_, err := session.Insert(&albumRow)
			return err
		}
		_, err := session.ID(album.Id).Update(&albumRow)
		return err
	})
}

func (r *MyPhotoAlbumRepository) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "photo_album.delete", func(session *xorm.Session) error {
		_, err := session.ID(id).Delete(&row.TPhotoAlbum{})
		return err
	})
}

type MyPhotoRepository struct{ engine *xorm.Engine }

func NewPhotoRepository(engine *xorm.Engine) *MyPhotoRepository {
	return &MyPhotoRepository{engine: engine}
}

func (r *MyPhotoRepository) List(ctx context.Context, current, size, albumID, isDelete int) ([]port.TPhoto, int64, error) {
	session, err := repoSession(r.engine, ctx, "photo.list")
	if err != nil {
		return nil, 0, err
	}
	where := " WHERE is_delete = ?"
	args := []interface{}{isDelete}
	if albumID != 0 {
		where += " AND album_id = ?"
		args = append(args, albumID)
	}
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_photo"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("photo.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var photos []row.TPhoto
	if err := session.SQL("SELECT * FROM t_photo"+where+" ORDER BY id, update_time DESC LIMIT ? OFFSET ?", args...).Find(&photos); err != nil {
		return nil, 0, apperrors.Unavailable("photo.list", err)
	}
	return row.FromPhotos(photos), count, nil
}

func (r *MyPhotoRepository) Update(ctx context.Context, photo port.TPhoto) error {
	return repoTx(r.engine, ctx, "photo.update", func(session *xorm.Session) error {
		photoRow := row.ToPhoto(photo)
		_, err := session.ID(photo.Id).Update(&photoRow)
		return err
	})
}

func (r *MyPhotoRepository) InsertMany(ctx context.Context, photos []port.TPhoto) error {
	if len(photos) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.insert", func(session *xorm.Session) error {
		rows := make([]row.TPhoto, 0, len(photos))
		for _, photo := range photos {
			rows = append(rows, row.ToPhoto(photo))
		}
		_, err := session.Insert(&rows)
		return err
	})
}

func (r *MyPhotoRepository) UpdateAlbum(ctx context.Context, ids []int, albumID int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.album", func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.ID(id).MustCols("album_id").Update(&row.TPhoto{Id: id, AlbumId: albumID}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *MyPhotoRepository) UpdateDelete(ctx context.Context, ids []int, isDelete int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.delete_flag", func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.ID(id).MustCols("is_delete").Update(&row.TPhoto{Id: id, IsDelete: isDelete}); err != nil {
				return err
			}
		}
		if isDelete == 0 {
			var albums []row.TPhoto
			if err := session.Select("album_id").In("id", ids).GroupBy("album_id").Find(&albums); err != nil {
				return err
			}
			for _, album := range albums {
				if _, err := session.ID(album.AlbumId).MustCols("is_delete").Update(&row.TPhotoAlbum{Id: album.AlbumId, IsDelete: 0}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *MyPhotoRepository) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.delete", func(session *xorm.Session) error {
		_, err := session.In("id", ids).Delete(&row.TPhoto{})
		return err
	})
}

func (r *MyPhotoRepository) ListPublicByAlbum(ctx context.Context, albumID, current, size int) ([]port.TPhoto, error) {
	session, err := repoSession(r.engine, ctx, "photo.public")
	if err != nil {
		return nil, err
	}
	limit, offset := pgsql.Page(current, size)
	var photos []row.TPhoto
	if err := session.SQL("SELECT photo_src FROM t_photo WHERE album_id = ? AND is_delete = 0 ORDER BY id DESC LIMIT ? OFFSET ?", albumID, limit, offset).Find(&photos); err != nil {
		return nil, apperrors.Unavailable("photo.public", err)
	}
	return row.FromPhotos(photos), nil
}
