package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"benetnasch/app/infra/persistence/row"
	"container/list"
	"context"
	"time"

	"xorm.io/xorm"
)

var _ port.ArticleRepository = (*MyArticleRepo)(nil)
var _ port.ArticleIndexSourceRepository = (*MyArticleRepo)(nil)

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

func (a *MyArticleRepo) attachTags(ctx context.Context, session *xorm.Session, articleID int) (interface{}, error) {
	var tags []string
	if err := session.SQL(pgsql.ArticleTags, articleID).Find(&tags); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.tags", err)
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
		article.Tags, err = a.attachTags(ctx, session, article.Id)
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
	if _, err := session.SQL(pgsql.CountPublicArticles).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListArticles, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article.Id)
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
	if _, err := session.SQL(pgsql.CountPublicArticlesByCategoryID, categoryID).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_category", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.GetArticlesByCategoryId, categoryID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_category", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article.Id)
		if err != nil {
			return nil, 0, err
		}
	}
	return articles, count, nil
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
	article.Tags, err = a.attachTags(ctx, session, article.Id)
	if err != nil {
		return port.Article{}, err
	}
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
	article.Tags, err = a.attachTags(ctx, session, article.Id)
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
	if _, err := session.SQL(pgsql.CountPublicArticlesByTagID, tagID).Get(&count); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.count_by_tag", err)
	}
	var articles []*port.ArticleCard
	if err := session.SQL(pgsql.ListArticlesByTagId, tagID, limit, offset).Find(&articles); err != nil {
		return nil, 0, apperrors.Wrap(apperrors.KindUnavailable, "article.list_by_tag", err)
	}
	for _, article := range articles {
		article.Tags, err = a.attachTags(ctx, session, article.Id)
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
	if _, err := session.SQL("SELECT count(0) FROM t_article WHERE is_delete = 0 AND status = 1").Get(&count); err != nil {
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
	query := "SELECT a.id, a.article_cover, a.article_title, a.is_top, a.is_featured, a.is_delete, a.status, a.type, a.create_time, c.category_name FROM (SELECT id, article_cover, article_title, is_top, is_featured, is_delete, status, type, create_time, category_id FROM t_article a" + filters + " ORDER BY is_top DESC, is_featured DESC, id DESC LIMIT ? OFFSET ?) a LEFT JOIN t_category c ON a.category_id = c.id ORDER BY is_top DESC, is_featured DESC, a.id DESC"
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

type articleIndexSourceRow struct {
	ID             int       `xorm:"id"`
	ArticleTitle   string    `xorm:"article_title"`
	ArticleContent string    `xorm:"article_content"`
	IsTop          int       `xorm:"is_top"`
	IsFeatured     int       `xorm:"is_featured"`
	IsDelete       int       `xorm:"is_delete"`
	Status         int       `xorm:"status"`
	Type           int       `xorm:"type"`
	OriginalURL    string    `xorm:"original_url"`
	CreateTime     time.Time `xorm:"create_time"`
	UpdateTime     time.Time `xorm:"update_time"`
	CategoryName   string    `xorm:"category_name"`
}

// ListPublicArticleIndexSources returns complete public article records using
// an exclusive ID cursor. Full content is intentionally not reused from the
// admin list DTO: a backfill must never index a truncated preview.
func (a *MyArticleRepo) ListPublicArticleIndexSources(ctx context.Context, afterArticleID, limit int) ([]port.ArticleIndexSource, error) {
	if afterArticleID < 0 {
		return nil, apperrors.Invalid("article.index_sources", "after article id cannot be negative")
	}
	if limit <= 0 || limit > 500 {
		return nil, apperrors.Invalid("article.index_sources", "limit must be between 1 and 500")
	}
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	var rows []articleIndexSourceRow
	if err := session.SQL(pgsql.ListPublicArticlesForIndex, afterArticleID, limit).Find(&rows); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.index_sources", err)
	}
	sources := make([]port.ArticleIndexSource, 0, len(rows))
	for _, row := range rows {
		if !port.IsPublicArticle(row.Status, row.IsDelete) {
			return nil, apperrors.Conflict("article.index_sources.visibility", "public index query returned a non-public article")
		}
		var tags []string
		if err := session.SQL(pgsql.ArticleTags, row.ID).Find(&tags); err != nil {
			return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.index_source_tags", err)
		}
		sources = append(sources, port.ArticleIndexSource{
			Article: port.TArticle{
				Id:             row.ID,
				ArticleTitle:   row.ArticleTitle,
				ArticleContent: row.ArticleContent,
				IsTop:          row.IsTop,
				IsFeatured:     row.IsFeatured,
				IsDelete:       row.IsDelete,
				Status:         row.Status,
				Type:           row.Type,
				OriginalUrl:    row.OriginalURL,
				CreateTime:     row.CreateTime,
				UpdateTime:     row.UpdateTime,
			},
			CategoryName: row.CategoryName,
			Tags:         append([]string(nil), tags...),
		})
	}
	return sources, nil
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

func (a *MyArticleRepo) GetArticleRecord(ctx context.Context, articleID int) (port.TArticle, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.TArticle{}, err
	}
	var article row.TArticle
	found, err := session.ID(articleID).Get(&article)
	if err != nil {
		return port.TArticle{}, apperrors.Wrap(apperrors.KindUnavailable, "article.get_record", err)
	}
	if !found {
		return port.TArticle{}, apperrors.NotFound("article.get_record")
	}
	return row.FromArticle(article), nil
}

func (a *MyArticleRepo) SaveOrUpdate(ctx context.Context, article port.TArticle, categoryName string, tagNames []string) (port.TArticle, error) {
	err := repoTx(a.engine, ctx, "article.save", func(session *xorm.Session) error {
		var category row.TCategory
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
		articleRow := row.ToArticle(article)
		if article.Id != 0 {
			if _, err := session.ID(article.Id).Update(&articleRow); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.update", err)
			}
		} else {
			if _, err := session.Insert(&articleRow); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.create", err)
			}
			article.Id = articleRow.Id
		}
		if article.Id != 0 {
			if _, err := session.Where("article_id = ?", article.Id).Delete(&row.TArticleTag{}); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete_tags", err)
			}
		}
		if len(tagNames) > 0 {
			var existing []row.TTag
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
				tag := row.TTag{TagName: name}
				if _, err := session.Insert(&tag); err != nil {
					return apperrors.Wrap(apperrors.KindUnavailable, "article.create_tag", err)
				}
				tagIDs = append(tagIDs, tag.Id)
				existingNames[name] = struct{}{}
			}
			for _, tagID := range tagIDs {
				if _, err := session.Insert(&row.TArticleTag{ArticleId: article.Id, TagId: tagID}); err != nil {
					return apperrors.Wrap(apperrors.KindUnavailable, "article.link_tag", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return port.TArticle{}, err
	}
	return a.GetArticleRecord(ctx, article.Id)
}

func (a *MyArticleRepo) UpdateTopAndFeatured(ctx context.Context, articleID, isTop, isFeatured int) (port.TArticle, error) {
	err := repoTx(a.engine, ctx, "article.update_featured", func(session *xorm.Session) error {
		if _, err := session.Exec("UPDATE t_article SET is_top = ?, is_featured = ? WHERE id = ?", isTop, isFeatured, articleID); err != nil {
			return apperrors.Wrap(apperrors.KindUnavailable, "article.update_featured", err)
		}
		return nil
	})
	if err != nil {
		return port.TArticle{}, err
	}
	return a.GetArticleRecord(ctx, articleID)
}

func (a *MyArticleRepo) UpdateDelete(ctx context.Context, ids []int, isDelete int) error {
	return repoTx(a.engine, ctx, "article.update_delete", func(session *xorm.Session) error {
		for _, id := range ids {
			article := row.TArticle{Id: id, IsDelete: isDelete}
			if _, err := session.MustCols("is_delete").ID(id).Update(&article); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.update_delete", err)
			}
		}
		return nil
	})
}

func (a *MyArticleRepo) Delete(ctx context.Context, ids []int) error {
	return repoTx(a.engine, ctx, "article.delete", func(session *xorm.Session) error {
		for _, id := range ids {
			if _, err := session.Exec("DELETE FROM t_article WHERE id = ?", id); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete", err)
			}
			if _, err := session.Where("article_id = ?", id).Delete(&row.TArticleTag{}); err != nil {
				return apperrors.Wrap(apperrors.KindUnavailable, "article.delete_tags", err)
			}
		}
		return nil
	})
}

func (a *MyArticleRepo) GetAdminArticle(ctx context.Context, articleID int) (port.TArticle, string, []string, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return port.TArticle{}, "", nil, err
	}
	var article row.TArticle
	found, err := session.ID(articleID).Get(&article)
	if err != nil {
		return port.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin", err)
	}
	if !found {
		return port.TArticle{}, "", nil, apperrors.NotFound("article.get_admin")
	}
	var category row.TCategory
	if _, err := session.ID(article.CategoryId).Get(&category); err != nil {
		return port.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin_category", err)
	}
	var tags []string
	if err := session.SQL(pgsql.ArticleTags, articleID).Find(&tags); err != nil {
		return port.TArticle{}, "", nil, apperrors.Wrap(apperrors.KindUnavailable, "article.get_admin_tags", err)
	}
	return row.FromArticle(article), category.CategoryName, tags, nil
}

func (a *MyArticleRepo) Export(ctx context.Context, ids []int) ([]port.TArticle, error) {
	session, err := a.articleSession(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []port.TArticle{}, nil
	}
	var articles []row.TArticle
	if err := session.Select("article_title, article_content").In("id", ids).Find(&articles); err != nil {
		return nil, apperrors.Wrap(apperrors.KindUnavailable, "article.export", err)
	}
	return row.FromArticles(articles), nil
}
