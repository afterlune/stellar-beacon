package migrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/casbin/xorm-adapter/v2"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
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
		new(entity.TRoleMenu), new(entity.TRoleResource), new(entity.TTag),
		new(entity.TTalk), new(entity.TUniqueView), new(entity.TUserAuth),
		new(entity.TUserInfo), new(entity.TUserRole), new(entity.TWebsiteConfig),
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
