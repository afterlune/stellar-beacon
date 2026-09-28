package repository

import (
	"container/list"
	"context"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/query"
	"strings"

	"xorm.io/xorm"
)

var (
	_ port.ArticleRepository          = (*MyArticleRepo)(nil)
	_ port.ScheduledPublishRepository = (*MyArticleRepo)(nil)
)

type MyArticleRepo struct {
	engine *xorm.Engine
}

func NewArticleRepo(engine *xorm.Engine) *MyArticleRepo {
	return &MyArticleRepo{engine: engine}
}

func (a *MyArticleRepo) articleSession(ctx context.Context) (*xorm.Session, error) {
	return repoSession(a.engine, ctx, "article")
}

func setArticleTags(tags []string) interface{} {
	if len(tags) == 0 {
		return list.New()
	}
	return tags
}

func (a *MyArticleRepo) attachTags(ctx context.Context, session *xorm.Session, article *port.ArticleCard) (interface{}, error) {
	if article == nil || article.Id <= 0 {
		return list.New(), nil
	}
	var tags []string
	if err := session.SQL(pgsql.ArticleTags, article.Id).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.tags", err)
	}
	var owner struct {
		UserId int `xorm:"user_id"`
	}
	if _, err := session.SQL(`SELECT user_id FROM t_article WHERE id = ?`, article.Id).Get(&owner); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.author_id", err)
	}
	if owner.UserId > 0 {
		article.UserId = owner.UserId
		var author entity.TUserInfo
		if found, err := session.ID(owner.UserId).Get(&author); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.author", err)
		} else if found {
			article.Author = toPublicAuthor(author)
		}
	}
	return setArticleTags(tags), nil
}

func (a *MyArticleRepo) ListTopAndFeaturedArticles(ctx context.Context) ([]*port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListTopAndFeaturedArticles).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.list_top_featured", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, err
		}
	}
	return articles, nil
}

func (a *MyArticleRepo) ListArticles(ctx context.Context, current, size int) ([]*port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(0) FROM t_article WHERE is_delete = 0 AND status = 1 AND moderation_status = 'visible'").Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListArticles, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
}

func (a *MyArticleRepo) GetArticlesByCategoryID(ctx context.Context, current, size, categoryID int) ([]*port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(0) FROM t_article WHERE category_id = ? AND is_delete = 0 AND status = 1 AND moderation_status = 'visible'", categoryID).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_category", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.GetArticlesByCategoryId, categoryID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_category", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
}

// GetArticlesByCategoryName aggregates every author's articles that share a
// normalised category name, which is how the public discovery surfaces group
// categories. The id-based lookup stays available for legacy links.
func (a *MyArticleRepo) GetArticlesByCategoryName(ctx context.Context, current, size int, name string) ([]*port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL(`
		SELECT count(DISTINCT a.id)
		FROM t_article a
		JOIN t_category c ON c.id = a.category_id
		WHERE lower(btrim(c.category_name)) = lower(btrim(?))
		  AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`, name).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_category_name", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		JOIN t_category c ON c.id = a.category_id
		WHERE lower(btrim(c.category_name)) = lower(btrim(?))
		  AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		ORDER BY a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, name, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_category_name", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
}

// ListArticlesByTagName is the tag counterpart of GetArticlesByCategoryName.
func (a *MyArticleRepo) ListArticlesByTagName(ctx context.Context, current, size int, name string) ([]*port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL(`
		SELECT count(DISTINCT a.id)
		FROM t_article a
		JOIN t_article_tag at ON at.article_id = a.id
		JOIN t_tag t ON t.id = at.tag_id
		WHERE lower(btrim(t.tag_name)) = lower(btrim(?))
		  AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'`, name).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_tag_name", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(`
		SELECT DISTINCT a.id, a.user_id, a.article_cover, a.article_title, SUBSTR(a.article_content, 1, 500) AS article_content,
		       a.is_top, a.is_featured, c.category_name, a.status, a.moderation_status, a.create_time, a.update_time
		FROM t_article a
		JOIN t_article_tag at ON at.article_id = a.id
		JOIN t_tag t ON t.id = at.tag_id
		LEFT JOIN t_category c ON c.id = a.category_id
		WHERE lower(btrim(t.tag_name)) = lower(btrim(?))
		  AND a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		ORDER BY a.create_time DESC, a.id DESC LIMIT ? OFFSET ?`, name, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_tag_name", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
}

// ListArticleCardsByIDs loads public article cards for an explicit id set. The
// reader-interaction feature uses it to render an account's favourites.
func (a *MyArticleRepo) ListArticleCardsByIDs(ctx context.Context, articleIDs []int) ([]*port.ArticleCard, error) {
	if len(articleIDs) == 0 {
		return []*port.ArticleCard{}, nil
	}
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	statement := strings.Replace(pgsql.ListArticlesByIds, "%s", placeholders(len(articleIDs)), 1)
	var articles []*port.ArticleCard
	if err := session.SQL(statement, intArgs(articleIDs)...).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_ids", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, err
		}
	}
	return articles, nil
}

// ListArticleCardsBySeries loads one collection in its authored order.
func (a *MyArticleRepo) ListArticleCardsBySeries(ctx context.Context, seriesID int) ([]*port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListArticlesBySeries, seriesID).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_series", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, err
		}
	}
	return articles, nil
}

// ListRelatedArticles returns rule-ranked public articles for the reading
// page. Same-series articles are deliberately excluded; fallback rows fill
// sparse matches without reintroducing the current article or its series.
func (a *MyArticleRepo) ListRelatedArticles(ctx context.Context, articleID, categoryID, seriesID, limit int) ([]*port.ArticleCard, error) {
	if articleID <= 0 || limit <= 0 {
		return []*port.ArticleCard{}, nil
	}
	if limit > 10 {
		limit = 10
	}
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var related []*port.ArticleCard
	if err := session.SQL(pgsql.ListRelatedArticles, articleID, seriesID, articleID, categoryID, limit).Find(&related); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.related", err)
	}
	if len(related) < limit {
		var fallback []*port.ArticleCard
		if err := session.SQL(pgsql.ListRelatedFallback, articleID, seriesID, limit).Find(&fallback); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.related_fallback", err)
		}
		seen := make(map[int]bool, len(related)+len(fallback))
		for _, article := range related {
			if article != nil {
				seen[article.Id] = true
			}
		}
		for _, article := range fallback {
			if article == nil || seen[article.Id] {
				continue
			}
			related = append(related, article)
			seen[article.Id] = true
			if len(related) == limit {
				break
			}
		}
	}
	for _, article := range related {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, err
		}
	}
	return related, nil
}

// PublishDueScheduledArticles atomically releases due scheduled articles and
// creates the durable notification hand-off record in the same transaction.
func (a *MyArticleRepo) PublishDueScheduledArticles(ctx context.Context, now time.Time, limit int) ([]port.ScheduledPublish, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	result := make([]port.ScheduledPublish, 0)
	err := repoTx(a.engine, ctx, "article.publish_scheduled", func(session *xorm.Session) error {
		var rows []struct {
			Id               int       `xorm:"id"`
			UserId           int       `xorm:"user_id"`
			ScheduledAt      time.Time `xorm:"scheduled_at"`
			ModerationStatus string    `xorm:"moderation_status"`
		}
		if err := session.SQL(`
			SELECT id, user_id, scheduled_at, moderation_status
			FROM t_article
			WHERE status = 4 AND is_delete = 0 AND scheduled_at IS NOT NULL AND scheduled_at <= ?
			ORDER BY scheduled_at ASC, id ASC
			LIMIT ? FOR UPDATE SKIP LOCKED`, now, limit).Find(&rows); err != nil {
			return apperrors.Unavailable("article.publish_scheduled.select", err)
		}
		for _, row := range rows {
			if _, err := session.Exec(`UPDATE t_article SET status = 1, update_time = ? WHERE id = ? AND status = 4 AND is_delete = 0`, now, row.Id); err != nil {
				return apperrors.Unavailable("article.publish_scheduled.update", err)
			}
			if row.ModerationStatus == "visible" {
				if err := recordAuthorPublishEvent(session, port.FollowContentArticle, row.Id, row.UserId, now); err != nil {
					return err
				}
			}
			var recordID int
			if _, err := session.SQL(`
				INSERT INTO t_article_publish_record
					(article_id, user_id, scheduled_at, published_at, notification_state, notification_attempts, next_retry_at, last_error, create_time, update_time)
				VALUES (?, ?, ?, ?, ?, 0, NULL, '', ?, ?)
				ON CONFLICT (article_id, scheduled_at) DO UPDATE
				SET published_at = EXCLUDED.published_at, update_time = EXCLUDED.update_time
				RETURNING id`, row.Id, row.UserId, row.ScheduledAt, now, port.ScheduledNotificationPending, now, now).Get(&recordID); err != nil {
				return apperrors.Unavailable("article.publish_scheduled.record", err)
			}
			result = append(result, port.ScheduledPublish{
				RecordID: recordID, ArticleID: row.Id, UserID: row.UserId, ScheduledAt: row.ScheduledAt, PublishedAt: now,
				ModerationStatus: row.ModerationStatus, NotificationState: port.ScheduledNotificationPending,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *MyArticleRepo) ListRetryableScheduledPublishes(ctx context.Context, now time.Time, limit int) ([]port.ScheduledPublish, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		RecordID             int        `xorm:"record_id"`
		ArticleID            int        `xorm:"article_id"`
		UserID               int        `xorm:"user_id"`
		ScheduledAt          time.Time  `xorm:"scheduled_at"`
		PublishedAt          time.Time  `xorm:"published_at"`
		ModerationStatus     string     `xorm:"moderation_status"`
		NotificationState    string     `xorm:"notification_state"`
		NotificationAttempts int        `xorm:"notification_attempts"`
		NextRetryAt          *time.Time `xorm:"next_retry_at"`
		LastError            string     `xorm:"last_error"`
	}
	if err := session.SQL(`
		SELECT r.id AS record_id, r.article_id, r.user_id, r.scheduled_at, r.published_at, a.moderation_status,
		       r.notification_state, r.notification_attempts, r.next_retry_at, r.last_error
		FROM t_article_publish_record r
		JOIN t_article a ON a.id = r.article_id
		WHERE r.notification_state IN ('pending', 'failed')
		  AND r.notification_attempts < 6
		  AND (r.next_retry_at IS NULL OR r.next_retry_at <= ?)
		  AND a.is_delete = 0 AND a.status = 1
		ORDER BY r.id ASC
		LIMIT ?`, now, limit).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("article.publish_retry.list", err)
	}
	out := make([]port.ScheduledPublish, 0, len(rows))
	for _, row := range rows {
		out = append(out, port.ScheduledPublish{
			RecordID: row.RecordID, ArticleID: row.ArticleID, UserID: row.UserID, ScheduledAt: row.ScheduledAt,
			PublishedAt: row.PublishedAt, ModerationStatus: row.ModerationStatus, NotificationState: row.NotificationState,
			NotificationAttempts: row.NotificationAttempts, NextRetryAt: row.NextRetryAt, LastError: row.LastError,
		})
	}
	return out, nil
}

func (a *MyArticleRepo) MarkScheduledNotificationQueued(ctx context.Context, recordID int, at time.Time) error {
	session, err := a.articleSession(ctx)
	if err != nil {
		return err
	}
	if _, err := session.Exec(`UPDATE t_article_publish_record SET notification_state = ?, next_retry_at = NULL, last_error = '', update_time = ? WHERE id = ?`, port.ScheduledNotificationQueued, at, recordID); err != nil {
		return apperrors.Unavailable("article.publish_retry.queued", err)
	}
	return nil
}

func (a *MyArticleRepo) MarkScheduledNotificationFailed(ctx context.Context, recordID int, message string, retryAt *time.Time) error {
	if len(message) > 2000 {
		message = message[:2000]
	}
	session, err := a.articleSession(ctx)
	if err != nil {
		return err
	}
	if _, err := session.Exec(`
		UPDATE t_article_publish_record
		SET notification_state = ?, notification_attempts = notification_attempts + 1,
		    next_retry_at = ?, last_error = ?, update_time = CURRENT_TIMESTAMP
		WHERE id = ?`, port.ScheduledNotificationFailed, retryAt, message, recordID); err != nil {
		return apperrors.Unavailable("article.publish_retry.failed", err)
	}
	return nil
}

func (a *MyArticleRepo) MarkScheduledNotificationSuppressed(ctx context.Context, recordID int, message string) error {
	session, err := a.articleSession(ctx)
	if err != nil {
		return err
	}
	if _, err := session.Exec(`UPDATE t_article_publish_record SET notification_state = ?, next_retry_at = NULL, last_error = ?, update_time = CURRENT_TIMESTAMP WHERE id = ?`, port.ScheduledNotificationSuppressed, message, recordID); err != nil {
		return apperrors.Unavailable("article.publish_retry.suppressed", err)
	}
	return nil
}

func (a *MyArticleRepo) GetArticleByID(ctx context.Context, articleID int) (port.Article, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.Article{}, err
	}
	var article port.Article
	found, err := session.SQL(pgsql.GetArticleById, articleID).Get(&article)
	if err != nil {
		return port.Article{}, apperrors.Wrap(apperrors.KindUnavailable, "article.get", err)
	}
	if !found {
		return port.Article{}, apperrors.NotFound("article.get")
	}
	card := port.ArticleCard{Id: article.Id}
	card.Tags, err = a.attachTags(ctx, session, &card)
	if err != nil {
		return port.Article{}, err
	}
	article.Tags = card.Tags
	article.UserId = card.UserId
	article.Author = card.Author
	return article, nil
}

func (a *MyArticleRepo) card(ctx context.Context, session *xorm.Session, query string, id ...interface{}) (port.ArticleCard, error) {
	var article port.ArticleCard
	found, err := session.SQL(query, id...).Get(&article)
	if err != nil {
		return port.ArticleCard{}, apperrors.Wrap(apperrors.KindUnavailable, "article.navigation", err)
	}
	if !found {
		return port.ArticleCard{}, nil
	}
	article.Tags, err = a.attachTags(ctx, session, &article)
	if err != nil {
		return port.ArticleCard{}, err
	}
	return article, nil
}

func (a *MyArticleRepo) GetPreArticleByID(ctx context.Context, articleID int) (port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.ArticleCard{}, err
	}
	return a.card(ctx, session, pgsql.GetPreArticleById, articleID)
}

func (a *MyArticleRepo) GetNextArticleByID(ctx context.Context, articleID int) (port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.ArticleCard{}, err
	}
	return a.card(ctx, session, pgsql.GetNextArticleById, articleID)
}

func (a *MyArticleRepo) GetFirstArticle(ctx context.Context) (port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.ArticleCard{}, err
	}
	return a.card(ctx, session, pgsql.GetFirstArticle)
}

func (a *MyArticleRepo) GetLastArticle(ctx context.Context) (port.ArticleCard, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.ArticleCard{}, err
	}
	return a.card(ctx, session, pgsql.GetLastArticle)
}

func (a *MyArticleRepo) ListArticlesByTagID(ctx context.Context, current, size, tagID int) ([]*port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(DISTINCT a.id) FROM t_article a JOIN t_article_tag at ON a.id = at.article_id WHERE at.tag_id = ? AND a.is_delete = 0 AND a.status = 1 AND moderation_status = 'visible'", tagID).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_tag", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListArticlesByTagId, tagID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_tag", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
}

func (a *MyArticleRepo) ListArchives(ctx context.Context, current, size int) ([]port.ArticleCard, int, error) {
	limit, offset := pgsql.Page(current, size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	var count int
	if _, err := session.SQL("SELECT count(0) FROM t_article WHERE is_delete = 0 AND status = 1 AND moderation_status = 'visible'").Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_archives", err)
	}
	var articles []port.ArticleCard
	if err := session.SQL(pgsql.ListArchives, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_archives", err)
	}
	return articles, count, nil
}

func articleAdminFilters(filter port.ArticleFilter) (string, []interface{}) {
	query := " WHERE a.is_delete = ?"
	args := []interface{}{filter.IsDelete}
	if filter.Keywords != "" {
		query += " AND a.article_title LIKE ? ESCAPE '\\'"
		args = append(args, pgsql.ContainsPattern(filter.Keywords))
	}
	if filter.Status != 0 {
		query += " AND a.status = ?"
		args = append(args, filter.Status)
	}
	if filter.ModerationStatus != "" {
		query += " AND a.moderation_status = ?"
		args = append(args, filter.ModerationStatus)
	}
	if filter.Category != 0 {
		query += " AND a.category_id = ?"
		args = append(args, filter.Category)
	}
	if filter.Type != 0 {
		query += " AND a.type = ?"
		args = append(args, filter.Type)
	}
	if filter.Tag != 0 {
		query += " AND a.id IN (SELECT article_id FROM t_article_tag WHERE tag_id = ?)"
		args = append(args, filter.Tag)
	}
	return query, args
}

func (a *MyArticleRepo) CountArticleAdmins(ctx context.Context, filter port.ArticleFilter) (int, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return 0, err
	}
	filters, args := articleAdminFilters(filter)
	var count int
	if _, err := session.SQL("SELECT count(DISTINCT a.id) FROM t_article a LEFT JOIN t_article_tag at ON a.id = at.article_id"+filters, args...).Get(&count); err != nil {
		return 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_admin", err)
	}
	return count, nil
}

func (a *MyArticleRepo) ListArticlesAdmin(ctx context.Context, filter port.ArticleFilter) ([]*port.ArticleAdmin, error) {
	limit, offset := pgsql.Page(filter.Current, filter.Size)
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	filters, args := articleAdminFilters(filter)
	query := `SELECT a.id, a.user_id, a.article_cover, a.article_title, a.is_top, a.is_featured, a.is_delete,
		a.status, a.type, a.moderation_status, a.moderation_reason, a.moderated_by, a.moderated_at, a.create_time,
		c.category_name, u.handle AS author_handle, u.nickname AS author_nickname, u.avatar AS author_avatar
		FROM (
			SELECT id, user_id, article_cover, article_title, is_top, is_featured, is_delete, status, type,
			       moderation_status, moderation_reason, moderated_by, moderated_at, create_time, category_id
			FROM t_article a` + filters + ` ORDER BY is_top DESC, is_featured DESC, id DESC LIMIT ? OFFSET ?
		) a
		LEFT JOIN t_category c ON a.category_id = c.id
		LEFT JOIN t_user_info u ON a.user_id = u.id
		ORDER BY a.is_top DESC, a.is_featured DESC, a.id DESC`
	args = append(args, limit, offset)
	var articles []*port.ArticleAdmin
	if err := session.SQL(query, args...).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.list_admin", err)
	}
	for _, article := range articles {
		var tags []port.Tag
		if err := session.SQL(pgsql.ArticlesAdminTags, article.Id).Find(&tags); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.admin_tags", err)
		}
		article.TagDTOs = tags
	}
	return articles, nil
}

func (a *MyArticleRepo) ListArticleStatistics(ctx context.Context) ([]port.ArticleStatistics, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var statistics []port.ArticleStatistics
	if err := session.SQL(pgsql.ListArticleStatistics).Find(&statistics); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.statistics", err)
	}
	return statistics, nil
}

type articleSearchRow struct {
	Id               int       `xorm:"id"`
	UserId           int       `xorm:"user_id"`
	ArticleCover     string    `xorm:"article_cover"`
	ArticleTitle     string    `xorm:"article_title"`
	ArticleContent   string    `xorm:"article_content"`
	CategoryName     string    `xorm:"category_name"`
	CreateTime       time.Time `xorm:"create_time"`
	UpdateTime       time.Time `xorm:"update_time"`
	IsDelete         int       `xorm:"is_delete"`
	Status           int       `xorm:"status"`
	ModerationStatus string    `xorm:"moderation_status"`
	AuthorId         int       `xorm:"author_id"`
	AuthorHandle     string    `xorm:"author_handle"`
	AuthorNickname   string    `xorm:"author_nickname"`
	AuthorAvatar     string    `xorm:"author_avatar"`
	AuthorIntro      string    `xorm:"author_intro"`
	AuthorWebsite    string    `xorm:"author_website"`
}

const articleSearchDocumentSelect = `
	SELECT a.id, a.user_id, a.article_cover, a.article_title, a.article_content,
	       COALESCE(c.category_name, '') AS category_name,
	       a.create_time, a.update_time, a.is_delete, a.status, a.moderation_status,
	       COALESCE(u.id, 0) AS author_id,
	       COALESCE(u.handle, '') AS author_handle,
	       COALESCE(u.nickname, '') AS author_nickname,
	       COALESCE(u.avatar, '') AS author_avatar,
	       COALESCE(u.intro, '') AS author_intro,
	       COALESCE(u.website, '') AS author_website
	FROM t_article a
	LEFT JOIN t_category c ON c.id = a.category_id
	LEFT JOIN t_user_info u ON u.id = a.user_id`

func articleSearchDocument(row articleSearchRow) port.ArticleSearch {
	return port.ArticleSearch{
		Id: row.Id, UserId: row.UserId, ArticleCover: row.ArticleCover,
		ArticleTitle: row.ArticleTitle, ArticleContent: row.ArticleContent,
		CategoryName: row.CategoryName, CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
		Author: port.PublicAuthor{
			Id: row.AuthorId, Handle: row.AuthorHandle, Nickname: row.AuthorNickname,
			Avatar: row.AuthorAvatar, Intro: row.AuthorIntro, Website: row.AuthorWebsite,
		},
		IsDelete: row.IsDelete, Status: row.Status, ModerationStatus: row.ModerationStatus,
	}
}

func (a *MyArticleRepo) GetArticleSearchDocument(ctx context.Context, articleID int) (port.ArticleSearch, bool, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.ArticleSearch{}, false, err
	}
	var row articleSearchRow
	found, err := session.SQL(articleSearchDocumentSelect+" WHERE a.id = ?", articleID).Get(&row)
	if err != nil {
		return port.ArticleSearch{}, false, apperrors.Wrap(apperrors.KindUnavailable, "article.search_document", err)
	}
	if !found || row.IsDelete != 0 || row.Status != 1 || row.ModerationStatus != "visible" {
		return port.ArticleSearch{}, false, nil
	}
	return articleSearchDocument(row), true, nil
}

func (a *MyArticleRepo) ListPublicArticleSearchDocuments(ctx context.Context) ([]port.ArticleSearch, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var rows []articleSearchRow
	if err := session.SQL(articleSearchDocumentSelect + `
		WHERE a.is_delete = 0 AND a.status = 1 AND a.moderation_status = 'visible'
		ORDER BY a.id ASC`).Find(&rows); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.search_documents", err)
	}
	documents := make([]port.ArticleSearch, 0, len(rows))
	for _, row := range rows {
		documents = append(documents, articleSearchDocument(row))
	}
	return documents, nil
}

func (a *MyArticleRepo) GetArticleRecord(ctx context.Context, articleID int) (entity.TArticle, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return entity.TArticle{}, err
	}
	var article entity.TArticle
	found, err := session.ID(articleID).Get(&article)
	if err != nil {
		return entity.TArticle{}, apperrors.Wrap(apperrors.KindUnavailable, "article.get_record", err)
	}
	if !found {
		return entity.TArticle{}, apperrors.NotFound("article.get_record")
	}
	return article, nil
}

func (a *MyArticleRepo) SaveOrUpdate(ctx context.Context, article entity.TArticle, categoryName string, tagNames []string) (entity.TArticle, error) {
	wasPublic := false
	err := ormInit.WithEngineTx(a.engine, ctx, func(session *xorm.Session) error {
		var category entity.TCategory
		if _, err := session.SQL("SELECT * FROM t_category WHERE category_name = ?", categoryName).Get(&category); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "article.category", err)
		}
		if category.Id == 0 && article.Status != 3 && categoryName != "" {
			category.CategoryName = categoryName
			if _, err := session.Insert(&category); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.create_category", err)
			}
		}
		if category.Id != 0 {
			article.CategoryId = category.Id
		}
		if article.Id == 0 && article.ModerationStatus == "" {
			article.ModerationStatus = "visible"
		}
		if article.Id != 0 {
			var existing struct {
				Status           int    `xorm:"status"`
				IsDelete         int    `xorm:"is_delete"`
				ModerationStatus string `xorm:"moderation_status"`
			}
			if _, err := session.SQL("SELECT status, is_delete, moderation_status FROM t_article WHERE id = ?", article.Id).Get(&existing); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.publish_state", err)
			}
			wasPublic = existing.Status == 1 && existing.IsDelete == 0 && existing.ModerationStatus == "visible"
			if _, err := session.ID(article.Id).MustCols("article_content_html").Update(&article); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.update", err)
			}
		} else if _, err := session.Insert(&article); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "article.create", err)
		}
		if article.Id != 0 {
			if _, err := session.Where("article_id = ?", article.Id).Delete(&entity.TArticleTag{}); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete_tags", err)
			}
		}
		if !wasPublic {
			if err := recordArticlePublishEvent(session, article.Id, time.Now()); err != nil {
				return err
			}
		}
		if len(tagNames) > 0 {
			var existing []entity.TTag
			if err := session.In("tag_name", tagNames).Find(&existing); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.find_tags", err)
			}
			existingNames := make(map[string]struct{}, len(existing))
			tagIDs := make([]int, 0, len(existing)+len(tagNames))
			for _, tag := range existing {
				existingNames[tag.TagName] = struct{}{}
				tagIDs = append(tagIDs, tag.Id)
			}
			for _, name := range tagNames {
				if _, ok := existingNames[name]; ok {
					continue
				}
				tag := entity.TTag{TagName: name}
				if _, err := session.Insert(&tag); err != nil {
					return apperrors.Wrap(apperrors.KindUnavailable, "article.create_tag", err)
				}
				tagIDs = append(tagIDs, tag.Id)
				existingNames[name] = struct{}{}
			}
			for _, tagID := range tagIDs {
				if _, err := session.Insert(&entity.TArticleTag{ArticleId: article.Id, TagId: tagID}); err != nil {
					return apperrors.Wrap(apperrors.KindUnavailable, "article.link_tag", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return entity.TArticle{}, err
	}
	return a.GetArticleRecord(ctx, article.Id)
}

func (a *MyArticleRepo) UpdateTopAndFeatured(ctx context.Context, articleID, isTop, isFeatured int) (entity.TArticle, error) {
	err := ormInit.WithEngineTx(a.engine, ctx, func(session *xorm.Session) error {
		if _, err := session.Exec("UPDATE t_article SET is_top = ?, is_featured = ? WHERE id = ?", isTop, isFeatured, articleID); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "article.update_featured", err)
		}
		return nil
	})
	if err != nil {
		return entity.TArticle{}, err
	}
	return a.GetArticleRecord(ctx, articleID)
}

func (a *MyArticleRepo) UpdateDelete(ctx context.Context, ids []int, isDelete int) error {
	return ormInit.WithEngineTx(a.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			article := entity.TArticle{Id: id, IsDelete: isDelete}
			if _, err := session.MustCols("is_delete").ID(id).Update(&article); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.update_delete", err)
			}
			if isDelete == 0 {
				if err := recordArticlePublishEvent(session, id, time.Now()); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (a *MyArticleRepo) Delete(ctx context.Context, ids []int) error {
	return ormInit.WithEngineTx(a.engine, ctx, func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.Exec("DELETE FROM t_article WHERE id = ?", id); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete", err)
			}
			if _, err := session.Where("article_id = ?", id).Delete(&entity.TArticleTag{}); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete_tags", err)
			}
		}
		return nil
	})
}

func (a *MyArticleRepo) GetAdminArticle(ctx context.Context, articleID int) (entity.TArticle, string, []string, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return entity.TArticle{}, "", nil, err
	}
	var article entity.TArticle
	found, err := session.ID(articleID).Get(&article)
	if err != nil {
		return entity.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin", err)
	}
	if !found {
		return entity.TArticle{}, "", nil, apperrors.NotFound("article.get_admin")
	}
	var category entity.TCategory
	if _, err := session.ID(article.CategoryId).Get(&category); err != nil {
		return entity.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin_category", err)
	}
	var tags []string
	if err := session.SQL(pgsql.ArticleTags, articleID).Find(&tags); err != nil {
		return entity.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin_tags", err)
	}
	return article, category.CategoryName, tags, nil
}

func (a *MyArticleRepo) Export(ctx context.Context, ids []int) ([]entity.TArticle, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []entity.TArticle{}, nil
	}
	var articles []entity.TArticle
	if err := session.Select("article_title, article_content").In("id", ids).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.export", err)
	}
	return articles, nil
}
