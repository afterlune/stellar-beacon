package migrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/casbin/xorm-adapter/v2"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/robfig/cron/v3"
	"xorm.io/xorm"
)

const migrationTable = "stellar_beacon_schema_migrations"

// Apply creates the current schema and inserts only the system defaults needed
// to operate a new site. It deliberately does not read the historical data dump.
func Apply(ctx context.Context, engine *xorm.Engine) error {
	if engine == nil {
		return errors.New("database engine is nil")
	}
	if err := engine.Ping(); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if _, err := engine.Exec(`
		CREATE TABLE IF NOT EXISTS ` + migrationTable + ` (
			version BIGINT PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	if err := applySchema(engine); err != nil {
		return err
	}
	if err := recordMigration(ctx, engine, 1, "baseline-schema"); err != nil {
		return err
	}
	if err := applySystemDefaults(ctx, engine); err != nil {
		return err
	}
	if err := applyGrowthSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyGrowthOperationsSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyArticleContentSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyArticleReactionSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCommentNotificationSchema(ctx, engine); err != nil {
		return err
	}
	if err := applySeriesSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyFriendLinkReviewSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyScheduledPublishSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyJobSchedulerSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyContentAnalyticsSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyContinuationAnalyticsSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyContinuationTargetSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyMultiUserSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyContentModerationSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyStudioOperationsSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyFollowSocialSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyInteractionNotificationSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyDiscoveryRankingSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyTopicSubscriptionSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyRecommendationFeedbackSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCollectionSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCollectionSubscriptionSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCollectionInteractionSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCollectionCommunitySchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCommentModerationSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyCommentGovernanceSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyStudioActivationSchema(ctx, engine); err != nil {
		return err
	}
	return nil
}

func applySchema(engine *xorm.Engine) error {
	checkSession := engine.NewSession()
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 1)").Get(&applied); err != nil {
		return fmt.Errorf("check baseline schema migration: %w", err)
	}
	if applied {
		return nil
	}
	var legacySchema bool
	if _, err := checkSession.SQL("SELECT to_regclass('public.t_article') IS NOT NULL").Get(&legacySchema); err != nil {
		return fmt.Errorf("check legacy schema: %w", err)
	}
	if legacySchema {
		// Existing installations already own the application tables. Running
		// xorm.Sync2 against them can rewrite historical constraints and
		// indexes, so baseline them without diffing the old schema.
		var casbinSchema bool
		if _, err := checkSession.SQL("SELECT to_regclass('public.casbin_rule') IS NOT NULL").Get(&casbinSchema); err != nil {
			return fmt.Errorf("check Casbin policy table: %w", err)
		}
		if !casbinSchema {
			if _, err := xormadapter.NewAdapterByEngine(engine); err != nil {
				return fmt.Errorf("create Casbin policy table: %w", err)
			}
		}
		return nil
	}
	models := []any{
		new(entity.TAbout), new(entity.TArticle), new(entity.TArticleTag),
		new(entity.TArticleReaction), new(entity.TCategory), new(entity.TComment),
		new(entity.TExceptionLog),
		new(entity.TFriendLink), new(entity.TJob), new(entity.TJobLog),
		new(entity.TMenu), new(entity.TOperationLog), new(entity.TPhoto),
		new(entity.TPhotoAlbum), new(entity.TResource), new(entity.TRole),
		new(entity.TRoleMenu), new(entity.TRoleResource), new(entity.TSeries), new(entity.TTag),
		new(entity.TTalk), new(entity.TUniqueView), new(entity.TUserAuth),
		new(entity.TUserInfo), new(entity.TUserRole), new(entity.TWebsiteConfig),
		new(entity.TArticleDailyMetric), new(entity.TArticleContinuationTarget),
	}
	if err := engine.Sync2(models...); err != nil {
		return fmt.Errorf("create application tables: %w", err)
	}
	if _, err := xormadapter.NewAdapterByEngine(engine); err != nil {
		return fmt.Errorf("create Casbin policy table: %w", err)
	}
	return nil
}

func applySystemDefaults(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 2)").Get(&applied); err != nil {
		return fmt.Errorf("check site defaults migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin site defaults migration: %w", err)
	}
	defer session.Rollback()

	if err := seedRoles(session); err != nil {
		return err
	}
	if err := seedMenus(session); err != nil {
		return err
	}
	if err := seedResourcesAndPolicies(session); err != nil {
		return err
	}
	if err := seedSiteContent(session); err != nil {
		return err
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 2, "blank-site-defaults"); err != nil {
		return fmt.Errorf("record site defaults migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit site defaults migration: %w", err)
	}
	return nil
}

func recordMigration(ctx context.Context, engine *xorm.Engine, version int, name string) error {
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin schema migration record: %w", err)
	}
	defer session.Rollback()
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?) ON CONFLICT (version) DO NOTHING", version, name); err != nil {
		return fmt.Errorf("record schema migration %d: %w", version, err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit schema migration %d: %w", version, err)
	}
	return nil
}

func seedRoles(session *xorm.Session) error {
	roles := []struct {
		id   int
		name string
	}{
		{1, "admin"},
		{2, "user"},
		{3, "visitor"},
	}
	for _, role := range roles {
		var existing entity.TRole
		found, err := session.Where("role_name = ?", role.name).Get(&existing)
		if err != nil {
			return fmt.Errorf("find default role %q: %w", role.name, err)
		}
		if found {
			if existing.Id != role.id {
				return fmt.Errorf("default role %q has unexpected id %d", role.name, existing.Id)
			}
			continue
		}
		var occupied entity.TRole
		found, err = session.ID(role.id).Get(&occupied)
		if err != nil {
			return fmt.Errorf("check default role id %d: %w", role.id, err)
		}
		if found {
			return fmt.Errorf("role id %d is already occupied", role.id)
		}
		if _, err := session.Exec("INSERT INTO t_role (id, role_name, is_disable, create_time) OVERRIDING SYSTEM VALUE VALUES (?, ?, 0, ?)", role.id, role.name, time.Now()); err != nil {
			return fmt.Errorf("insert default role %q: %w", role.name, err)
		}
	}
	if _, err := session.Exec("SELECT setval(pg_get_serial_sequence('t_role', 'id'), GREATEST((SELECT MAX(id) FROM t_role), 1), true)"); err != nil {
		return fmt.Errorf("advance role id sequence: %w", err)
	}
	return nil
}

type menuSeed struct {
	name       string
	path       string
	component  string
	icon       string
	order      int
	parentPath string
	hidden     int
}

var defaultMenus = []menuSeed{
	{name: "工作台", path: "/", component: "/dashboard/Workplace.vue", icon: "home", order: 1},
	{name: "仪表盘", path: "/dashboard", component: "/dashboard/Dashboard.vue", icon: "dashboard", order: 2},
	{name: "实时监控", path: "/monitor", component: "/dashboard/Monitor.vue", icon: "monitor", order: 3},
	{name: "图片资源", path: "/media", component: "/media/Media.vue", icon: "image", order: 4},
	{name: "文章管理", path: "/article-submenu", component: "Layout", icon: "article", order: 5},
	{name: "消息管理", path: "/message-submenu", component: "Layout", icon: "message", order: 6},
	{name: "说说管理", path: "/talk-submenu", component: "Layout", icon: "talk", order: 7},
	{name: "相册管理", path: "/album-submenu", component: "Layout", icon: "album", order: 8},
	{name: "系统管理", path: "/system-submenu", component: "Layout", icon: "settings", order: 9},
	{name: "用户管理", path: "/users-submenu", component: "Layout", icon: "users", order: 10},
	{name: "权限管理", path: "/permission-submenu", component: "Layout", icon: "permissions", order: 11},
	{name: "日志管理", path: "/log-submenu", component: "Layout", icon: "logs", order: 12},
	{name: "个人中心", path: "/setting", component: "/setting/Setting.vue", icon: "profile", order: 13},
	{name: "订阅与增长", path: "/growth", component: "/growth/Newsletter.vue", icon: "notification", order: 14},
	{name: "发布文章", path: "/articles", component: "/article/Article.vue", icon: "pen", order: 1, parentPath: "/article-submenu"},
	{name: "修改文章", path: "/articles/:articleId", component: "/article/Article.vue", icon: "pen", order: 2, parentPath: "/article-submenu", hidden: 1},
	{name: "文章列表", path: "/article-list", component: "/article/ArticleList.vue", icon: "list", order: 3, parentPath: "/article-submenu"},
	{name: "分类管理", path: "/categories", component: "/category/Category.vue", icon: "category", order: 4, parentPath: "/article-submenu"},
	{name: "标签管理", path: "/tags", component: "/tag/Tag.vue", icon: "tags", order: 5, parentPath: "/article-submenu"},
	{name: "系列管理", path: "/series", component: "/series/Series.vue", icon: "list", order: 6, parentPath: "/article-submenu"},
	{name: "内容表现", path: "/content-performance", component: "/content/ContentPerformance.vue", icon: "chart", order: 7, parentPath: "/article-submenu"},
	{name: "内容审核", path: "/content-moderation", component: "/content/ContentModeration.vue", icon: "audit", order: 8, parentPath: "/article-submenu"},
	{name: "评论管理", path: "/comments", component: "/comment/Comment.vue", icon: "comments", order: 1, parentPath: "/message-submenu"},
	{name: "说说列表", path: "/talk-list", component: "/talk/TalkList.vue", icon: "list", order: 1, parentPath: "/talk-submenu"},
	{name: "发布说说", path: "/talks", component: "/talk/Talk.vue", icon: "pen", order: 2, parentPath: "/talk-submenu"},
	{name: "修改说说", path: "/talks/:talkId", component: "/talk/Talk.vue", icon: "pen", order: 3, parentPath: "/talk-submenu", hidden: 1},
	{name: "相册列表", path: "/albums", component: "/album/Album.vue", icon: "albums", order: 1, parentPath: "/album-submenu"},
	{name: "照片管理", path: "/albums/:albumId", component: "/album/Photo.vue", icon: "photos", order: 2, parentPath: "/album-submenu", hidden: 1},
	{name: "照片回收站", path: "/photos/delete", component: "/album/Delete.vue", icon: "delete", order: 3, parentPath: "/album-submenu", hidden: 1},
	{name: "友链管理", path: "/links", component: "/friendLink/FriendLink.vue", icon: "links", order: 1, parentPath: "/system-submenu"},
	{name: "关于我", path: "/about", component: "/about/About.vue", icon: "about", order: 2, parentPath: "/system-submenu"},
	{name: "定时任务", path: "/quartz", component: "/quartz/Quartz.vue", icon: "schedule", order: 3, parentPath: "/system-submenu"},
	{name: "网站配置", path: "/website", component: "/website/Website.vue", icon: "website", order: 4, parentPath: "/system-submenu"},
	{name: "用户列表", path: "/users", component: "/user/User.vue", icon: "users", order: 1, parentPath: "/users-submenu"},
	{name: "在线用户", path: "/online/users", component: "/user/Online.vue", icon: "online", order: 2, parentPath: "/users-submenu"},
	{name: "角色管理", path: "/roles", component: "/role/Role.vue", icon: "roles", order: 1, parentPath: "/permission-submenu"},
	{name: "接口管理", path: "/resources", component: "/resource/Resource.vue", icon: "resources", order: 2, parentPath: "/permission-submenu"},
	{name: "菜单管理", path: "/menus", component: "/menu/Menu.vue", icon: "menus", order: 3, parentPath: "/permission-submenu"},
	{name: "操作日志", path: "/operation/log", component: "/log/OperationLog.vue", icon: "logs", order: 1, parentPath: "/log-submenu"},
	{name: "异常日志", path: "/exception/log", component: "/log/ExceptionLog.vue", icon: "errors", order: 2, parentPath: "/log-submenu"},
	{name: "定时任务日志", path: "/quartz/log/:quartzId", component: "/log/QuartzLog.vue", icon: "schedule", order: 3, parentPath: "/log-submenu", hidden: 1},
}

// applyGrowthSchema is an additive migration. It uses explicit PostgreSQL DDL
// so databases created by older releases receive the same tables as a fresh
// install without relying on xorm's schema diff behavior.
func applyGrowthSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 3)").Get(&applied); err != nil {
		return fmt.Errorf("check growth migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin growth migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_newsletter_subscriber (
			id SERIAL PRIMARY KEY,
			email VARCHAR(254) NOT NULL UNIQUE,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			confirm_token_hash VARCHAR(64) NOT NULL DEFAULT '',
			confirm_token_expires_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			unsubscribe_token_hash VARCHAR(64) NOT NULL DEFAULT '',
			confirmed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_subscriber_status ON t_newsletter_subscriber(status)`,
		`CREATE TABLE IF NOT EXISTS t_newsletter_delivery (
			id SERIAL PRIMARY KEY,
			subscriber_id INTEGER NOT NULL REFERENCES t_newsletter_subscriber(id) ON DELETE CASCADE,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'queued',
			attempts INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT '',
			sent_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (subscriber_id, article_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_delivery_status ON t_newsletter_delivery(status, id)`,
		`CREATE TABLE IF NOT EXISTS t_growth_event (
			id SERIAL PRIMARY KEY,
			event_name VARCHAR(32) NOT NULL,
			article_id INTEGER NULL REFERENCES t_article(id) ON DELETE SET NULL,
			path VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_growth_event_created ON t_growth_event(created_at, event_name)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply growth schema: %w", err)
		}
	}
	if err := seedMenus(session); err != nil {
		return err
	}
	if err := seedResourcesAndPolicies(session); err != nil {
		return err
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 3, "newsletter-and-growth"); err != nil {
		return fmt.Errorf("record growth migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit growth migration: %w", err)
	}
	return nil
}

func applyGrowthOperationsSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 4)").Get(&applied); err != nil {
		return fmt.Errorf("check growth operations migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin growth operations migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS idx_newsletter_delivery_updated ON t_newsletter_delivery(status, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_newsletter_delivery_sent ON t_newsletter_delivery(sent_at)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply growth operations schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 4, "growth-operations"); err != nil {
		return fmt.Errorf("record growth operations migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit growth operations migration: %w", err)
	}
	return nil
}

// applyArticleContentSchema adds the optional HTML representation used by the
// new editor. Legacy Markdown stays in article_content and is intentionally
// not rewritten during deployment; the public API can fall back to it.
func applyArticleContentSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 5)").Get(&applied); err != nil {
		return fmt.Errorf("check article content migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin article content migration: %w", err)
	}
	defer session.Rollback()
	if _, err := session.Exec(`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS article_content_html TEXT NULL`); err != nil {
		return fmt.Errorf("apply article content migration: %w", err)
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 5, "article-content-html"); err != nil {
		return fmt.Errorf("record article content migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit article content migration: %w", err)
	}
	return nil
}

// applyArticleReactionSchema adds the reader-interaction ledger. Reactions are
// stored per account and article; the aggregate counts are derived on read so
// the table stays the single source of truth.
func applyArticleReactionSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 6)").Get(&applied); err != nil {
		return fmt.Errorf("check article reaction migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin article reaction migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS t_article_reaction (
			id SERIAL PRIMARY KEY,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			user_info_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			reaction VARCHAR(16) NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (article_id, user_info_id, reaction)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_article_reaction_article ON t_article_reaction(article_id, reaction)`,
		`CREATE INDEX IF NOT EXISTS idx_article_reaction_user ON t_article_reaction(user_info_id, reaction, id DESC)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply article reaction schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 6, "article-reactions"); err != nil {
		return fmt.Errorf("record article reaction migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit article reaction migration: %w", err)
	}
	return nil
}

// applyCommentNotificationSchema adds the per-account opt-out used by comment
// notification emails. The site-wide switch already lives in the website
// configuration, so only the account preference needs a column.
func applyCommentNotificationSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 7)").Get(&applied); err != nil {
		return fmt.Errorf("check comment notification migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin comment notification migration: %w", err)
	}
	defer session.Rollback()
	if _, err := session.Exec(`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS notify_comment SMALLINT NOT NULL DEFAULT 1`); err != nil {
		return fmt.Errorf("apply comment notification schema: %w", err)
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 7, "comment-notification"); err != nil {
		return fmt.Errorf("record comment notification migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit comment notification migration: %w", err)
	}
	return nil
}

// applySeriesSchema adds ordered article collections plus the admin menu entry
// that manages them.
func applySeriesSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 8)").Get(&applied); err != nil {
		return fmt.Errorf("check series migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin series migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS t_series (
			id SERIAL PRIMARY KEY,
			series_name VARCHAR(50) NOT NULL UNIQUE,
			series_desc VARCHAR(255) NOT NULL DEFAULT '',
			cover VARCHAR(1024) NOT NULL DEFAULT '',
			is_delete SMALLINT NOT NULL DEFAULT 0,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS series_id INTEGER NULL`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS series_order INTEGER NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS idx_article_series ON t_article(series_id, series_order, id)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply series schema: %w", err)
		}
	}
	if err := seedMenus(session); err != nil {
		return err
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 8, "article-series"); err != nil {
		return fmt.Errorf("record series migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit series migration: %w", err)
	}
	return nil
}

// applyFriendLinkReviewSchema turns friend links into a reviewed resource:
// reader submissions land as pending and only approved links stay public.
// Historical rows are approved by definition, which the default preserves.
func applyFriendLinkReviewSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 9)").Get(&applied); err != nil {
		return fmt.Errorf("check friend link review migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin friend link review migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`ALTER TABLE t_friend_link ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 1`,
		`ALTER TABLE t_friend_link ADD COLUMN IF NOT EXISTS applicant_email VARCHAR(254) NOT NULL DEFAULT ''`,
		`ALTER TABLE t_friend_link ADD COLUMN IF NOT EXISTS audit_time TIMESTAMPTZ NULL`,
		`ALTER TABLE t_friend_link ALTER COLUMN link_address TYPE VARCHAR(255)`,
		`CREATE INDEX IF NOT EXISTS idx_friend_link_status ON t_friend_link(status, id)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply friend link review schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 9, "friend-link-review"); err != nil {
		return fmt.Errorf("record friend link review migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit friend link review migration: %w", err)
	}
	return nil
}

// applyScheduledPublishSchema adds the release timestamp used by status 4
// (scheduled). Scheduled articles stay invisible because every public query
// only reads status 1 and 2.
func applyScheduledPublishSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 10)").Get(&applied); err != nil {
		return fmt.Errorf("check scheduled publish migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin scheduled publish migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ NULL`,
		`CREATE INDEX IF NOT EXISTS idx_article_scheduled ON t_article(status, scheduled_at)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply scheduled publish schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 10, "scheduled-publish"); err != nil {
		return fmt.Errorf("record scheduled publish migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit scheduled publish migration: %w", err)
	}
	return nil
}

// applyJobSchedulerSchema converts the legacy Quartz-shaped task table to the
// standard five-field scheduler. Known targets are migrated and unknown
// targets remain visible but paused so custom configurations are not lost.
func applyJobSchedulerSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 11)").Get(&applied); err != nil {
		return fmt.Errorf("check job scheduler migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin job scheduler migration: %w", err)
	}
	defer session.Rollback()

	var jobs []entity.TJob
	if err := session.OrderBy("id ASC").Find(&jobs); err != nil {
		return fmt.Errorf("load legacy jobs: %w", err)
	}
	knownTargets := map[string]string{
		"auroraQuartz.statisticalUserArea": "userArea.refresh",
		"auroraQuartz.clearJobLogs":        "jobLogs.cleanup",
	}
	registered := map[string]bool{
		"article.publishScheduled": true,
		"growth.cleanup":           true,
		"jobLogs.cleanup":          true,
		"userArea.refresh":         true,
	}
	for _, job := range jobs {
		target := strings.TrimSpace(job.InvokeTarget)
		if migrated, ok := knownTargets[target]; ok {
			target = migrated
		}
		expression := normalizeLegacyCron(job.CronExpression)
		status := job.Status
		remark := job.Remark
		if !registered[target] || !validStandardCron(expression) {
			status = 0
			remark = appendMigrationRemark(remark, "migration 11: target or cron is not supported")
		}
		if _, err := session.Exec(`
			UPDATE t_job
			SET invoke_target = ?, cron_expression = ?, misfire_policy = 3,
			    concurrent = 0, status = ?, remark = ?, update_time = CURRENT_TIMESTAMP
			WHERE id = ?
		`, target, expression, status, remark, job.Id); err != nil {
			return fmt.Errorf("migrate job %d: %w", job.Id, err)
		}
	}

	seedJobs := []struct {
		name, group, target, expression, remark string
	}{
		{"发布定时文章", "系统", "article.publishScheduled", "* * * * *", "每分钟发布到期的定时文章"},
		{"清理增长事件", "系统", "growth.cleanup", "20 3 * * *", "每天删除 180 天前的增长事件"},
		{"清理任务日志", "系统", "jobLogs.cleanup", "0 4 * * *", "每天清理任务执行日志"},
		{"刷新用户地域统计", "系统", "userArea.refresh", "*/30 * * * *", "每 30 分钟刷新用户地域分布"},
	}
	for _, seed := range seedJobs {
		var exists bool
		if _, err := session.SQL("SELECT EXISTS (SELECT 1 FROM t_job WHERE invoke_target = ?)", seed.target).Get(&exists); err != nil {
			return fmt.Errorf("check default job %q: %w", seed.target, err)
		}
		if exists {
			continue
		}
		if _, err := session.Insert(&entity.TJob{
			JobName: seed.name, JobGroup: seed.group, InvokeTarget: seed.target,
			CronExpression: seed.expression, MisfirePolicy: 3, Concurrent: 0,
			Status: 1, Remark: seed.remark,
		}); err != nil {
			return fmt.Errorf("seed default job %q: %w", seed.target, err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 11, "job-scheduler"); err != nil {
		return fmt.Errorf("record job scheduler migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit job scheduler migration: %w", err)
	}
	return nil
}

// applyContentAnalyticsSchema adds privacy-preserving daily article metrics
// and the admin entry point for the content performance dashboard.
func applyContentAnalyticsSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 12)").Get(&applied); err != nil {
		return fmt.Errorf("check content analytics migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin content analytics migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS t_article_daily_metric (
			id SERIAL PRIMARY KEY,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			metric_date DATE NOT NULL,
			views BIGINT NOT NULL DEFAULT 0,
			unique_readers BIGINT NOT NULL DEFAULT 0,
			effective_sessions BIGINT NOT NULL DEFAULT 0,
			total_active_ms BIGINT NOT NULL DEFAULT 0,
			completed_sessions BIGINT NOT NULL DEFAULT 0,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (article_id, metric_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_article_daily_metric_date ON t_article_daily_metric(metric_date)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply content analytics schema: %w", err)
		}
	}
	if err := seedMenus(session); err != nil {
		return err
	}
	if err := seedResourcesAndPolicies(session); err != nil {
		return err
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 12, "content-analytics"); err != nil {
		return fmt.Errorf("record content analytics migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit content analytics migration: %w", err)
	}
	return nil
}

// applyContinuationAnalyticsSchema adds counters for the two continuation
// surfaces on an article page. The counters stay anonymous and aggregate at
// the same per-article daily grain as the rest of content performance.
func applyContinuationAnalyticsSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 13)").Get(&applied); err != nil {
		return fmt.Errorf("check continuation analytics migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin continuation analytics migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`ALTER TABLE t_article_daily_metric ADD COLUMN IF NOT EXISTS series_impressions BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE t_article_daily_metric ADD COLUMN IF NOT EXISTS series_clicks BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE t_article_daily_metric ADD COLUMN IF NOT EXISTS related_impressions BIGINT NOT NULL DEFAULT 0`,
		`ALTER TABLE t_article_daily_metric ADD COLUMN IF NOT EXISTS related_clicks BIGINT NOT NULL DEFAULT 0`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply continuation analytics schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 13, "continuation-analytics"); err != nil {
		return fmt.Errorf("record continuation analytics migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit continuation analytics migration: %w", err)
	}
	return nil
}

// applyContinuationTargetSchema stores click attribution for the concrete
// recommendation target while keeping the source article aggregate counters.
func applyContinuationTargetSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 14)").Get(&applied); err != nil {
		return fmt.Errorf("check continuation target migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin continuation target migration: %w", err)
	}
	defer session.Rollback()
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS t_article_continuation_target (
			id SERIAL PRIMARY KEY,
			source_article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			target_type VARCHAR(16) NOT NULL,
			target_id INTEGER NOT NULL,
			placement VARCHAR(24) NOT NULL,
			position SMALLINT NOT NULL DEFAULT 0,
			metric_date DATE NOT NULL,
			clicks BIGINT NOT NULL DEFAULT 0,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (source_article_id, target_type, target_id, placement, position, metric_date)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_continuation_target_source_date ON t_article_continuation_target(source_article_id, metric_date)`,
		`CREATE INDEX IF NOT EXISTS idx_continuation_target_target_date ON t_article_continuation_target(target_type, target_id, metric_date)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply continuation target schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 14, "continuation-targets"); err != nil {
		return fmt.Errorf("record continuation target migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit continuation target migration: %w", err)
	}
	return nil
}

// applyMultiUserSchema introduces public author identities, owner-scoped
// taxonomy, moderation metadata and the public/private content boundary.
func applyMultiUserSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 15)").Get(&applied); err != nil {
		return fmt.Errorf("check multi-user migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin multi-user migration: %w", err)
	}
	defer session.Rollback()

	for _, statement := range []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS handle VARCHAR(40)`,
		`ALTER TABLE t_category ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_tag ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 1`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`UPDATE t_article SET status = 1 WHERE status = 2 AND COALESCE(password, '') <> ''`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply multi-user schema: %w", err)
		}
	}

	var ownerID int
	if _, err := session.SQL(`
		SELECT ui.id
		FROM t_user_info ui
		JOIN t_user_role ur ON ur.user_id = ui.id
		JOIN t_role r ON r.id = ur.role_id
		WHERE r.role_name = 'admin'
		ORDER BY ui.id ASC
		LIMIT 1`).Get(&ownerID); err != nil {
		return fmt.Errorf("find legacy content owner: %w", err)
	}
	if ownerID == 0 {
		if _, err := session.SQL(`SELECT id FROM t_user_info ORDER BY id ASC LIMIT 1`).Get(&ownerID); err != nil {
			return fmt.Errorf("find fallback content owner: %w", err)
		}
	}

	var users []entity.TUserInfo
	if err := session.Cols("id", "handle", "nickname", "email").OrderBy("id ASC").Find(&users); err != nil {
		return fmt.Errorf("load users for handles: %w", err)
	}
	usedHandles := make(map[string]struct{}, len(users))
	for _, user := range users {
		if handle := strings.TrimSpace(strings.ToLower(user.Handle)); validHandle(handle) {
			usedHandles[handle] = struct{}{}
		}
	}
	for _, user := range users {
		if strings.TrimSpace(user.Handle) != "" {
			continue
		}
		base := handleBase(user.Nickname)
		if base == "" {
			base = handleBase(strings.Split(user.Email, "@")[0])
		}
		if base == "" {
			base = fmt.Sprintf("user-%d", user.Id)
		}
		base = truncateHandle(base, 32)
		handle := base
		for suffix := 2; ; suffix++ {
			if _, exists := usedHandles[handle]; !exists {
				break
			}
			handle = fmt.Sprintf("%s-%d", base, suffix)
		}
		usedHandles[handle] = struct{}{}
		if _, err := session.ID(user.Id).Cols("handle").Update(&entity.TUserInfo{Handle: handle}); err != nil {
			return fmt.Errorf("backfill user handle %d: %w", user.Id, err)
		}
	}

	if ownerID > 0 {
		for _, statement := range []string{
			`UPDATE t_article SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_series SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_category SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_tag SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
		} {
			if _, err := session.Exec(statement, ownerID); err != nil {
				return fmt.Errorf("backfill content owner: %w", err)
			}
		}
	}

	type duplicateName struct {
		UserId int    `xorm:"user_id"`
		Name   string `xorm:"normalized_name"`
		Id     int    `xorm:"id"`
	}
	var categoryDuplicates []duplicateName
	if err := session.SQL(`
		SELECT user_id, lower(btrim(category_name)) AS normalized_name, id
		FROM (
			SELECT id, user_id, category_name,
			       row_number() OVER (PARTITION BY user_id, lower(btrim(category_name)) ORDER BY id) AS rn
			FROM t_category
		) ranked
		WHERE rn > 1`).Find(&categoryDuplicates); err != nil {
		return fmt.Errorf("find duplicate categories: %w", err)
	}
	for _, duplicate := range categoryDuplicates {
		var keepID int
		if _, err := session.SQL(`
			SELECT id FROM t_category
			WHERE user_id = ? AND lower(btrim(category_name)) = ?
			ORDER BY id ASC LIMIT 1`, duplicate.UserId, duplicate.Name).Get(&keepID); err != nil {
			return fmt.Errorf("resolve duplicate category: %w", err)
		}
		if keepID == 0 || keepID == duplicate.Id {
			continue
		}
		if _, err := session.Exec(`UPDATE t_article SET category_id = ? WHERE category_id = ?`, keepID, duplicate.Id); err != nil {
			return fmt.Errorf("remap duplicate category: %w", err)
		}
		if _, err := session.Exec(`DELETE FROM t_category WHERE id = ?`, duplicate.Id); err != nil {
			return fmt.Errorf("delete duplicate category: %w", err)
		}
	}

	var tagDuplicates []duplicateName
	if err := session.SQL(`
		SELECT user_id, lower(btrim(tag_name)) AS normalized_name, id
		FROM (
			SELECT id, user_id, tag_name,
			       row_number() OVER (PARTITION BY user_id, lower(btrim(tag_name)) ORDER BY id) AS rn
			FROM t_tag
		) ranked
		WHERE rn > 1`).Find(&tagDuplicates); err != nil {
		return fmt.Errorf("find duplicate tags: %w", err)
	}
	for _, duplicate := range tagDuplicates {
		var keepID int
		if _, err := session.SQL(`
			SELECT id FROM t_tag
			WHERE user_id = ? AND lower(btrim(tag_name)) = ?
			ORDER BY id ASC LIMIT 1`, duplicate.UserId, duplicate.Name).Get(&keepID); err != nil {
			return fmt.Errorf("resolve duplicate tag: %w", err)
		}
		if keepID == 0 || keepID == duplicate.Id {
			continue
		}
		if _, err := session.Exec(`UPDATE t_article_tag SET tag_id = ? WHERE tag_id = ?`, keepID, duplicate.Id); err != nil {
			return fmt.Errorf("remap duplicate tag: %w", err)
		}
		if _, err := session.Exec(`DELETE FROM t_tag WHERE id = ?`, duplicate.Id); err != nil {
			return fmt.Errorf("delete duplicate tag: %w", err)
		}
	}
	if _, err := session.Exec(`
		DELETE FROM t_article_tag
		WHERE id IN (
			SELECT id FROM (
				SELECT id, row_number() OVER (PARTITION BY article_id, tag_id ORDER BY id) AS rn
				FROM t_article_tag
			) ranked WHERE rn > 1
		)`); err != nil {
		return fmt.Errorf("deduplicate article tags: %w", err)
	}

	for _, statement := range []string{
		`ALTER TABLE t_user_info ALTER COLUMN handle SET NOT NULL`,
		`ALTER TABLE t_category ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_series ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_tag ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_series DROP CONSTRAINT IF EXISTS t_series_series_name_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_info_handle_lower ON t_user_info (lower(handle))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_series_owner_name ON t_series (user_id, lower(btrim(series_name))) WHERE is_delete = 0`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_category_owner_name ON t_category (user_id, lower(btrim(category_name)))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_tag_owner_name ON t_tag (user_id, lower(btrim(tag_name)))`,
		`CREATE INDEX IF NOT EXISTS idx_article_public_owner ON t_article(status, moderation_status, user_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_talk_public_owner ON t_talk(status, moderation_status, user_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_series_public_owner ON t_series(status, moderation_status, user_id, id DESC)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("finalize multi-user schema: %w", err)
		}
	}

	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 15, "multi-user-platform"); err != nil {
		return fmt.Errorf("record multi-user migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit multi-user migration: %w", err)
	}
	return nil
}

// applyContentModerationSchema adds the unified admin moderation menu. The
// moderation fields themselves were introduced by migration 15.
func applyContentModerationSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 16)").Get(&applied); err != nil {
		return fmt.Errorf("check content moderation migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin content moderation migration: %w", err)
	}
	defer session.Rollback()
	if err := seedMenus(session); err != nil {
		return err
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 16, "content-moderation"); err != nil {
		return fmt.Errorf("record content moderation migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit content moderation migration: %w", err)
	}
	return nil
}

// applyStudioOperationsSchema adds durable scheduled-publish outcomes and an
// append-only audit trail for owner-facing batch mutations.
func applyStudioOperationsSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 17)").Get(&applied); err != nil {
		return fmt.Errorf("check studio operations migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin studio operations migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_article_publish_record (
			id BIGSERIAL PRIMARY KEY,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL,
			scheduled_at TIMESTAMPTZ NOT NULL,
			published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			notification_state VARCHAR(20) NOT NULL DEFAULT 'pending',
			notification_attempts INTEGER NOT NULL DEFAULT 0,
			next_retry_at TIMESTAMPTZ NULL,
			last_error TEXT NOT NULL DEFAULT '',
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (article_id, scheduled_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_article_publish_record_user_published ON t_article_publish_record(user_id, published_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_article_publish_record_retry ON t_article_publish_record(notification_state, next_retry_at)`,
		`CREATE TABLE IF NOT EXISTS t_content_operation_audit (
			id BIGSERIAL PRIMARY KEY,
			operator_id INTEGER NOT NULL,
			operator_nickname VARCHAR(64) NOT NULL DEFAULT '',
			content_type VARCHAR(16) NOT NULL,
			operation VARCHAR(32) NOT NULL,
			target_mode VARCHAR(16) NOT NULL,
			filter_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
			snapshot_max_id INTEGER NOT NULL DEFAULT 0,
			requested_count INTEGER NOT NULL DEFAULT 0,
			affected_count INTEGER NOT NULL DEFAULT 0,
			result VARCHAR(16) NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			ip_address VARCHAR(255) NOT NULL DEFAULT '',
			ip_source VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_content_operation_audit_created ON t_content_operation_audit(created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_content_operation_audit_filters ON t_content_operation_audit(content_type, operation, result)`,
		`CREATE TABLE IF NOT EXISTS t_content_operation_audit_item (
			id BIGSERIAL PRIMARY KEY,
			audit_id BIGINT NOT NULL REFERENCES t_content_operation_audit(id) ON DELETE CASCADE,
			content_id INTEGER NOT NULL,
			title VARCHAR(255) NOT NULL DEFAULT '',
			previous_status SMALLINT NOT NULL,
			next_status SMALLINT NOT NULL,
			result VARCHAR(16) NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_content_operation_audit_item_audit ON t_content_operation_audit_item(audit_id, id)`,
		`UPDATE t_job SET remark = '每天清理 30 天前的任务日志', update_time = CURRENT_TIMESTAMP WHERE invoke_target = 'jobLogs.cleanup' AND remark = '每天清理任务日志'`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply studio operations schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 17, "studio-operations"); err != nil {
		return fmt.Errorf("record studio operations migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit studio operations migration: %w", err)
	}
	return nil
}
func applyFollowSocialSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 18)").Get(&applied); err != nil {
		return fmt.Errorf("check follow social migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin follow social migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_user_follow (
			id BIGSERIAL PRIMARY KEY,
			follower_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			author_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			start_event_id BIGINT NOT NULL DEFAULT 0,
			last_read_event_id BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (follower_id, author_id),
			CHECK (follower_id <> author_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_follow_follower ON t_user_follow(follower_id, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_follow_author ON t_user_follow(author_id, created_at DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS t_author_publish_event (
			id BIGSERIAL PRIMARY KEY,
			author_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			content_type VARCHAR(16) NOT NULL,
			content_id BIGINT NOT NULL,
			published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (content_type, content_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_author_publish_event_timeline ON t_author_publish_event(author_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_author_publish_event_published ON t_author_publish_event(published_at DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply follow social schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 18, "author-following"); err != nil {
		return fmt.Errorf("record follow social migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit follow social migration: %w", err)
	}
	return nil
}
func applyInteractionNotificationSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 19)").Get(&applied); err != nil {
		return fmt.Errorf("check interaction notification migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin interaction notification migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS notify_interaction SMALLINT NOT NULL DEFAULT 1`,
		`ALTER TABLE t_comment ADD COLUMN IF NOT EXISTS notification_dispatched_at TIMESTAMPTZ NULL`,
		`CREATE TABLE IF NOT EXISTS t_user_notification (
			id BIGSERIAL PRIMARY KEY,
			recipient_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			actor_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			type VARCHAR(16) NOT NULL,
			content_type VARCHAR(16) NOT NULL,
			content_id BIGINT NOT NULL,
			comment_id BIGINT NOT NULL DEFAULT 0,
			dedupe_key VARCHAR(191) NOT NULL,
			read_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (recipient_id, dedupe_key),
			CHECK (recipient_id <> actor_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_notification_recipient ON t_user_notification(recipient_id, created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_user_notification_unread ON t_user_notification(recipient_id, read_at, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply interaction notification schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 19, "interaction-notifications"); err != nil {
		return fmt.Errorf("record interaction notification migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit interaction notification migration: %w", err)
	}
	return nil
}

// applyDiscoveryRankingSchema adds the covering indexes the public discovery
// surfaces rely on. Trending feeds aggregate reactions, approved comments and
// daily read metrics over a short window, so each table needs an index that
// keeps the window scan off a full table scan. It only adds indexes and never
// rewrites or deletes existing data.
func applyDiscoveryRankingSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 20)").Get(&applied); err != nil {
		return fmt.Errorf("check discovery ranking migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin discovery ranking migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_comment_type_topic_created ON t_comment(type, topic_id, create_time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_article_daily_metric_window ON t_article_daily_metric(metric_date, article_id)`,
		`CREATE INDEX IF NOT EXISTS idx_article_reaction_article_created ON t_article_reaction(article_id, reaction, create_time DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply discovery ranking schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 20, "discovery-ranking"); err != nil {
		return fmt.Errorf("record discovery ranking migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit discovery ranking migration: %w", err)
	}
	return nil
}

// applyTopicSubscriptionSchema adds the reader-facing topic subscription ledger.
// A subscription is keyed by the normalised topic name rather than a taxonomy id,
// because public categories and tags are per author while the plaza aggregates
// them by name. Only article ticks are published, so talks need no key.
func applyTopicSubscriptionSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 21)").Get(&applied); err != nil {
		return fmt.Errorf("check topic subscription migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin topic subscription migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS notify_topic SMALLINT NOT NULL DEFAULT 1`,
		`CREATE TABLE IF NOT EXISTS t_topic_subscription (
			id BIGSERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			topic_type VARCHAR(16) NOT NULL,
			topic_key VARCHAR(64) NOT NULL,
			topic_name VARCHAR(50) NOT NULL,
			muted SMALLINT NOT NULL DEFAULT 0,
			start_event_id BIGINT NOT NULL DEFAULT 0,
			last_read_event_id BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id, topic_type, topic_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_topic_subscription_user ON t_topic_subscription(user_id, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_topic_subscription_topic ON t_topic_subscription(topic_type, topic_key)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply topic subscription schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 21, "topic-subscriptions"); err != nil {
		return fmt.Errorf("record topic subscription migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit topic subscription migration: %w", err)
	}
	return nil
}

// applyRecommendationFeedbackSchema adds the reader-controlled feedback ledger
// used by the personalised discovery feed. Article feedback is a hard hide;
// author and topic feedback are stored as ranking penalties and can be undone.
func applyRecommendationFeedbackSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 22)").Get(&applied); err != nil {
		return fmt.Errorf("check recommendation feedback migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin recommendation feedback migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_recommendation_feedback (
			id BIGSERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			target_type VARCHAR(16) NOT NULL,
			target_key VARCHAR(160) NOT NULL,
			article_id INTEGER NULL REFERENCES t_article(id) ON DELETE CASCADE,
			author_id INTEGER NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			topic_type VARCHAR(16) NULL,
			topic_key VARCHAR(64) NULL,
			target_label VARCHAR(255) NOT NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id, target_type, target_key),
			CHECK (target_type IN ('article', 'author', 'topic')),
			CHECK (
				(target_type = 'article' AND article_id IS NOT NULL AND author_id IS NULL AND topic_type IS NULL AND topic_key IS NULL)
				OR (target_type = 'author' AND article_id IS NULL AND author_id IS NOT NULL AND topic_type IS NULL AND topic_key IS NULL)
				OR (target_type = 'topic' AND article_id IS NULL AND author_id IS NULL AND topic_type IN ('category', 'tag') AND topic_key IS NOT NULL)
			)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_recommendation_feedback_user ON t_recommendation_feedback(user_id, create_time DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply recommendation feedback schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 22, "recommendation-feedback"); err != nil {
		return fmt.Errorf("record recommendation feedback migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit recommendation feedback migration: %w", err)
	}
	return nil
}

// applyCollectionSchema adds reader-curated article collections. Collections
// are owner-scoped; public and unlisted items remain references so article
// withdrawal and moderation can never be bypassed.
func applyCollectionSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 23)").Get(&applied); err != nil {
		return fmt.Errorf("check collection migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin collection migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_collection (
			id BIGSERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			slug VARCHAR(80) NOT NULL UNIQUE,
			title VARCHAR(80) NOT NULL,
			description VARCHAR(500) NOT NULL DEFAULT '',
			visibility VARCHAR(16) NOT NULL DEFAULT 'private',
			moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible',
			moderation_reason VARCHAR(255) NOT NULL DEFAULT '',
			moderated_by INTEGER NOT NULL DEFAULT 0,
			moderated_at TIMESTAMPTZ NULL,
			is_delete SMALLINT NOT NULL DEFAULT 0,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (visibility IN ('private', 'unlisted', 'public')),
			CHECK (moderation_status IN ('visible', 'hidden'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_owner ON t_collection(user_id, is_delete, update_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_public ON t_collection(visibility, moderation_status, is_delete, update_time DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS t_collection_item (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL REFERENCES t_collection(id) ON DELETE CASCADE,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			note VARCHAR(280) NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (collection_id, article_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_item_order ON t_collection_item(collection_id, sort_order, id)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_item_article ON t_collection_item(article_id, collection_id)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply collection schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 23, "reader-collections"); err != nil {
		return fmt.Errorf("record collection migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit collection migration: %w", err)
	}
	return nil
}

// applyCollectionSubscriptionSchema adds reader subscriptions and the append-
// only update stream used by the in-app feed and notification inbox. Updates
// remain references, so private or moderated collections disappear at read
// time without deleting subscriber state.
func applyCollectionSubscriptionSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 24)").Get(&applied); err != nil {
		return fmt.Errorf("check collection subscription migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin collection subscription migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS notify_collection SMALLINT NOT NULL DEFAULT 1`,
		`CREATE TABLE IF NOT EXISTS t_collection_update_event (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL REFERENCES t_collection(id) ON DELETE CASCADE,
			owner_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_update_event_stream ON t_collection_update_event(collection_id, created_at DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS t_collection_subscription (
			id BIGSERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			collection_id BIGINT NOT NULL REFERENCES t_collection(id) ON DELETE CASCADE,
			muted SMALLINT NOT NULL DEFAULT 0,
			start_event_id BIGINT NOT NULL DEFAULT 0,
			last_read_event_id BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id, collection_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_subscription_user ON t_collection_subscription(user_id, updated_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_subscription_collection ON t_collection_subscription(collection_id)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply collection subscription schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 24, "collection-subscriptions"); err != nil {
		return fmt.Errorf("record collection subscription migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit collection subscription migration: %w", err)
	}
	return nil
}

// applyCollectionInteractionSchema adds direct likes for reader-curated
// collections. Comments reuse t_comment with type 6, so only the reaction
// ledger needs a new table.
func applyCollectionInteractionSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 25)").Get(&applied); err != nil {
		return fmt.Errorf("check collection interaction migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin collection interaction migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_collection_reaction (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL REFERENCES t_collection(id) ON DELETE CASCADE,
			user_info_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			reaction VARCHAR(16) NOT NULL DEFAULT 'like',
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (reaction = 'like'),
			UNIQUE (collection_id, user_info_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_reaction_collection ON t_collection_reaction(collection_id, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_collection_reaction_user ON t_collection_reaction(user_info_id, create_time DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply collection interaction schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 25, "collection-interactions"); err != nil {
		return fmt.Errorf("record collection interaction migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit collection interaction migration: %w", err)
	}
	return nil
}

// applyCollectionCommunitySchema extends bookmarks to reader-curated
// collections and adds a ledger for likes on comments and replies.
func applyCollectionCommunitySchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 26)").Get(&applied); err != nil {
		return fmt.Errorf("check collection community migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin collection community migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`ALTER TABLE t_collection_reaction DROP CONSTRAINT IF EXISTS t_collection_reaction_collection_id_user_info_id_key`,
		`ALTER TABLE t_collection_reaction DROP CONSTRAINT IF EXISTS t_collection_reaction_reaction_check`,
		`ALTER TABLE t_collection_reaction ADD CONSTRAINT t_collection_reaction_reaction_check CHECK (reaction IN ('like', 'favorite'))`,
		`ALTER TABLE t_collection_reaction ADD CONSTRAINT t_collection_reaction_collection_user_reaction_key UNIQUE (collection_id, user_info_id, reaction)`,
		`CREATE TABLE IF NOT EXISTS t_comment_reaction (
			id BIGSERIAL PRIMARY KEY,
			comment_id BIGINT NOT NULL REFERENCES t_comment(id) ON DELETE CASCADE,
			user_info_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			reaction VARCHAR(16) NOT NULL DEFAULT 'like',
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (reaction = 'like'),
			UNIQUE (comment_id, user_info_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_reaction_comment ON t_comment_reaction(comment_id, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_reaction_user ON t_comment_reaction(user_info_id, create_time DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply collection community schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 26, "collection-community"); err != nil {
		return fmt.Errorf("record collection community migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit collection community migration: %w", err)
	}
	return nil
}

// applyCommentModerationSchema adds the root-comment pin marker used by
// collection owners. The partial unique index enforces one pinned root per
// content target without affecting replies or deleted comments.
func applyCommentModerationSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 27)").Get(&applied); err != nil {
		return fmt.Errorf("check comment moderation migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin comment moderation migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`ALTER TABLE t_comment ADD COLUMN IF NOT EXISTS is_top SMALLINT NOT NULL DEFAULT 0`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_comment_one_pinned_root ON t_comment(type, topic_id)
			WHERE is_top = 1 AND parent_id = 0 AND is_delete = 0`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply comment moderation schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 27, "comment-moderation"); err != nil {
		return fmt.Errorf("record comment moderation migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit comment moderation migration: %w", err)
	}
	return nil
}

// applyCommentGovernanceSchema adds the moderation audit trail, reader reports
// and appeal workflow used by reading-list comment governance. The audit table
// keeps before/after snapshots so an accidental delete can be traced and
// reversed without relying on the generic HTTP operation log.
func applyCommentGovernanceSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 28)").Get(&applied); err != nil {
		return fmt.Errorf("check comment governance migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin comment governance migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_comment_moderation_log (
			id BIGSERIAL PRIMARY KEY,
			comment_id BIGINT NOT NULL,
			collection_id INTEGER NOT NULL,
			actor_id INTEGER NOT NULL,
			actor_role VARCHAR(16) NOT NULL,
			action VARCHAR(32) NOT NULL,
			before_snapshot JSONB,
			after_snapshot JSONB,
			cascade_ids BIGINT[],
			batch_id UUID,
			reason VARCHAR(255) NOT NULL DEFAULT '',
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (actor_role IN ('owner', 'admin', 'commenter', 'reader')),
			CHECK (action IN ('pin', 'unpin', 'delete', 'restore', 'report', 'report_dismiss', 'report_hide', 'appeal_submit', 'appeal_reject', 'appeal_escalate'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_moderation_collection ON t_comment_moderation_log(collection_id, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_moderation_comment ON t_comment_moderation_log(comment_id, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_moderation_batch ON t_comment_moderation_log(batch_id)`,
		`CREATE TABLE IF NOT EXISTS t_comment_report (
			id BIGSERIAL PRIMARY KEY,
			comment_id BIGINT NOT NULL REFERENCES t_comment(id) ON DELETE CASCADE,
			collection_id INTEGER NOT NULL,
			reporter_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			reason VARCHAR(32) NOT NULL,
			detail VARCHAR(200) NOT NULL DEFAULT '',
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			handled_by INTEGER NOT NULL DEFAULT 0,
			handled_at TIMESTAMPTZ,
			decision_reason VARCHAR(255) NOT NULL DEFAULT '',
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (reason IN ('spam', 'harassment', 'porn', 'illegal', 'privacy', 'other')),
			CHECK (status IN ('pending', 'dismissed', 'resolved')),
			CHECK (detail = '' OR char_length(detail) <= 200)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_comment_report_pending_unique ON t_comment_report(comment_id, reporter_id) WHERE status = 'pending'`,
		`CREATE INDEX IF NOT EXISTS idx_comment_report_queue ON t_comment_report(collection_id, status, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_report_comment ON t_comment_report(comment_id, create_time DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS t_comment_appeal (
			id BIGSERIAL PRIMARY KEY,
			comment_id BIGINT NOT NULL REFERENCES t_comment(id) ON DELETE CASCADE,
			collection_id INTEGER NOT NULL,
			appellant_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
			reason VARCHAR(500) NOT NULL,
			stage VARCHAR(16) NOT NULL DEFAULT 'owner',
			status VARCHAR(16) NOT NULL DEFAULT 'pending',
			handled_by INTEGER NOT NULL DEFAULT 0,
			handled_at TIMESTAMPTZ,
			decision_reason VARCHAR(255) NOT NULL DEFAULT '',
			escalated_at TIMESTAMPTZ,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (stage IN ('owner', 'admin')),
			CHECK (status IN ('pending', 'restored', 'rejected'))
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_comment_appeal_pending_unique ON t_comment_appeal(comment_id, appellant_id) WHERE status = 'pending'`,
		`CREATE INDEX IF NOT EXISTS idx_comment_appeal_queue ON t_comment_appeal(collection_id, stage, status, create_time DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_appeal_appellant ON t_comment_appeal(appellant_id, create_time DESC, id DESC)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply comment governance schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 28, "comment-governance"); err != nil {
		return fmt.Errorf("record comment governance migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit comment governance migration: %w", err)
	}
	return nil
}

// applyStudioActivationSchema makes the Studio onboarding checklist account
// scoped while preserving pre-existing complete creators outside the new-user
// funnel by allowing started_at to remain null.
func applyStudioActivationSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 29)").Get(&applied); err != nil {
		return fmt.Errorf("check studio activation migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin studio activation migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS t_studio_activation (
			user_id INTEGER PRIMARY KEY REFERENCES t_user_info(id) ON DELETE CASCADE,
			collapsed SMALLINT NOT NULL DEFAULT 0,
			started_at TIMESTAMPTZ NULL,
			identity_completed_at TIMESTAMPTZ NULL,
			content_completed_at TIMESTAMPTZ NULL,
			profile_visited_at TIMESTAMPTZ NULL,
			completed_at TIMESTAMPTZ NULL,
			create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CHECK (collapsed IN (0, 1))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_studio_activation_started ON t_studio_activation(started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_studio_activation_completed ON t_studio_activation(completed_at)`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply studio activation schema: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 29, "studio-activation"); err != nil {
		return fmt.Errorf("record studio activation migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit studio activation migration: %w", err)
	}
	return nil
}

func handleBase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func validHandle(value string) bool {
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

func truncateHandle(value string, limit int) string {
	value = strings.Trim(value, "-")
	if len(value) <= limit {
		return value
	}
	return strings.Trim(value[:limit], "-")
}
func normalizeLegacyCron(expression string) string {
	fields := strings.Fields(strings.TrimSpace(expression))
	if len(fields) == 6 {
		fields = fields[1:]
	}
	for index := range fields {
		if fields[index] == "?" {
			fields[index] = "*"
		}
	}
	return strings.Join(fields, " ")
}

func validStandardCron(expression string) bool {
	if len(strings.Fields(strings.TrimSpace(expression))) != 5 {
		return false
	}
	_, err := cron.ParseStandard(expression)
	return err == nil
}

func appendMigrationRemark(remark, addition string) string {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return addition
	}
	value := remark + "; " + addition
	if len(value) > 500 {
		return value[:500]
	}
	return value
}

func seedMenus(session *xorm.Session) error {
	menuIDs := make(map[string]int, len(defaultMenus))
	for _, seed := range defaultMenus {
		var current entity.TMenu
		found, err := session.Where("path = ?", seed.path).Get(&current)
		if err != nil {
			return fmt.Errorf("find default menu %q: %w", seed.path, err)
		}
		if found {
			menuIDs[seed.path] = current.Id
			continue
		}
		parentID := 0
		if seed.parentPath != "" {
			var parent entity.TMenu
			found, err := session.Where("path = ?", seed.parentPath).Get(&parent)
			if err != nil {
				return fmt.Errorf("find parent menu %q: %w", seed.parentPath, err)
			}
			if !found {
				return fmt.Errorf("parent menu %q is missing", seed.parentPath)
			}
			parentID = parent.Id
		}
		menu := entity.TMenu{
			Name: seed.name, Path: seed.path, Component: seed.component, Icon: seed.icon,
			OrderNum: seed.order, ParentId: parentID, IsHidden: seed.hidden,
		}
		if _, err := session.Insert(&menu); err != nil {
			return fmt.Errorf("insert default menu %q: %w", seed.path, err)
		}
		menuIDs[seed.path] = menu.Id
	}
	for _, seed := range defaultMenus {
		menuID := menuIDs[seed.path]
		if _, err := session.Exec(`
			INSERT INTO t_role_menu (role_id, menu_id)
			SELECT 1, ?
			WHERE NOT EXISTS (SELECT 1 FROM t_role_menu WHERE role_id = 1 AND menu_id = ?)
		`, menuID, menuID); err != nil {
			return fmt.Errorf("grant default admin menu: %w", err)
		}
	}
	return nil
}

func seedResourcesAndPolicies(session *xorm.Session) error {
	var root entity.TResource
	found, err := session.Where("resource_name = ? AND parent_id = 0", "后台管理接口").Get(&root)
	if err != nil {
		return fmt.Errorf("find default resource group: %w", err)
	}
	if !found {
		root = entity.TResource{ResourceName: "后台管理接口", IsAnonymous: 0}
		if _, err := session.Insert(&root); err != nil {
			return fmt.Errorf("insert default resource group: %w", err)
		}
	}
	for _, route := range []struct {
		name   string
		path   string
		method string
	}{
		{"查看管理数据", "/admin/*", "GET"}, {"创建管理数据", "/admin/*", "POST"},
		{"修改管理数据", "/admin/*", "PUT"}, {"删除管理数据", "/admin/*", "DELETE"},
		{"查看管理首页", "/admin", "GET"}, {"创建管理首页数据", "/admin", "POST"},
		{"修改管理首页数据", "/admin", "PUT"}, {"删除管理首页数据", "/admin", "DELETE"},
	} {
		var resource entity.TResource
		found, err := session.Where("url = ? AND request_method = ?", route.path, route.method).Get(&resource)
		if err != nil {
			return fmt.Errorf("find default resource %s %s: %w", route.method, route.path, err)
		}
		if !found {
			resource = entity.TResource{
				ResourceName: route.name, Url: route.path, RequestMethod: route.method,
				ParentId: root.Id, IsAnonymous: 0,
			}
			if _, err := session.Insert(&resource); err != nil {
				return fmt.Errorf("insert default resource %s %s: %w", route.method, route.path, err)
			}
		}
		if _, err := session.Exec(`
			INSERT INTO t_role_resource (role_id, resource_id)
			SELECT 1, ?
			WHERE NOT EXISTS (SELECT 1 FROM t_role_resource WHERE role_id = 1 AND resource_id = ?)
		`, resource.Id, resource.Id); err != nil {
			return fmt.Errorf("grant default admin resource: %w", err)
		}
		if _, err := session.Exec(`
			INSERT INTO casbin_rule (p_type, v0, v1, v2, v3, v4, v5)
			SELECT 'p', 'admin', ?, ?, '', '', ''
			WHERE NOT EXISTS (
				SELECT 1 FROM casbin_rule WHERE p_type = 'p' AND v0 = 'admin' AND v1 = ? AND v2 = ?
			)
		`, route.path, route.method, route.path, route.method); err != nil {
			return fmt.Errorf("insert default Casbin policy: %w", err)
		}
	}
	return nil
}

func seedSiteContent(session *xorm.Session) error {
	websiteConfig := model.WebsiteConfigDTO{
		Name: "Stellar Beacon", EnglishName: "Stellar Beacon",
		WebsiteCreateTime: time.Now().Format("2006-01-02"),
		IsCommentReview:   1,
	}
	configJSON, err := json.Marshal(websiteConfig)
	if err != nil {
		return fmt.Errorf("encode default website configuration: %w", err)
	}
	if _, err := session.Exec(`
		INSERT INTO t_website_config (id, config, create_time) OVERRIDING SYSTEM VALUE
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, string(configJSON)); err != nil {
		return fmt.Errorf("insert default website configuration: %w", err)
	}
	aboutJSON, err := json.Marshal(model.AboutDTO{Content: ""})
	if err != nil {
		return fmt.Errorf("encode default about content: %w", err)
	}
	if _, err := session.Exec(`
		INSERT INTO t_about (id, content, create_time) OVERRIDING SYSTEM VALUE
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING
	`, string(aboutJSON)); err != nil {
		return fmt.Errorf("insert default about content: %w", err)
	}
	for _, table := range []string{"t_website_config", "t_about"} {
		if _, err := session.Exec("SELECT setval(pg_get_serial_sequence('" + table + "', 'id'), GREATEST((SELECT MAX(id) FROM " + table + "), 1), true)"); err != nil {
			return fmt.Errorf("advance %s id sequence: %w", table, err)
		}
	}
	return nil
}
