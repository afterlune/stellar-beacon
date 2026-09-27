package migrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"xorm.io/xorm"
)

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
		"article.publishScheduled":   true,
		"growth.cleanup":             true,
		"jobLogs.cleanup":            true,
		"userArea.refresh":           true,
		"studio.activationReminders": true,
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
		{"提醒创作者完成激活", "系统", "studio.activationReminders", "15 * * * *", "每小时提醒停滞的新创作者完成激活"},
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
