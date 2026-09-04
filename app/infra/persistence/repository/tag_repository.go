package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/row"
	"context"

	"xorm.io/xorm"
)

var _ port.TagRepository = (*MyTagRepo)(nil)

type MyTagRepo struct {
	engine *xorm.Engine
}

func NewTagRepo(engine *xorm.Engine) *MyTagRepo {
	return &MyTagRepo{engine: engine}
}

func (t *MyTagRepo) tagSession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(t.engine, ctx, "tag")
}

func tagFilter(filter port.TagFilter) (string, []interface{}) {
	if filter.Keywords == "" {
		return "", nil
	}
	return " WHERE t.tag_name LIKE ? ESCAPE '\\'", []interface{}{pgsql.ContainsPattern(filter.Keywords)}
}

func (t *MyTagRepo) List(ctx context.Context) ([]*port.Tag, error) {
	session, err := t.tagSession(ctx)
	if err != nil {
		return nil, err
	}
	var tags []*port.Tag
	if err := session.SQL(pgsql.ListTags).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "tag.list", err)
	}
	return tags, nil
}

func (t *MyTagRepo) ListTopTen(ctx context.Context) ([]*port.Tag, error) {
	session, err := t.tagSession(ctx)
	if err != nil {
		return nil, err
	}
	var tags []*port.Tag
	if err := session.SQL(pgsql.ListTopTenTags).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "tag.list_top", err)
	}
	return tags, nil
}

func (t *MyTagRepo) ListNamesByArticleID(ctx context.Context, articleID int) ([]string, error) {
	session, err := t.tagSession(ctx)
	if err != nil {
		return nil, err
	}
	var names []string
	if err := session.SQL("SELECT tag_name FROM t_tag t JOIN t_article_tag at ON t.id = at.tag_id WHERE article_id = ?", articleID).Find(&names); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "tag.list_names", err)
	}
	return names, nil
}

func (t *MyTagRepo) CountAdmin(ctx context.Context, filter port.TagFilter) (int64, error) {
	session, err := t.tagSession(ctx)
	if err != nil {
		return 0, err
	}
	where, args := tagFilter(filter)
	var count int64
	if _, err := session.SQL("SELECT count(1) FROM t_tag t"+where, args...).Get(&count); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "tag.count_admin", err)
	}
	return count, nil
}

func (t *MyTagRepo) ListAdmin(ctx context.Context, current, size int, filter port.TagFilter) ([]*port.TagAdmin, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := t.tagSession(ctx)
	if err != nil {
		return nil, err
	}
	where, args := tagFilter(filter)
	query := "SELECT t.id, tag_name, COUNT(at.article_id) AS article_count, t.create_time FROM t_tag t LEFT JOIN t_article_tag at ON t.id = at.tag_id" + where + " GROUP BY t.id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	var tags []*port.TagAdmin
	if err := session.SQL(query, args...).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "tag.list_admin", err)
	}
	return tags, nil
}

func (t *MyTagRepo) Search(ctx context.Context, keywords string) ([]*port.TagAdmin, error) {
	session, err := t.tagSession(ctx)
	if err != nil {
		return nil, err
	}
	var tags []*port.TagAdmin
	if err := session.SQL("SELECT id, tag_name, 0 AS article_count, create_time FROM t_tag WHERE tag_name LIKE ? ESCAPE '\\' ORDER BY id DESC", pgsql.ContainsPattern(keywords)).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "tag.search", err)
	}
	return tags, nil
}

func (t *MyTagRepo) SaveOrUpdate(ctx context.Context, tag port.TTag) error {
	session, err := t.tagSession(ctx)
	if err != nil {
		return err
	}
	var existing row.TTag
	found, err := session.Select("id").Where("tag_name = ?", tag.TagName).Get(&existing)
	if err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "tag.check_name", err)
	}
	if found && existing.Id != tag.Id {
		return apperrors.Conflict("tag.save", "tag name already exists")
	}
	return repoTx(t.engine, ctx, "tag.save", func(tx *xorm.Session) error {
		var err error
		tagRow := row.ToTag(tag)
		if tag.Id != 0 {
			_, err = tx.ID(tag.Id).Update(&tagRow)
		} else {
			_, err = tx.Insert(&tagRow)
		}
		if err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "tag.save", err)
		}
		return nil
	})
}

func (t *MyTagRepo) Delete(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	session, err := t.tagSession(ctx)
	if err != nil {
		return err
	}
	var count int64
	if count, err = session.In("tag_id", ids).Count(&row.TArticleTag{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "tag.check_delete", err)
	}
	if count > 0 {
		return apperrors.Conflict("tag.delete", "tag has articles")
	}
	if _, err := session.In("id", ids).Delete(&row.TTag{}); err != nil {
		return apperrors.Wrap(apperrors.KindUnavailable, "tag.delete", err)
	}
	return nil
}
