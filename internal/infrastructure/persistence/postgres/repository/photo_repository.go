package repository

import (
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"strings"

	"xorm.io/xorm"
)

var _ port.PhotoAlbumRepository = (*MyPhotoAlbumRepository)(nil)
var _ port.PhotoRepository = (*MyPhotoRepository)(nil)

type MyPhotoAlbumRepository struct{ engine *xorm.Engine }

func NewPhotoAlbumRepository(engine *xorm.Engine) *MyPhotoAlbumRepository {
	return &MyPhotoAlbumRepository{engine: engine}
}

func (r *MyPhotoAlbumRepository) ListPublic(ctx context.Context) ([]entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.public")
	if err != nil {
		return nil, err
	}
	var albums []entity.TPhotoAlbum
	if err := session.Where("status = ? AND is_delete = ?", 1, 0).OrderBy("id DESC").Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.public", err)
	}
	return albums, nil
}

func (r *MyPhotoAlbumRepository) ListPublicByHandle(ctx context.Context, handle string) ([]entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.author_public")
	if err != nil {
		return nil, err
	}
	var albums []entity.TPhotoAlbum
	query := `SELECT a.* FROM t_photo_album a
		JOIN t_user_info u ON u.id = a.user_id AND u.is_disable = 0
		WHERE lower(u.handle) = lower(?) AND a.status = 1 AND a.is_delete = 0 ORDER BY a.id DESC`
	if err := session.SQL(query, strings.TrimSpace(handle)).Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.author_public", err)
	}
	return albums, nil
}

func (r *MyPhotoAlbumRepository) GetPublicByHandle(ctx context.Context, handle string, albumID int) (entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.author_get")
	if err != nil {
		return entity.TPhotoAlbum{}, err
	}
	var album entity.TPhotoAlbum
	found, err := session.SQL(`SELECT a.* FROM t_photo_album a
		JOIN t_user_info u ON u.id = a.user_id AND u.is_disable = 0
		WHERE lower(u.handle) = lower(?) AND a.id = ? AND a.status = 1 AND a.is_delete = 0`,
		strings.TrimSpace(handle), albumID).Get(&album)
	if err != nil {
		return entity.TPhotoAlbum{}, apperrors.Unavailable("photo_album.author_get", err)
	}
	if !found {
		return entity.TPhotoAlbum{}, apperrors.NotFound("photo_album.author_get")
	}
	return album, nil
}

func (r *MyPhotoAlbumRepository) ListOwned(ctx context.Context, userID int) ([]entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.owned_list")
	if err != nil {
		return nil, err
	}
	var albums []entity.TPhotoAlbum
	if err := session.Where("user_id = ? AND is_delete = 0", userID).OrderBy("id DESC").Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.owned_list", err)
	}
	return albums, nil
}

func (r *MyPhotoAlbumRepository) GetOwned(ctx context.Context, id, userID int) (entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.owned_get")
	if err != nil {
		return entity.TPhotoAlbum{}, err
	}
	var album entity.TPhotoAlbum
	found, err := session.Where("user_id = ? AND is_delete = 0", userID).ID(id).Get(&album)
	if err != nil {
		return entity.TPhotoAlbum{}, apperrors.Unavailable("photo_album.owned_get", err)
	}
	if !found {
		return entity.TPhotoAlbum{}, apperrors.NotFound("photo_album.owned_get")
	}
	return album, nil
}

func (r *MyPhotoAlbumRepository) SaveOwned(ctx context.Context, album entity.TPhotoAlbum, userID int) error {
	return repoTx(r.engine, ctx, "photo_album.owned_save", func(session *xorm.Session) error {
		album.UserId = userID
		if album.Id == 0 {
			album.IsDelete = 0
			_, err := session.Insert(&album)
			return err
		}
		result, err := session.Where("user_id = ? AND is_delete = 0", userID).ID(album.Id).Cols("album_name", "album_desc", "album_cover", "status").Update(&album)
		if err != nil {
			return err
		}
		if result == 0 {
			return apperrors.NotFound("photo_album.owned_save")
		}
		return nil
	})
}

func (r *MyPhotoAlbumRepository) DeleteOwned(ctx context.Context, id, userID int) error {
	return repoTx(r.engine, ctx, "photo_album.owned_delete", func(session *xorm.Session) error {
		result, err := session.Where("user_id = ? AND is_delete = 0", userID).ID(id).Cols("is_delete").Update(&entity.TPhotoAlbum{IsDelete: 1})
		if err != nil {
			return err
		}
		if result == 0 {
			return apperrors.NotFound("photo_album.owned_delete")
		}
		return nil
	})
}

func (r *MyPhotoAlbumRepository) FindByName(ctx context.Context, name string) (entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.find_name")
	if err != nil {
		return entity.TPhotoAlbum{}, err
	}
	var album entity.TPhotoAlbum
	found, err := session.Select("id, album_name").Where("album_name = ?", name).Get(&album)
	if err != nil {
		return entity.TPhotoAlbum{}, apperrors.Unavailable("photo_album.find_name", err)
	}
	if !found {
		return entity.TPhotoAlbum{}, nil
	}
	return album, nil
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

func (r *MyPhotoAlbumRepository) ListOptions(ctx context.Context) ([]entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.options")
	if err != nil {
		return nil, err
	}
	var albums []entity.TPhotoAlbum
	if err := session.Where("is_delete = ?", 0).OrderBy("id DESC").Find(&albums); err != nil {
		return nil, apperrors.Unavailable("photo_album.options", err)
	}
	return albums, nil
}

func (r *MyPhotoAlbumRepository) Get(ctx context.Context, id int) (entity.TPhotoAlbum, error) {
	session, err := repoSession(r.engine, ctx, "photo_album.get")
	if err != nil {
		return entity.TPhotoAlbum{}, err
	}
	var album entity.TPhotoAlbum
	found, err := session.ID(id).Get(&album)
	if err != nil {
		return entity.TPhotoAlbum{}, apperrors.Unavailable("photo_album.get", err)
	}
	if !found {
		return entity.TPhotoAlbum{}, apperrors.NotFound("photo_album.get")
	}
	return album, nil
}

func (r *MyPhotoAlbumRepository) SaveOrUpdate(ctx context.Context, album entity.TPhotoAlbum) error {
	return repoTx(r.engine, ctx, "photo_album.save", func(session *xorm.Session) error {
		if album.Id == 0 {
			_, err := session.Insert(&album)
			return err
		}
		_, err := session.ID(album.Id).Update(&album)
		return err
	})
}

func (r *MyPhotoAlbumRepository) Delete(ctx context.Context, id int) error {
	return repoTx(r.engine, ctx, "photo_album.delete", func(session *xorm.Session) error {
		_, err := session.ID(id).Delete(&entity.TPhotoAlbum{})
		return err
	})
}

type MyPhotoRepository struct{ engine *xorm.Engine }

func NewPhotoRepository(engine *xorm.Engine) *MyPhotoRepository {
	return &MyPhotoRepository{engine: engine}
}

func (r *MyPhotoRepository) List(ctx context.Context, current, size, albumID, isDelete int, keywords string) ([]entity.TPhoto, int64, error) {
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
	if strings.TrimSpace(keywords) != "" {
		where += " AND (photo_name LIKE ? ESCAPE '\\' OR COALESCE(photo_desc, '') LIKE ? ESCAPE '\\')"
		pattern := pgsql.ContainsPattern(keywords)
		args = append(args, pattern, pattern)
	}
	var count int64
	if _, err := session.SQL("SELECT count(0) FROM t_photo"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("photo.count", err)
	}
	limit, offset := pgsql.Page(current, size)
	args = append(args, limit, offset)
	var photos []entity.TPhoto
	if err := session.SQL("SELECT * FROM t_photo"+where+" ORDER BY id, update_time DESC LIMIT ? OFFSET ?", args...).Find(&photos); err != nil {
		return nil, 0, apperrors.Unavailable("photo.list", err)
	}
	return photos, count, nil
}

func (r *MyPhotoRepository) ListOwnedByAlbum(ctx context.Context, userID, albumID int) ([]entity.TPhoto, error) {
	session, err := repoSession(r.engine, ctx, "photo.owned_list")
	if err != nil {
		return nil, err
	}
	var photos []entity.TPhoto
	if err := session.SQL(`SELECT p.* FROM t_photo p
		JOIN t_photo_album a ON a.id = p.album_id
		WHERE a.user_id = ? AND a.id = ? AND a.is_delete = 0 AND p.is_delete = 0
		ORDER BY p.id DESC`, userID, albumID).Find(&photos); err != nil {
		return nil, apperrors.Unavailable("photo.owned_list", err)
	}
	return photos, nil
}

func (r *MyPhotoRepository) InsertOwned(ctx context.Context, userID, albumID int, photos []entity.TPhoto) error {
	if len(photos) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.owned_insert", func(session *xorm.Session) error {
		var albumIDFound int
		found, err := session.SQL("SELECT id FROM t_photo_album WHERE id = ? AND user_id = ? AND is_delete = 0", albumID, userID).Get(&albumIDFound)
		if err != nil {
			return err
		}
		if !found {
			return apperrors.NotFound("photo.owned_insert")
		}
		for i := range photos {
			photos[i].AlbumId = albumID
			photos[i].IsDelete = 0
		}
		_, err = session.Insert(&photos)
		return err
	})
}

func (r *MyPhotoRepository) DeleteOwned(ctx context.Context, userID int, photoIDs []int) error {
	if len(photoIDs) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.owned_delete", func(session *xorm.Session) error {
		result, err := session.SQL(`DELETE FROM t_photo WHERE id IN (`+placeholders(len(photoIDs))+`)
			AND album_id IN (SELECT id FROM t_photo_album WHERE user_id = ? AND is_delete = 0)`, append(intArgs(photoIDs), userID)...).Exec()
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != int64(len(photoIDs)) {
			return apperrors.NotFound("photo.owned_delete")
		}
		return nil
	})
}

func (r *MyPhotoRepository) Update(ctx context.Context, photo entity.TPhoto) error {
	return repoTx(r.engine, ctx, "photo.update", func(session *xorm.Session) error {
		_, err := session.ID(photo.Id).Update(&photo)
		return err
	})
}

func (r *MyPhotoRepository) InsertMany(ctx context.Context, photos []entity.TPhoto) error {
	if len(photos) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.insert", func(session *xorm.Session) error {
		_, err := session.Insert(&photos)
		return err
	})
}

func (r *MyPhotoRepository) UpdateAlbum(ctx context.Context, ids []int, albumID int) error {
	if len(ids) == 0 {
		return nil
	}
	return repoTx(r.engine, ctx, "photo.album", func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.ID(id).MustCols("album_id").Update(&entity.TPhoto{Id: id, AlbumId: albumID}); err != nil {
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
			if _, err := session.ID(id).MustCols("is_delete").Update(&entity.TPhoto{Id: id, IsDelete: isDelete}); err != nil {
				return err
			}
		}
		if isDelete == 0 {
			var albums []entity.TPhoto
			if err := session.Select("album_id").In("id", ids).GroupBy("album_id").Find(&albums); err != nil {
				return err
			}
			for _, album := range albums {
				if _, err := session.ID(album.AlbumId).MustCols("is_delete").Update(&entity.TPhotoAlbum{Id: album.AlbumId, IsDelete: 0}); err != nil {
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
		_, err := session.In("id", ids).Delete(&entity.TPhoto{})
		return err
	})
}

func (r *MyPhotoRepository) ListPublicByAlbum(ctx context.Context, albumID, current, size int) ([]entity.TPhoto, error) {
	session, err := repoSession(r.engine, ctx, "photo.public")
	if err != nil {
		return nil, err
	}
	limit, offset := pgsql.Page(current, size)
	var photos []entity.TPhoto
	if err := session.SQL("SELECT photo_src FROM t_photo WHERE album_id = ? AND is_delete = 0 ORDER BY id DESC LIMIT ? OFFSET ?", albumID, limit, offset).Find(&photos); err != nil {
		return nil, apperrors.Unavailable("photo.public", err)
	}
	return photos, nil
}
