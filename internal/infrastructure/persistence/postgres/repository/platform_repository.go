package repository

import (
	"context"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"xorm.io/xorm"
)

var _ port.PlatformRepository = (*MyPlatformRepo)(nil)

type MyPlatformRepo struct{ engine *xorm.Engine }

func NewPlatformRepo(engine *xorm.Engine) *MyPlatformRepo { return &MyPlatformRepo{engine: engine} }

func (r *MyPlatformRepo) GetAuthorByHandle(ctx context.Context, handle string) (port.AuthorCard, error) {
	session, err := repoSession(r.engine, ctx, "platform.author")
	if err != nil {
		return port.AuthorCard{}, err
	}
	var user entity.TUserInfo
	found, err := session.SQL(`SELECT * FROM t_user_info WHERE lower(handle) = lower(?) AND is_disable = 0`, strings.TrimSpace(handle)).Get(&user)
	if err != nil {
		return port.AuthorCard{}, apperrors.Unavailable("platform.author.get", err)
	}
	if !found {
		return port.AuthorCard{}, apperrors.NotFound("platform.author.get")
	}
	author := port.AuthorCard{PublicAuthor: toPublicAuthor(user)}
	if err := r.attachAuthorCounts(session, &author); err != nil {
		return port.AuthorCard{}, err
	}
	return author, nil
}

func (r *MyPlatformRepo) attachAuthorCounts(session *xorm.Session, author *port.AuthorCard) error {
	if author == nil || author.Id <= 0 {
		return nil
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.ArticleCount); err != nil {
		return apperrors.Unavailable("platform.author.article_count", err)
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE user_id = ? AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.TalkCount); err != nil {
		return apperrors.Unavailable("platform.author.talk_count", err)
	}
	if _, err := session.SQL(`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, author.Id).Get(&author.SeriesCount); err != nil {
		return apperrors.Unavailable("platform.author.series_count", err)
	}
	return nil
}

func (r *MyPlatformRepo) ListAuthors(ctx context.Context, current, size int) ([]*port.AuthorCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.authors")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`
		SELECT count(1) FROM t_user_info ui
		WHERE ui.is_disable = 0 AND (
			EXISTS (
				SELECT 1 FROM t_article a
				WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_talk t
				WHERE t.user_id = ui.id AND t.status = 1 AND t.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_series s
				WHERE s.user_id = ui.id AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
			)
		)`).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.authors.count", err)
	}
	var users []entity.TUserInfo
	if err := session.SQL(`
		SELECT ui.*
		FROM t_user_info ui
		WHERE ui.is_disable = 0 AND (
			EXISTS (
				SELECT 1 FROM t_article a
				WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_talk t
				WHERE t.user_id = ui.id AND t.status = 1 AND t.moderation_status = 'visible'
			) OR EXISTS (
				SELECT 1 FROM t_series s
				WHERE s.user_id = ui.id AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
			)
		)
		ORDER BY (
			SELECT count(1) FROM t_article a
			WHERE a.user_id = ui.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		) DESC, ui.id ASC LIMIT ? OFFSET ?`, limit, offset).Find(&users); err != nil {
		return nil, 0, apperrors.Unavailable("platform.authors.list", err)
	}
	authors := make([]*port.AuthorCard, 0, len(users))
	for _, user := range users {
		author := &port.AuthorCard{PublicAuthor: toPublicAuthor(user)}
		if err := r.attachAuthorCounts(session, author); err != nil {
			return nil, 0, err
		}
		authors = append(authors, author)
	}
	return authors, count, nil
}

func (r *MyPlatformRepo) StudioDashboard(ctx context.Context, userID int) (port.StudioDashboard, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.dashboard")
	if err != nil {
		return port.StudioDashboard{}, err
	}
	var dashboard port.StudioDashboard
	queries := []struct {
		sql    string
		target *int
	}{
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0`, &dashboard.ArticleCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 3`, &dashboard.DraftCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 2`, &dashboard.PrivateCount},
		{`SELECT count(1) FROM t_talk WHERE user_id = ?`, &dashboard.TalkCount},
		{`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0`, &dashboard.SeriesCount},
		{`SELECT count(1) FROM t_article_reaction WHERE user_info_id = ? AND reaction = 'favorite'`, &dashboard.FavoriteCount},
	}
	for _, item := range queries {
		if _, err := session.SQL(item.sql, userID).Get(item.target); err != nil {
			return port.StudioDashboard{}, apperrors.Unavailable("platform.studio.dashboard", err)
		}
	}
	return dashboard, nil
}

func (r *MyPlatformRepo) UpdateAuthorProfile(ctx context.Context, userID int, handle, nickname, intro, website string) error {
	handle = strings.ToLower(strings.TrimSpace(handle))
	nickname = strings.TrimSpace(nickname)
	if !validPublicHandle(handle) {
		return apperrors.Invalid("platform.profile.handle", "handle is invalid")
	}
	if nickname == "" {
		return apperrors.Invalid("platform.profile.nickname", "nickname is required")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TUserInfo
		found, err := session.Where("lower(handle) = lower(?) AND id <> ?", handle, userID).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.profile.handle.unique", err)
		}
		if found {
			return apperrors.Conflict("platform.profile.handle", "handle already exists")
		}
		affected, err := session.ID(userID).Cols("handle", "nickname", "intro", "website").Update(&entity.TUserInfo{
			Handle: handle, Nickname: nickname, Intro: intro, Website: website,
		})
		if err != nil {
			return apperrors.Unavailable("platform.profile.update", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.profile.update")
		}
		return nil
	})
}

func validPublicHandle(value string) bool {
	if len(value) < 3 || len(value) > 40 {
		return false
	}
	for index, r := range value {
		if index == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
func (r *MyPlatformRepo) ListFeedArticles(ctx context.Context, current, size int, featuredOnly bool) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.feed.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	where := `a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`
	if featuredOnly {
		where += ` AND a.is_featured = 1`
	}
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article a WHERE ` + where).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		LEFT JOIN t_category c ON a.category_id = c.id
		WHERE `+where+`
		ORDER BY a.is_top DESC, a.is_featured DESC, a.create_time DESC, a.id DESC
		LIMIT ? OFFSET ?`, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.articles.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) attachArticleAuthorsAndTags(ctx context.Context, session *xorm.Session, articles []*port.ArticleCard) error {
	if len(articles) == 0 {
		return nil
	}
	userIDs := make([]int, 0, len(articles))
	seen := map[int]struct{}{}
	for _, article := range articles {
		if article.UserId > 0 {
			if _, ok := seen[article.UserId]; !ok {
				seen[article.UserId] = struct{}{}
				userIDs = append(userIDs, article.UserId)
			}
		}
	}
	var users []entity.TUserInfo
	if len(userIDs) > 0 {
		if err := session.In("id", userIDs).Find(&users); err != nil {
			return apperrors.Unavailable("platform.feed.authors", err)
		}
	}
	byID := make(map[int]entity.TUserInfo, len(users))
	for _, user := range users {
		byID[user.Id] = user
	}
	for _, article := range articles {
		article.Author = toPublicAuthor(byID[article.UserId])
		var tags []string
		if err := session.SQL(pgsql.ArticleTags, article.Id).Find(&tags); err != nil {
			return apperrors.Unavailable("platform.feed.tags", err)
		}
		article.Tags = tags
	}
	return nil
}

func (r *MyPlatformRepo) ListFeedTalks(ctx context.Context, current, size int) ([]*port.Talk, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.feed.talks")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE status = 1 AND moderation_status = 'visible'`).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.talks.count", err)
	}
	var talks []*port.Talk
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.content, t.images, t.is_top, t.status, t.moderation_status,
		       t.create_time, u.handle, u.nickname, u.avatar
		FROM t_talk t JOIN t_user_info u ON t.user_id = u.id
		WHERE t.status = 1 AND t.moderation_status = 'visible'
		ORDER BY t.is_top DESC, t.create_time DESC, t.id DESC LIMIT ? OFFSET ?`, limit, offset).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.feed.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) ListAuthorArticles(ctx context.Context, userID, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id
		WHERE a.user_id = ? AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		ORDER BY a.is_top DESC, a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.articles.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

func (r *MyPlatformRepo) ListAuthorTalks(ctx context.Context, userID, current, size int) ([]*port.Talk, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.talks")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_talk WHERE user_id = ? AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.talks.count", err)
	}
	var talks []*port.Talk
	if err := session.SQL(`
		SELECT t.id, t.user_id, t.content, t.images, t.is_top, t.status, t.moderation_status,
		       t.create_time, u.handle, u.nickname, u.avatar
		FROM t_talk t JOIN t_user_info u ON t.user_id = u.id
		WHERE t.user_id = ? AND t.status = 1 AND t.moderation_status = 'visible'
		ORDER BY t.is_top DESC, t.create_time DESC, t.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&talks); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.talks.list", err)
	}
	return talks, count, nil
}

func (r *MyPlatformRepo) ListAuthorSeries(ctx context.Context, userID, current, size int) ([]*port.Series, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.author.series")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	var count int
	if _, err := session.SQL(`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'`, userID).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.series.count", err)
	}
	var series []*port.Series
	if err := session.SQL(`
		SELECT s.id, s.user_id, s.series_name, s.series_desc, s.cover, s.status, s.moderation_status, s.update_time,
		       count(a.id) AS article_count
		FROM t_series s
		LEFT JOIN t_article a ON a.series_id = s.id AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		WHERE s.user_id = ? AND s.is_delete = 0 AND s.status = 1 AND s.moderation_status = 'visible'
		GROUP BY s.id ORDER BY s.update_time DESC, s.id DESC LIMIT ? OFFSET ?`, userID, limit, offset).Find(&series); err != nil {
		return nil, 0, apperrors.Unavailable("platform.author.series.list", err)
	}
	return series, count, nil
}

func (r *MyPlatformRepo) ListTopicArticles(ctx context.Context, topic, slug string, current, size int) ([]*port.ArticleCard, int, error) {
	session, err := repoSession(r.engine, ctx, "platform.topic.articles")
	if err != nil {
		return nil, 0, err
	}
	limit, offset := pgsql.Page(current, size)
	join := ""
	where := ""
	if topic == "category" {
		join = ` JOIN t_category c ON a.category_id = c.id`
		where = ` AND lower(btrim(c.category_name)) = lower(btrim(?))`
	} else if topic == "tag" {
		join = ` JOIN t_article_tag at ON at.article_id = a.id JOIN t_tag t ON t.id = at.tag_id`
		where = ` AND lower(btrim(t.tag_name)) = lower(btrim(?))`
	} else {
		return nil, 0, apperrors.Invalid("platform.topic", "unsupported topic")
	}
	var count int
	if _, err := session.SQL(`SELECT count(DISTINCT a.id) FROM t_article a`+join+` WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`+where, slug).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("platform.topic.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT DISTINCT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c2.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a`+join+`
		LEFT JOIN t_category c2 ON a.category_id = c2.id
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`+where+`
		ORDER BY a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, slug, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Unavailable("platform.topic.list", err)
	}
	if err := r.attachArticleAuthorsAndTags(ctx, session, articles); err != nil {
		return nil, 0, err
	}
	return articles, count, nil
}

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
	var articles []*port.ArticleAdmin
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, a.is_top, a.is_featured, a.is_delete, a.status,
		       a.moderation_status, a.moderation_reason, a.type, a.create_time, c.category_name
		FROM t_article a LEFT JOIN t_category c ON a.category_id = c.id
		WHERE `+where+` ORDER BY a.update_time DESC NULLS LAST, a.id DESC LIMIT ? OFFSET ?`, pageArgs...).Find(&articles); err != nil {
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
			return nil
		}
		var existing entity.TTalk
		found, err := session.Where("id = ? AND user_id = ?", talk.Id, talk.UserId).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.studio.talk.owner", err)
		}
		if !found {
			return apperrors.NotFound("platform.studio.talk.update")
		}
		if _, err := session.ID(talk.Id).Where("user_id = ?", talk.UserId).Cols(
			"content", "images", "is_top", "status", "update_time",
		).Update(&talk); err != nil {
			return apperrors.Unavailable("platform.studio.talk.update", err)
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
	if strings.TrimSpace(filter.Keywords) != "" {
		column := alias + ".article_title"
		if alias == "t" {
			column = alias + ".content"
		} else if alias == "s" {
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
	}
}
