package migrations

import (
	"context"
	"fmt"

	"xorm.io/xorm"
)

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
