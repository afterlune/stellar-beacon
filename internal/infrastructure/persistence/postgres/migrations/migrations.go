package migrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/casbin/xorm-adapter/v2"
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
	if err := applyArticleReactionUniquenessSchema(ctx, engine); err != nil {
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
	if err := applyStudioActivationReminderSchema(ctx, engine); err != nil {
		return err
	}
	if err := applyArticleSearchReconcileJob(ctx, engine); err != nil {
		return err
	}
	if err := applySystemMonitorSchema(ctx, engine); err != nil {
		return err
	}
	if err := applySystemMonitorHistorySchema(ctx, engine); err != nil {
		return err
	}
	if err := applyPerUserLegacyContentSchema(ctx, engine); err != nil {
		return err
	}
	return applyTimestampDefaults(ctx, engine)
}

// applyTimestampDefaults restores database defaults for conventional audit
// timestamps. XORM creates the baseline tables before the SQL migrations, so
// CREATE TABLE IF NOT EXISTS statements cannot add their declared defaults to
// those existing tables. Raw SQL inserts rely on these defaults.
func applyTimestampDefaults(ctx context.Context, engine *xorm.Engine) error {
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	_, err := session.Exec(`
		DO $$
		DECLARE timestamp_column RECORD;
		BEGIN
			FOR timestamp_column IN
				SELECT table_schema, table_name, column_name
				FROM information_schema.columns
				WHERE table_schema = current_schema()
				  AND column_name IN ('create_time', 'update_time', 'created_at', 'updated_at')
				  AND data_type IN ('timestamp with time zone', 'timestamp without time zone')
				  AND column_default IS NULL
			LOOP
				EXECUTE format(
					'ALTER TABLE %I.%I ALTER COLUMN %I SET DEFAULT CURRENT_TIMESTAMP',
					timestamp_column.table_schema,
					timestamp_column.table_name,
					timestamp_column.column_name
				);
			END LOOP;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("apply audit timestamp defaults: %w", err)
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

// applyPerUserLegacyContentSchema moves the remaining single-owner profile,
// guestbook, links, and albums onto the one account whose nickname exactly
// matches the legacy site author. Unmatched material is archived before it is
// removed from public tables.
func applyPerUserLegacyContentSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 34)").Get(&applied); err != nil {
		return fmt.Errorf("check per-user legacy-content migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin per-user legacy-content migration: %w", err)
	}
	defer session.Rollback()

	statements := []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS about TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS profile_links_json JSONB NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE t_photo_album ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`CREATE TABLE IF NOT EXISTS t_legacy_blog_archive (
			id BIGSERIAL PRIMARY KEY,
			source_table VARCHAR(80) NOT NULL,
			source_id BIGINT NOT NULL,
			payload JSONB NOT NULL,
			archived_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TEMP TABLE migration_legacy_site_owner ON COMMIT DROP AS
			WITH configured_author AS (
				SELECT NULLIF(lower(trim(config::jsonb->>'author')), '') AS nickname
				FROM t_website_config WHERE id = 1
			), matches AS (
				SELECT u.id
				FROM t_user_info u CROSS JOIN configured_author a
				WHERE a.nickname IS NOT NULL AND lower(trim(u.nickname)) = a.nickname
			)
			SELECT min(id) AS user_id FROM matches HAVING count(*) = 1`,
		`INSERT INTO t_legacy_blog_archive (source_table, source_id, payload)
			SELECT 't_about', a.id, to_jsonb(a) FROM t_about a
			WHERE trim(COALESCE(a.content, '')) <> '' AND NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`INSERT INTO t_legacy_blog_archive (source_table, source_id, payload)
			SELECT 't_friend_link', f.id, to_jsonb(f) FROM t_friend_link f
			WHERE f.status <> 1 OR NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`INSERT INTO t_legacy_blog_archive (source_table, source_id, payload)
			SELECT 't_comment', c.id, to_jsonb(c) FROM t_comment c
			WHERE c.type IN (2, 3, 4) AND NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`INSERT INTO t_legacy_blog_archive (source_table, source_id, payload)
			SELECT 't_photo_album', a.id, to_jsonb(a) FROM t_photo_album a
			WHERE NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`INSERT INTO t_legacy_blog_archive (source_table, source_id, payload)
			SELECT 't_photo', p.id, to_jsonb(p) FROM t_photo p
			JOIN t_photo_album a ON a.id = p.album_id
			WHERE NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`UPDATE t_user_info u SET
			about = CASE WHEN trim(COALESCE(u.about, '')) = '' THEN COALESCE((
				SELECT CASE WHEN left(ltrim(a.content), 1) = '{' THEN COALESCE(a.content::jsonb->>'content', '') ELSE a.content END
				FROM t_about a ORDER BY a.id LIMIT 1
			), '') ELSE u.about END,
			intro = CASE WHEN trim(COALESCE(u.intro, '')) = '' THEN COALESCE(wc.config::jsonb->>'authorIntro', '') ELSE u.intro END,
			avatar = CASE WHEN trim(COALESCE(u.avatar, '')) = '' OR u.avatar = COALESCE(wc.config::jsonb->>'userAvatar', '')
				THEN COALESCE(NULLIF(wc.config::jsonb->>'authorAvatar', ''), u.avatar) ELSE u.avatar END,
			profile_links_json = (
				SELECT COALESCE(jsonb_agg(link_value), '[]'::jsonb) FROM (
					SELECT existing_links.link_value FROM jsonb_array_elements(COALESCE(u.profile_links_json, '[]'::jsonb)) AS existing_links(link_value)
					UNION ALL
					SELECT jsonb_build_object('label', social.label, 'url', social.url)
					FROM (VALUES
						('GitHub', wc.config::jsonb->>'github'), ('Gitee', wc.config::jsonb->>'gitee'),
						('Weibo', wc.config::jsonb->>'weibo'), ('CSDN', wc.config::jsonb->>'csdn'),
						('Zhihu', wc.config::jsonb->>'zhihu'), ('Juejin', wc.config::jsonb->>'juejin'),
						('Twitter', wc.config::jsonb->>'twitter'), ('Stack Overflow', wc.config::jsonb->>'stackoverflow')
					) AS social(label, url)
					WHERE social.url ~* '^https?://'
					UNION ALL
					SELECT jsonb_build_object('label', f.link_name, 'url', f.link_address, 'description', f.link_intro)
					FROM t_friend_link f WHERE f.status = 1
				) profile_links
			)
		FROM t_website_config wc WHERE wc.id = 1 AND u.id = (SELECT user_id FROM migration_legacy_site_owner)`,
		`UPDATE t_comment SET type = 7, topic_id = (SELECT user_id FROM migration_legacy_site_owner)
			WHERE type IN (2, 3, 4) AND EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`UPDATE t_comment SET is_delete = 1, is_review = 0, topic_id = 0
			WHERE type IN (2, 3, 4) AND NOT EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`UPDATE t_photo_album SET user_id = (SELECT user_id FROM migration_legacy_site_owner)
			WHERE user_id IS NULL AND EXISTS (SELECT 1 FROM migration_legacy_site_owner)`,
		`DELETE FROM t_photo WHERE album_id IN (SELECT id FROM t_photo_album WHERE user_id IS NULL)`,
		`DELETE FROM t_photo_album WHERE user_id IS NULL`,
		`ALTER TABLE t_photo_album ALTER COLUMN user_id SET NOT NULL`,
		`DELETE FROM t_friend_link`,
		`DELETE FROM t_about`,
		`DELETE FROM t_role_menu WHERE menu_id IN (SELECT id FROM t_menu WHERE path IN ('/album-submenu', '/albums', '/albums/:albumId', '/photos/delete', '/links', '/about'))`,
		`DELETE FROM t_menu WHERE path IN ('/album-submenu', '/albums', '/albums/:albumId', '/photos/delete', '/links', '/about')`,
		`UPDATE t_website_config SET config = (config::jsonb - ARRAY[
			'author', 'authorAvatar', 'authorIntro', 'github', 'gitee', 'qq', 'weChat', 'weibo',
			'csdn', 'zhihu', 'juejin', 'twitter', 'stackoverflow'
		]::text[])::text WHERE id = 1`,
		`CREATE INDEX IF NOT EXISTS idx_photo_album_user_id ON t_photo_album(user_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_comment_profile_wall ON t_comment(topic_id, id DESC) WHERE type = 7`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply per-user legacy-content migration: %w", err)
		}
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 34, "per-user-legacy-content"); err != nil {
		return fmt.Errorf("record per-user legacy-content migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit per-user legacy-content migration: %w", err)
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
	websiteConfig := entity.WebsiteConfig{
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
	for _, table := range []string{"t_website_config"} {
		if _, err := session.Exec("SELECT setval(pg_get_serial_sequence('" + table + "', 'id'), GREATEST((SELECT MAX(id) FROM " + table + "), 1), true)"); err != nil {
			return fmt.Errorf("advance %s id sequence: %w", table, err)
		}
	}
	return nil
}
