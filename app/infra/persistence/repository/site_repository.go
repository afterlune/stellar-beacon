package repository

import (
	"benetnasch/app/domain/entity"
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"context"

	"xorm.io/xorm"
)

var _ port.SiteInfoRepository = (*MySiteInfoRepo)(nil)

type MySiteInfoRepo struct{ engine *xorm.Engine }

func NewSiteInfoRepo(engine *xorm.Engine) *MySiteInfoRepo { return &MySiteInfoRepo{engine: engine} }

func (s *MySiteInfoRepo) session(ctx context.Context, op string) (*xorm.Session, error) {
	return repoSession(s.engine, ctx, op)
}

func (s *MySiteInfoRepo) count(ctx context.Context, op, query string, args ...interface{}) (int64, error) {
	session, err := s.session(ctx, op)
	if err != nil {
		return 0, err
	}
	var count int64
	if _, err := session.SQL(query, args...).Get(&count); err != nil {
		return 0, apperrors.Unavailable(op, err)
	}
	return count, nil
}

func (s *MySiteInfoRepo) CountArticles(ctx context.Context) (int64, error) {
	return s.count(ctx, "site.count_articles", "SELECT count(0) FROM t_article WHERE is_delete = 0")
}

func (s *MySiteInfoRepo) CountCategories(ctx context.Context) (int64, error) {
	return s.count(ctx, "site.count_categories", "SELECT count(0) FROM t_category")
}

func (s *MySiteInfoRepo) CountTags(ctx context.Context) (int64, error) {
	return s.count(ctx, "site.count_tags", "SELECT count(0) FROM t_tag")
}

func (s *MySiteInfoRepo) CountTalks(ctx context.Context) (int64, error) {
	return s.count(ctx, "site.count_talks", "SELECT count(0) FROM t_talk")
}

func (s *MySiteInfoRepo) CountComments(ctx context.Context, commentType int) (int64, error) {
	return s.count(ctx, "site.count_comments", "SELECT count(0) FROM t_comment WHERE type = ?", commentType)
}

func (s *MySiteInfoRepo) CountUsers(ctx context.Context) (int64, error) {
	return s.count(ctx, "site.count_users", "SELECT count(0) FROM t_user_info")
}

func (s *MySiteInfoRepo) ListUniqueViews(ctx context.Context, startTime, endTime string) ([]port.UniqueView, error) {
	session, err := s.session(ctx, "site.unique_views")
	if err != nil {
		return nil, err
	}
	var views []port.UniqueView
	if err := session.SQL(pgsql.ListUniqueViews, startTime, endTime).Find(&views); err != nil {
		return nil, apperrors.Unavailable("site.unique_views", err)
	}
	return views, nil
}

func (s *MySiteInfoRepo) ListArticleRank(ctx context.Context, ids []int) ([]port.ArticleRank, error) {
	if len(ids) == 0 {
		return []port.ArticleRank{}, nil
	}
	session, err := s.session(ctx, "site.article_rank")
	if err != nil {
		return nil, err
	}
	var articles []port.ArticleRank
	if err := session.Select("id, article_title").In("id", ids).Find(&articles); err != nil {
		return nil, apperrors.Unavailable("site.article_rank", err)
	}
	return articles, nil
}

func (s *MySiteInfoRepo) GetWebsiteConfig(ctx context.Context) (string, error) {
	session, err := s.session(ctx, "site.website_config.get")
	if err != nil {
		return "", err
	}
	var row entity.TWebsiteConfig
	if _, err := session.ID(1).Get(&row); err != nil {
		return "", apperrors.Unavailable("site.website_config.get", err)
	}
	return row.Config, nil
}

func (s *MySiteInfoRepo) UpdateWebsiteConfig(ctx context.Context, config string) error {
	return repoTx(s.engine, ctx, "site.website_config.update", func(session *xorm.Session) error {
		_, err := session.Exec("UPDATE t_website_config SET config = ? WHERE id = 1", config)
		return err
	})
}

func (s *MySiteInfoRepo) GetAbout(ctx context.Context, id int) (string, error) {
	session, err := s.session(ctx, "site.about.get")
	if err != nil {
		return "", err
	}
	var row entity.TAbout
	if _, err := session.ID(id).Get(&row); err != nil {
		return "", apperrors.Unavailable("site.about.get", err)
	}
	return row.Content, nil
}

func (s *MySiteInfoRepo) UpdateAbout(ctx context.Context, id int, content string) error {
	return repoTx(s.engine, ctx, "site.about.update", func(session *xorm.Session) error {
		_, err := session.Exec("UPDATE t_about SET content = ? WHERE id = ?", content, id)
		return err
	})
}
