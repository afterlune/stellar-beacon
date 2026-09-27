package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

func (r *MyPlatformRepo) ListOwnedArticles(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.ArticleAdmin, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.articles")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "a")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article a WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.articles.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	order := "a.update_time DESC NULLS LAST, a.id DESC"
	if filter.SeriesID > 0 {
		order = "a.series_order ASC, a.id ASC"
	} else if filter.Status == 4 {
		order = "a.scheduled_at ASC NULLS LAST, a.id ASC"
	}
	var articles []*port.ArticleAdmin
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, a.series_id, a.series_order, a.scheduled_at,
		       a.is_top, a.is_featured, a.is_delete, a.status, a.moderation_status, a.moderation_reason,
		       a.type, a.create_time, c.category_name
		FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id
		WHERE `+where+` ORDER BY `+order+` LIMIT ? OFFSET ?`, pageArgs...).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.articles.list", err)
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) GetOwnedArticle(ctx context.Context, userID, articleID int) (port.ArticleAdminView, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.article")
	if err != nil {
		return port.ArticleAdminView{}, err
	}
	var article entity.TArticle
	found, err := session.Where("id = ? AND user_id = ?", articleID, userID).Get(&article)
	if err != nil {
		return port.ArticleAdminView{}, apperrors.Unavailable("platform.studio.article.get", err)
	}
	if !found {
		return port.ArticleAdminView{}, apperrors.NotFound("platform.studio.article.get")
	}
	categoryName := ""
	if article.CategoryId > 0 {
		var category entity.TCategory
		if _, err := session.Where("id = ? AND user_id = ?", article.CategoryId, userID).Get(&category); err == nil {
			categoryName = category.CategoryName
		}
	}
	var tagIDs []int
	if err := session.SQL(`SELECT tag_id FROM t_article_tag WHERE article_id = ?`, articleID).Find(&tagIDs); err != nil {
		return port.ArticleAdminView{}, apperrors.Unavailable("platform.studio.article.tags", err)
	}
	scheduledAt := ""
	if !article.ScheduledAt.IsZero() {
		scheduledAt = article.ScheduledAt.Format(time.RFC3339)
	}
	return port.ArticleAdminView{
		Id: article.Id, UserId: article.UserId, ArticleCover: article.ArticleCover, ArticleTitle: article.ArticleTitle,
		ArticleContent: article.ArticleContent, ArticleContentHTML: article.ArticleContentHTML, IsTop: article.IsTop,
		IsFeatured: article.IsFeatured, CategoryId: article.CategoryId, CategoryName: categoryName, TagNames: tagIDs, Status: article.Status,
		Type: article.Type, Password: article.Password, OriginalUrl: article.OriginalUrl, SeriesId: article.SeriesId,
		SeriesOrder: article.SeriesOrder, ScheduledAt: scheduledAt,
		ModerationStatus: article.ModerationStatus, ModerationReason: article.ModerationReason,
	}, nil
}

func (r *MyPlatformRepo) SaveOwnedArticle(ctx context.Context, userID int, article entity.TArticle, categoryID int, tagIDs []int) (entity.TArticle, error) {
	wasPublic := false
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if categoryID > 0 {
			ok, err := session.Where("id = ? AND user_id = ?", categoryID, userID).Exist(&entity.TCategory{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.category", err)
			}
			if !ok {
				return apperrors.Invalid("platform.studio.article.category", "category does not belong to author")
			}
			article.CategoryId = categoryID
		} else {
			article.CategoryId = 0
		}
		if article.SeriesId > 0 {
			ok, err := session.Where("id = ? AND user_id = ? AND is_delete = 0", article.SeriesId, userID).Exist(&entity.TSeries{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.series", err)
			}
			if !ok {
				return apperrors.Invalid("platform.studio.article.series", "series does not belong to author")
			}
		}
		uniqueTags := uniqueInts(tagIDs)
		if len(uniqueTags) > 0 {
			count, err := session.Where("user_id = ?", userID).In("id", uniqueTags).Count(&entity.TTag{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.tags", err)
			}
			if int(count) != len(uniqueTags) {
				return apperrors.Invalid("platform.studio.article.tags", "tag does not belong to author")
			}
		}
		article.UserId = userID
		if article.Status == 0 {
			article.Status = 3
		}
		if article.Id == 0 {
			if article.ModerationStatus == "" {
				article.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&article); err != nil {
				return apperrors.Unavailable("platform.studio.article.create", err)
			}
		} else {
			var existing entity.TArticle
			found, err := session.Where("id = ? AND user_id = ?", article.Id, userID).Get(&existing)
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.owner", err)
			}
			if !found {
				return apperrors.NotFound("platform.studio.article.update")
			}
			wasPublic = existing.Status == 1 && existing.IsDelete == 0 && existing.ModerationStatus == "visible"
			if _, err := session.ID(article.Id).Where("user_id = ?", userID).Cols(
				"category_id", "article_cover", "article_title", "article_content", "article_content_html",
				"series_id", "series_order", "scheduled_at", "status", "type", "password", "original_url", "update_time",
			).Update(&article); err != nil {
				return apperrors.Unavailable("platform.studio.article.update", err)
			}
		}
		if _, err := session.Where("article_id = ?", article.Id).Delete(&entity.TArticleTag{}); err != nil {
			return apperrors.Unavailable("platform.studio.article.tags.clear", err)
		}
		for _, tagID := range uniqueTags {
			if _, err := session.Insert(&entity.TArticleTag{ArticleId: article.Id, TagId: tagID}); err != nil {
				return apperrors.Unavailable("platform.studio.article.tags.save", err)
			}
		}
		if !wasPublic {
			if err := recordArticlePublishEvent(session, article.Id, time.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return entity.TArticle{}, err
	}
	return r.getArticleRecord(ctx, article.Id)
}

func (r *MyPlatformRepo) getArticleRecord(ctx context.Context, id int) (entity.TArticle, error) {
	session, err := repoSession(r.engine, ctx, "platform.article.record")
	if err != nil {
		return entity.TArticle{}, err
	}
	var article entity.TArticle
	found, err := session.ID(id).Get(&article)
	if err != nil {
		return entity.TArticle{}, apperrors.Unavailable("platform.article.record", err)
	}
	if !found {
		return entity.TArticle{}, apperrors.NotFound("platform.article.record")
	}
	return article, nil
}

func (r *MyPlatformRepo) DeleteOwnedArticles(ctx context.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		for _, id := range uniqueInts(ids) {
			found, err := session.Where("id = ? AND user_id = ?", id, userID).Exist(&entity.TArticle{})
			if err != nil {
				return apperrors.Unavailable("platform.studio.article.delete.owner", err)
			}
			if !found {
				return apperrors.NotFound("platform.studio.article.delete")
			}
			if _, err := session.Where("article_id = ?", id).Delete(&entity.TArticleTag{}); err != nil {
				return apperrors.Unavailable("platform.studio.article.delete.tags", err)
			}
			if _, err := session.Where("id = ?", id).Delete(&entity.TArticle{}); err != nil {
				return apperrors.Unavailable("platform.studio.article.delete", err)
			}
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedTalks(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.TalkAdmin, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.talks")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "t")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk t WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.talks.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var talks []*port.TalkAdmin
	if err := session.SQL(`
		SELECT t.id, t.user_id, ui.nickname, ui.avatar, t.content, t.images, t.is_top, t.status,
		       t.moderation_status, t.moderation_reason, t.create_time
		FROM t_talk t JOIN t_user_info ui ON t.user_id = ui.id
		WHERE `+where+` ORDER BY t.id DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) GetOwnedTalk(ctx context.Context, userID, talkID int) (entity.TTalk, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.talk")
	if err != nil {
		return entity.TTalk{}, err
	}
	var talk entity.TTalk
	found, err := session.Where("id = ? AND user_id = ?", talkID, userID).Get(&talk)
	if err != nil {
		return entity.TTalk{}, apperrors.Unavailable("platform.studio.talk.get", err)
	}
	if !found {
		return entity.TTalk{}, apperrors.NotFound("platform.studio.talk.get")
	}
	return talk, nil
}

func (r *MyPlatformRepo) SaveOwnedTalk(ctx context.Context, talk entity.TTalk) (entity.TTalk, error) {
	wasPublic := false
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if talk.Status == 0 {
			talk.Status = 3
		}
		if talk.Id == 0 {
			if talk.ModerationStatus == "" {
				talk.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&talk); err != nil {
				return apperrors.Unavailable("platform.studio.talk.create", err)
			}
			return recordTalkPublishEvent(session, talk.Id, time.Now())
		}
		var existing entity.TTalk
		found, err := session.Where("id = ? AND user_id = ?", talk.Id, talk.UserId).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.talk.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.talk.update")
		}
		wasPublic = existing.Status == 1 && existing.ModerationStatus == "visible"
		if _, err := session.ID(talk.Id).Where("user_id = ?", talk.UserId).Cols(
			"content", "images", "is_top", "status", "update_time",
		).Update(&talk); err != nil {
			return apperrors.Unavailable("platform.studio.talk.update", err)
		}
		if !wasPublic {
			if err := recordTalkPublishEvent(session, talk.Id, time.Now()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return entity.TTalk{}, err
	}
	return talk, nil
}
func (r *MyPlatformRepo) DeleteOwnedTalks(ctx context.Context, userID int, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range uniqueInts(ids) {
		if _, err := r.GetOwnedTalk(ctx, userID, id); err != nil {
			return err
		}
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Where("user_id = ?", userID).In("id", uniqueInts(ids)).Delete(&entity.TTalk{}); err != nil {
			return apperrors.Unavailable("platform.studio.talks.delete", err)
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedSeries(ctx context.Context, userID int, filter port.StudioFilter) ([]*port.Series, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.series")
	if err != nil {
		return nil, 0, err
	}
	where, args := ownedFilter(userID, filter, "s")
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_series s WHERE `+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.series.count", err)
	}
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	pageArgs := append(append([]interface{}{}, args...), limit, offset)
	var series []*port.Series
	if err := session.SQL(`
		SELECT s.id, s.user_id, s.series_name, s.series_desc, s.cover, s.status, s.moderation_status, s.moderation_reason, s.update_time,
		       count(a.id) AS article_count
		FROM t_series s LEFT JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0
		WHERE `+where+` GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&series); err != nil {
		return nil, 0, apperrors.Unavailable("platform.studio.series.list", err)
	}
	return series, count, nil
}

func (r *MyPlatformRepo) GetOwnedSeries(ctx context.Context, userID, seriesID int) (entity.TSeries, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.series.get")
	if err != nil {
		return entity.TSeries{}, err
	}
	var series entity.TSeries
	found, err := session.Where("id = ? AND user_id = ? AND is_delete = 0", seriesID, userID).Get(&series)
	if err != nil {
		return entity.TSeries{}, apperrors.Unavailable("platform.studio.series.get", err)
	}
	if !found {
		return entity.TSeries{}, apperrors.NotFound("platform.studio.series.get")
	}
	return series, nil
}

func (r *MyPlatformRepo) SaveOwnedSeries(ctx context.Context, series entity.TSeries) (entity.TSeries, error) {
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if series.Id == 0 {
			if series.Status == 0 {
				series.Status = 1
			}
			if series.ModerationStatus == "" {
				series.ModerationStatus = "visible"
			}
			if _, err := session.Insert(&series); err != nil {
				return apperrors.Unavailable("platform.studio.series.create", err)
			}
			return nil
		}
		var existing entity.TSeries
		found, err := session.Where("id = ? AND user_id = ?", series.Id, series.UserId).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.series.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.series.update")
		}
		if _, err := session.ID(series.Id).Where("user_id = ?", series.UserId).Cols(
			"series_name", "series_desc", "cover", "status", "update_time",
		).Update(&series); err != nil {
			return apperrors.Unavailable("platform.studio.series.update", err)
		}
		return nil
	})
	if err != nil {
		return entity.TSeries{}, err
	}
	return series, nil
}

func (r *MyPlatformRepo) DeleteOwnedSeries(ctx context.Context, userID, seriesID int) error {
	if _, err := r.GetOwnedSeries(ctx, userID, seriesID); err != nil {
		return err
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Exec(`UPDATE t_article SET series_id = NULL, series_order = 0 WHERE series_id = ? AND user_id = ?`, seriesID, userID); err != nil {
			return apperrors.Unavailable("platform.studio.series.detach", err)
		}
		if _, err := session.Where("id = ? AND user_id = ?", seriesID, userID).Cols("is_delete", "update_time").Update(&entity.TSeries{IsDelete: 1, UpdateTime: time.Now()}); err != nil {
			return apperrors.Unavailable("platform.studio.series.delete", err)
		}
		return nil
	})
}

func questionMarks(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func (r *MyPlatformRepo) ListOwnedCategories(ctx context.Context, userID int) ([]*port.Category, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.categories")
	if err != nil {
		return nil, err
	}
	var categories []*port.Category
	if err := session.SQL(`
		SELECT c.id, c.user_id, c.category_name, count(a.id) AS article_count
		FROM t_category c LEFT JOIN t_article a ON a.category_id = c.id AND a.is_delete = 0
		WHERE c.user_id = ? GROUP BY c.id ORDER BY c.category_name`, userID).Find(&categories); err != nil {
		return nil, apperrors.Unavailable("platform.studio.categories.list", err)
	}
	return categories, nil
}

func (r *MyPlatformRepo) SaveOwnedCategory(ctx context.Context, category entity.TCategory) (entity.TCategory, error) {
	if strings.TrimSpace(category.CategoryName) == "" {
		return entity.TCategory{}, apperrors.Invalid("platform.studio.category", "name is required")
	}
	category.CategoryName = strings.TrimSpace(category.CategoryName)
	var result entity.TCategory
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TCategory
		found, err := session.Where("user_id = ? AND lower(btrim(category_name)) = lower(btrim(?))", category.UserId, category.CategoryName).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.unique", err)
		}
		if found && existing.Id != category.Id {
			return apperrors.Conflict("platform.studio.category", "category already exists")
		}
		if category.Id == 0 {
			if _, err := session.Insert(&category); err != nil {
				return apperrors.Unavailable("platform.studio.category.create", err)
			}
			result = category
			return nil
		}
		var current entity.TCategory
		found, err = session.Where("id = ? AND user_id = ?", category.Id, category.UserId).Get(&current)
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.category")
		}
		if _, err := session.ID(category.Id).Where("user_id = ?", category.UserId).Cols("category_name").Update(&category); err != nil {
			return apperrors.Unavailable("platform.studio.category.update", err)
		}
		result = category
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) DeleteOwnedCategory(ctx context.Context, userID, categoryID int) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		count, err := session.Where("category_id = ? AND user_id = ? AND is_delete = 0", categoryID, userID).Count(&entity.TArticle{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.in_use", err)
		}
		if count > 0 {
			return apperrors.Conflict("platform.studio.category.delete", "category is in use")
		}
		affected, err := session.Where("id = ? AND user_id = ?", categoryID, userID).Delete(&entity.TCategory{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.category.delete", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.studio.category.delete")
		}
		return nil
	})
}

func (r *MyPlatformRepo) ListOwnedTags(ctx context.Context, userID int) ([]*port.Tag, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.tags")
	if err != nil {
		return nil, err
	}
	var tags []*port.Tag
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.tag_name, count(at.article_id) AS count
		FROM t_tag t
		LEFT JOIN t_article_tag at ON at.tag_id = t.id
		LEFT JOIN t_article a ON a.id = at.article_id AND a.is_delete = 0
		WHERE t.user_id = ? GROUP BY t.id ORDER BY t.tag_name`, userID).Find(&tags); err != nil {
		return nil, apperrors.Unavailable("platform.studio.tags.list", err)
	}
	return tags, nil
}

func (r *MyPlatformRepo) SaveOwnedTag(ctx context.Context, tag entity.TTag) (entity.TTag, error) {
	if strings.TrimSpace(tag.TagName) == "" {
		return entity.TTag{}, apperrors.Invalid("platform.studio.tag", "name is required")
	}
	tag.TagName = strings.TrimSpace(tag.TagName)
	var result entity.TTag
	err := ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TTag
		found, err := session.Where("user_id = ? AND lower(btrim(tag_name)) = lower(btrim(?))", tag.UserId, tag.TagName).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.unique", err)
		}
		if found && existing.Id != tag.Id {
			return apperrors.Conflict("platform.studio.tag", "tag already exists")
		}
		if tag.Id == 0 {
			if _, err := session.Insert(&tag); err != nil {
				return apperrors.Unavailable("platform.studio.tag.create", err)
			}
			result = tag
			return nil
		}
		var current entity.TTag
		found, err = session.Where("id = ? AND user_id = ?", tag.Id, tag.UserId).Get(&current)
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.tag")
		}
		if _, err := session.ID(tag.Id).Where("user_id = ?", tag.UserId).Cols("tag_name").Update(&tag); err != nil {
			return apperrors.Unavailable("platform.studio.tag.update", err)
		}
		result = tag
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) DeleteOwnedTag(ctx context.Context, userID, tagID int) error {
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		count, err := session.Where("tag_id = ?", tagID).Count(&entity.TArticleTag{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.in_use", err)
		}
		if count > 0 {
			return apperrors.Conflict("platform.studio.tag.delete", "tag is in use")
		}
		affected, err := session.Where("id = ? AND user_id = ?", tagID, userID).Delete(&entity.TTag{})
		if err != nil {
			return apperrors.Unavailable("platform.studio.tag.delete", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.studio.tag.delete")
		}
		return nil
	})
}

func (r *MyPlatformRepo) ModerateContent(ctx context.Context, contentType string, id, adminID int, hidden bool, reason string) error {
	table := ""
	switch contentType {
	case "article":
		table = "t_article"
	case "talk":
		table = "t_talk"
	case "series":
		table = "t_series"
	case "collection":
		table = "t_collection"
	default:
		return apperrors.Invalid("platform.moderation", "unsupported content type")
	}
	status := "visible"
	reason = strings.TrimSpace(reason)
	recommendationReset := ""
	if hidden {
		status = "hidden"
		switch table {
		case "t_article":
			recommendationReset = ", is_top = 0, is_featured = 0"
		case "t_talk":
			recommendationReset = ", is_top = 0"
		}
	} else {
		reason = ""
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		result, err := session.Exec("UPDATE "+table+" SET moderation_status = ?, moderation_reason = ?, moderated_by = ?, moderated_at = CURRENT_TIMESTAMP"+recommendationReset+" WHERE id = ?", status, reason, adminID, id)
		if err != nil {
			return apperrors.Unavailable("platform.moderation.update", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return apperrors.NotFound("platform.moderation.update")
		}
		if !hidden {
			switch contentType {
			case "article":
				return recordArticlePublishEvent(session, id, time.Now())
			case "talk":
				return recordTalkPublishEvent(session, id, time.Now())
			}
		}
		return nil
	})
}

func ownedFilter(userID int, filter port.StudioFilter, alias string) (string, []interface{}) {
	where := alias + ".user_id = ?"
	args := []interface{}{userID}
	if alias == "a" || alias == "s" {
		where += " AND " + alias + ".is_delete = 0"
	}
	if filter.Status > 0 {
		where += " AND " + alias + ".status = ?"
		args = append(args, filter.Status)
	}
	if alias == "a" && filter.SeriesID > 0 {
		where += " AND " + alias + ".series_id = ?"
		args = append(args, filter.SeriesID)
	}
	if strings.TrimSpace(filter.Keywords) != "" {
		column := alias + ".article_title"
		switch alias {
		case "t":
			column = alias + ".content"
		case "s":
			column = alias + ".series_name"
		}
		where += " AND " + column + " LIKE ?"
		args = append(args, "%"+strings.TrimSpace(filter.Keywords)+"%")
	}
	return where, args
}

func uniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func toPublicAuthor(user entity.TUserInfo) port.PublicAuthor {
	return port.PublicAuthor{
		Id:       user.Id,
		Handle:   user.Handle,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Intro:    user.Intro,
		Website:  user.Website,
		About:    user.About,
		Links:    decodeProfileLinks(user.ProfileLinksJSON),
	}
}

func decodeProfileLinks(raw string) []port.ProfileLink {
	var links []port.ProfileLink
	if raw == "" || json.Unmarshal([]byte(raw), &links) != nil {
		return []port.ProfileLink{}
	}
	return links
}
