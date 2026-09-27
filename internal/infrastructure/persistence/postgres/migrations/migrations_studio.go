package migrations

import (
	"context"
	"fmt"

	"xorm.io/xorm"
)

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

// applyStudioActivationReminderSchema adds the dedicated creator-progress
// preference, arbitrary system-notification copy, and the hourly reminder job.
func applyStudioActivationReminderSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 30)").Get(&applied); err != nil {
		return fmt.Errorf("check studio activation reminder migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin studio activation reminder migration: %w", err)
	}
	defer session.Rollback()
	statements := []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS notify_studio_activation SMALLINT NOT NULL DEFAULT 1`,
		`ALTER TABLE t_user_notification ADD COLUMN IF NOT EXISTS title VARCHAR(120) NOT NULL DEFAULT ''`,
		`ALTER TABLE t_user_notification ADD COLUMN IF NOT EXISTS excerpt VARCHAR(240) NOT NULL DEFAULT ''`,
		`ALTER TABLE t_user_notification ADD COLUMN IF NOT EXISTS action_url VARCHAR(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE t_user_notification ALTER COLUMN type TYPE VARCHAR(32)`,
		`ALTER TABLE t_user_notification ALTER COLUMN actor_id DROP NOT NULL`,
	}
	for _, statement := range statements {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply studio activation reminder schema: %w", err)
		}
	}
	if _, err := session.Exec(`
		INSERT INTO t_job (job_name, job_group, invoke_target, cron_expression, misfire_policy, concurrent, status, remark, create_time, update_time)
		SELECT ?, ?, ?, ?, 3, 0, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		WHERE NOT EXISTS (SELECT 1 FROM t_job WHERE invoke_target = ?)
	`, "提醒创作者完成激活", "系统", "studio.activationReminders", "15 * * * *", "每小时提醒停滞的新创作者完成激活", "studio.activationReminders"); err != nil {
		return fmt.Errorf("seed studio activation reminder job: %w", err)
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 30, "studio-activation-reminders"); err != nil {
		return fmt.Errorf("record studio activation reminder migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit studio activation reminder migration: %w", err)
	}
	return nil
}

// applyArticleSearchReconcileJob registers the recurring search repair job.
// The index itself lives in Meilisearch, so this migration only owns the
// durable scheduler entry.
func applyArticleSearchReconcileJob(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 31)").Get(&applied); err != nil {
		return fmt.Errorf("check article search reconcile migration: %w", err)
	}
	if applied {
		return nil
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin article search reconcile migration: %w", err)
	}
	defer session.Rollback()
	if _, err := session.Exec(`
		INSERT INTO t_job (job_name, job_group, invoke_target, cron_expression, misfire_policy, concurrent, status, remark, create_time, update_time)
		SELECT ?, ?, ?, ?, 3, 0, 1, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		WHERE NOT EXISTS (SELECT 1 FROM t_job WHERE invoke_target = ?)
	`, "对账文章搜索索引", "系统", "article.searchReconcile", "*/5 * * * *", "每 5 分钟修复 Meilisearch 中的缺失和陈旧文章文档", "article.searchReconcile"); err != nil {
		return fmt.Errorf("seed article search reconcile job: %w", err)
	}
	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 31, "article-search-reconcile"); err != nil {
		return fmt.Errorf("record article search reconcile migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit article search reconcile migration: %w", err)
	}
	return nil
}
