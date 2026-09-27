package migrations

import (
	"context"
	"fmt"

	"xorm.io/xorm"
)

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
