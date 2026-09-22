//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"xorm.io/xorm"
)

func TestEphemeralPostgresAndRedis(t *testing.T) {
	if os.Getenv("TESTCONTAINERS_ENABLED") != "1" {
		t.Skip("set TESTCONTAINERS_ENABLED=1 to run isolated container tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16.15-alpine3.23",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "integration",
				"POSTGRES_PASSWORD": "integration",
				"POSTGRES_DB":       "integration",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start isolated postgres: %v", err)
	}
	t.Cleanup(func() { _ = postgres.Terminate(context.Background()) })

	pgHost, err := postgres.Host(ctx)
	if err != nil {
		t.Fatalf("resolve postgres host: %v", err)
	}
	pgPort, err := postgres.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("resolve postgres port: %v", err)
	}
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://integration:integration@%s:%s/integration?sslmode=disable", pgHost, pgPort.Port()))
	if err != nil {
		t.Fatalf("open isolated postgres: %v", err)
	}
	defer db.Close()
	if err := pingDatabase(ctx, db); err != nil {
		t.Fatalf("ping isolated postgres: %v", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE integration_probe (id INTEGER PRIMARY KEY, value TEXT NOT NULL)"); err != nil {
		t.Fatalf("create postgres probe: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO integration_probe (id, value) VALUES ($1, $2)", 1, "ok"); err != nil {
		t.Fatalf("insert postgres probe: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
CREATE TABLE t_user_info (
    id INTEGER PRIMARY KEY,
    handle VARCHAR(40) NOT NULL UNIQUE,
    email VARCHAR(50), nickname VARCHAR(50) NOT NULL, avatar VARCHAR(1024) NOT NULL,
    intro VARCHAR(255), website VARCHAR(255), is_subscribe SMALLINT DEFAULT 0,
    notify_comment SMALLINT NOT NULL DEFAULT 1, notify_interaction SMALLINT NOT NULL DEFAULT 1, notify_topic SMALLINT NOT NULL DEFAULT 1,
    is_disable SMALLINT DEFAULT 0, create_time TIMESTAMP, update_time TIMESTAMP
);
CREATE TABLE t_user_auth (
    id INTEGER PRIMARY KEY, user_info_id INTEGER NOT NULL, username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL, login_type SMALLINT NOT NULL, ip_address VARCHAR(50),
    ip_source VARCHAR(50), create_time TIMESTAMP, update_time TIMESTAMP, last_login_time TIMESTAMP
);
CREATE TABLE t_role (
    id INTEGER PRIMARY KEY, role_name VARCHAR(20) NOT NULL, is_disable SMALLINT DEFAULT 0,
    create_time TIMESTAMP, update_time TIMESTAMP
);
CREATE TABLE t_user_role (id INTEGER PRIMARY KEY, user_id INTEGER, role_id INTEGER);
CREATE TABLE t_category (id INTEGER PRIMARY KEY, category_name VARCHAR(50) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_tag (id INTEGER PRIMARY KEY, tag_name VARCHAR(50) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_talk (id SERIAL PRIMARY KEY, user_id INTEGER NOT NULL, content TEXT NOT NULL, images TEXT, is_top SMALLINT NOT NULL, status SMALLINT NOT NULL, moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible', moderation_reason VARCHAR(255), moderated_by INTEGER DEFAULT 0, moderated_at TIMESTAMP, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_comment (id SERIAL PRIMARY KEY, user_id INTEGER NOT NULL, topic_id INTEGER, comment_content TEXT NOT NULL, reply_user_id INTEGER, parent_id INTEGER, type SMALLINT NOT NULL, is_delete SMALLINT NOT NULL, is_review SMALLINT NOT NULL, notification_dispatched_at TIMESTAMPTZ NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_unique_view (id INTEGER PRIMARY KEY, views_count INTEGER NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_website_config (id INTEGER PRIMARY KEY, config TEXT, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_about (id INTEGER PRIMARY KEY, content TEXT, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_friend_link (id INTEGER PRIMARY KEY, link_name VARCHAR(50) NOT NULL, link_avatar VARCHAR(255) NOT NULL, link_address VARCHAR(255) NOT NULL, link_intro VARCHAR(255) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_job (id INTEGER PRIMARY KEY, job_name VARCHAR(64) NOT NULL, job_group VARCHAR(64) NOT NULL, invoke_target VARCHAR(500) NOT NULL, cron_expression VARCHAR(255), misfire_policy SMALLINT, concurrent SMALLINT, status SMALLINT, create_time TIMESTAMP, update_time TIMESTAMP, remark VARCHAR(500));
CREATE TABLE t_job_log (id INTEGER PRIMARY KEY, job_id INTEGER NOT NULL, job_name VARCHAR(64) NOT NULL, job_group VARCHAR(64) NOT NULL, invoke_target VARCHAR(500) NOT NULL, job_message VARCHAR(500), status SMALLINT, exception_info VARCHAR(2000), create_time TIMESTAMP, start_time TIMESTAMP, end_time TIMESTAMP);
CREATE TABLE t_exception_log (id INTEGER PRIMARY KEY, opt_uri VARCHAR(255) NOT NULL, opt_method VARCHAR(255) NOT NULL, request_method VARCHAR(255), request_param TEXT, opt_desc VARCHAR(255), exception_info TEXT, ip_address VARCHAR(255), ip_source VARCHAR(255), create_time TIMESTAMP);
CREATE TABLE t_operation_log (id INTEGER PRIMARY KEY, opt_module VARCHAR(50) NOT NULL, opt_type VARCHAR(50) NOT NULL, opt_uri VARCHAR(255) NOT NULL, opt_method VARCHAR(255) NOT NULL, opt_desc VARCHAR(255) NOT NULL, request_param TEXT NOT NULL, request_method VARCHAR(20) NOT NULL, response_data TEXT NOT NULL, user_id INTEGER NOT NULL, nickname VARCHAR(50) NOT NULL, ip_address VARCHAR(255) NOT NULL, ip_source VARCHAR(255) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_menu (id INTEGER PRIMARY KEY, name VARCHAR(50) NOT NULL, path VARCHAR(100) NOT NULL, component VARCHAR(100) NOT NULL, icon VARCHAR(50) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP, order_num SMALLINT NOT NULL, parent_id INTEGER, is_hidden SMALLINT NOT NULL);
CREATE TABLE t_resource (id INTEGER PRIMARY KEY, resource_name VARCHAR(50) NOT NULL, url VARCHAR(255), request_method VARCHAR(10), parent_id INTEGER, is_anonymous SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_role_menu (id INTEGER PRIMARY KEY, role_id INTEGER, menu_id INTEGER);
CREATE TABLE t_role_resource (id INTEGER PRIMARY KEY, role_id INTEGER, resource_id INTEGER);
CREATE TABLE t_photo_album (id INTEGER PRIMARY KEY, album_name VARCHAR(50) NOT NULL, album_desc VARCHAR(100) NOT NULL, album_cover VARCHAR(255) NOT NULL, is_delete SMALLINT NOT NULL, status SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_photo (id INTEGER PRIMARY KEY, album_id INTEGER NOT NULL, photo_name VARCHAR(50) NOT NULL, photo_desc VARCHAR(100), photo_src VARCHAR(255) NOT NULL, is_delete SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_series (
    id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL DEFAULT 0, series_name VARCHAR(50) NOT NULL, series_desc VARCHAR(255), cover VARCHAR(1024),
    is_delete SMALLINT NOT NULL DEFAULT 0, status SMALLINT NOT NULL DEFAULT 1,
    moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible', create_time TIMESTAMP, update_time TIMESTAMP
);CREATE TABLE t_article (
    id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, category_id INTEGER, article_cover VARCHAR(1024),
    article_title VARCHAR(50) NOT NULL, article_content TEXT NOT NULL, article_content_html TEXT,
    series_id INTEGER, series_order INTEGER,
    is_top SMALLINT NOT NULL,
    is_featured SMALLINT NOT NULL, is_delete SMALLINT NOT NULL, status SMALLINT NOT NULL,
    scheduled_at TIMESTAMP, moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible', moderation_reason VARCHAR(255), moderated_by INTEGER DEFAULT 0, moderated_at TIMESTAMP,
    type SMALLINT NOT NULL, password VARCHAR(255), original_url VARCHAR(255),
    create_time TIMESTAMP, update_time TIMESTAMP
);
CREATE TABLE t_article_tag (id INTEGER PRIMARY KEY, article_id INTEGER NOT NULL, tag_id INTEGER NOT NULL);
CREATE TABLE t_article_continuation_target (
    id SERIAL PRIMARY KEY, source_article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
    target_type VARCHAR(16) NOT NULL, target_id INTEGER NOT NULL, placement VARCHAR(24) NOT NULL,
    position SMALLINT NOT NULL DEFAULT 0, metric_date DATE NOT NULL, clicks BIGINT NOT NULL DEFAULT 0,
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_article_id, target_type, target_id, placement, position, metric_date)
);
CREATE TABLE t_article_daily_metric (
    id SERIAL PRIMARY KEY, article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL, views BIGINT NOT NULL DEFAULT 0, unique_readers BIGINT NOT NULL DEFAULT 0,
    effective_sessions BIGINT NOT NULL DEFAULT 0, total_active_ms BIGINT NOT NULL DEFAULT 0,
    completed_sessions BIGINT NOT NULL DEFAULT 0, series_impressions BIGINT NOT NULL DEFAULT 0, series_clicks BIGINT NOT NULL DEFAULT 0, related_impressions BIGINT NOT NULL DEFAULT 0, related_clicks BIGINT NOT NULL DEFAULT 0, create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE (article_id, metric_date)
);
CREATE TABLE t_article_publish_record (
    id BIGSERIAL PRIMARY KEY, article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL, scheduled_at TIMESTAMPTZ NOT NULL, published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notification_state VARCHAR(20) NOT NULL DEFAULT 'pending', notification_attempts INTEGER NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NULL, last_error TEXT NOT NULL DEFAULT '',
    create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (article_id, scheduled_at)
);
CREATE TABLE t_content_operation_audit (
    id BIGSERIAL PRIMARY KEY, operator_id INTEGER NOT NULL, operator_nickname VARCHAR(64) NOT NULL DEFAULT '',
    content_type VARCHAR(16) NOT NULL, operation VARCHAR(32) NOT NULL, target_mode VARCHAR(16) NOT NULL,
    filter_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb, snapshot_max_id INTEGER NOT NULL DEFAULT 0,
    requested_count INTEGER NOT NULL DEFAULT 0, affected_count INTEGER NOT NULL DEFAULT 0,
    result VARCHAR(16) NOT NULL, error_message TEXT NOT NULL DEFAULT '', ip_address VARCHAR(255) NOT NULL DEFAULT '',
    ip_source VARCHAR(255) NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE t_content_operation_audit_item (
    id BIGSERIAL PRIMARY KEY, audit_id BIGINT NOT NULL REFERENCES t_content_operation_audit(id) ON DELETE CASCADE,
    content_id INTEGER NOT NULL, title VARCHAR(255) NOT NULL DEFAULT '', previous_status SMALLINT NOT NULL,
    next_status SMALLINT NOT NULL, result VARCHAR(16) NOT NULL
);
CREATE TABLE t_user_follow (
    id BIGSERIAL PRIMARY KEY, follower_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    start_event_id BIGINT NOT NULL DEFAULT 0, last_read_event_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (follower_id, author_id), CHECK (follower_id <> author_id)
);
CREATE TABLE t_author_publish_event (
    id BIGSERIAL PRIMARY KEY, author_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    content_type VARCHAR(16) NOT NULL, content_id BIGINT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (content_type, content_id)
);
CREATE TABLE t_user_notification (
    id BIGSERIAL PRIMARY KEY, recipient_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    actor_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    type VARCHAR(16) NOT NULL, content_type VARCHAR(16) NOT NULL, content_id BIGINT NOT NULL,
    comment_id BIGINT NOT NULL DEFAULT 0, dedupe_key VARCHAR(191) NOT NULL,
    read_at TIMESTAMPTZ NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (recipient_id, dedupe_key), CHECK (recipient_id <> actor_id)
);
CREATE TABLE t_article_reaction (
    id SERIAL PRIMARY KEY, article_id INTEGER NOT NULL, user_info_id INTEGER NOT NULL,
    reaction VARCHAR(16) NOT NULL, create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (article_id, user_info_id, reaction)
);
CREATE TABLE t_topic_subscription (
    id BIGSERIAL PRIMARY KEY, user_id INTEGER NOT NULL, topic_type VARCHAR(16) NOT NULL,
    topic_key VARCHAR(64) NOT NULL, topic_name VARCHAR(50) NOT NULL, muted SMALLINT NOT NULL DEFAULT 0,
    start_event_id BIGINT NOT NULL DEFAULT 0, last_read_event_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, topic_type, topic_key)
);
CREATE TABLE t_recommendation_feedback (
    id BIGSERIAL PRIMARY KEY, user_id INTEGER NOT NULL, target_type VARCHAR(16) NOT NULL,
    target_key VARCHAR(160) NOT NULL, article_id INTEGER NULL, author_id INTEGER NULL,
    topic_type VARCHAR(16) NULL, topic_key VARCHAR(64) NULL, target_label VARCHAR(255) NOT NULL,
    create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, target_type, target_key)
);
CREATE TABLE t_collection (
    id BIGSERIAL PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES t_user_info(id) ON DELETE CASCADE,
    slug VARCHAR(80) NOT NULL UNIQUE, title VARCHAR(80) NOT NULL, description VARCHAR(500) NOT NULL DEFAULT '',
    visibility VARCHAR(16) NOT NULL DEFAULT 'private', moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible',
    moderation_reason VARCHAR(255) NOT NULL DEFAULT '', moderated_by INTEGER NOT NULL DEFAULT 0, moderated_at TIMESTAMPTZ NULL,
    is_delete SMALLINT NOT NULL DEFAULT 0, create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE t_collection_item (
    id BIGSERIAL PRIMARY KEY, collection_id BIGINT NOT NULL REFERENCES t_collection(id) ON DELETE CASCADE,
    article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE, note VARCHAR(280) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0, create_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE (collection_id, article_id)
);`); err != nil {
		t.Fatalf("create repository fixtures: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO t_user_info (id, handle, email, nickname, avatar, is_subscribe, is_disable) VALUES (1, 'integration', 'integration@example.com', 'integration', '', 0, 0), (2, 'reader', 'reader@example.com', 'reader', '', 0, 0);
INSERT INTO t_user_auth (id, user_info_id, username, password, login_type) VALUES (1, 1, 'integration@example.com', 'hashed-password', 1);
INSERT INTO t_role (id, role_name, is_disable) VALUES (1, 'user', 0);
INSERT INTO t_user_role (id, user_id, role_id) VALUES (1, 1, 1);
INSERT INTO t_category (id, category_name) VALUES (1, 'integration category'), (2, 'second category');
INSERT INTO t_tag (id, tag_name) VALUES (1, 'integration tag'), (2, 'second tag'), (3, 'third tag'), (4, 'fourth tag');
INSERT INTO t_talk (id, user_id, content, is_top, status) VALUES (1, 1, 'integration talk', 0, 1);
INSERT INTO t_website_config (id, config) VALUES (1, '{"name":"integration"}');
INSERT INTO t_about (id, content) VALUES (1, '{"content":"integration"}');
INSERT INTO t_menu (id, name, path, component, icon, order_num, parent_id, is_hidden) VALUES (1, 'integration menu', '/', 'Layout', '', 1, 0, 0);
INSERT INTO t_resource (id, resource_name, url, request_method, parent_id, is_anonymous) VALUES (1, 'integration resource', '/integration', 'GET', 0, 0);
INSERT INTO t_role_menu (id, role_id, menu_id) VALUES (1, 1, 1);
INSERT INTO t_role_resource (id, role_id, resource_id) VALUES (1, 1, 1);
INSERT INTO t_photo_album (id, album_name, album_desc, album_cover, is_delete, status) VALUES (1, 'integration album', 'integration', '', 0, 1);
INSERT INTO t_photo (id, album_id, photo_name, photo_src, is_delete) VALUES (1, 1, 'integration photo', 'https://example.com/photo.jpg', 0);
INSERT INTO t_series (id, user_id, series_name, is_delete, status) VALUES (10, 1, 'integration series', 0, 1);INSERT INTO t_article (id, user_id, category_id, article_title, article_content, series_id, series_order, is_top, is_featured, is_delete, status, type) VALUES (1, 1, 1, 'integration article', 'content', 10, 1, 0, 0, 0, 1, 1);
INSERT INTO t_article (id, user_id, category_id, article_title, article_content, series_id, series_order, scheduled_at, is_top, is_featured, is_delete, status, type) VALUES (7, 1, 1, 'scheduled integration article', 'content', NULL, 0, CURRENT_TIMESTAMP - INTERVAL '1 minute', 0, 0, 0, 4, 1);
INSERT INTO t_article (id, user_id, category_id, article_title, article_content, series_id, series_order, is_top, is_featured, is_delete, status, type) VALUES
    (2, 1, 1, 'same series article', 'content', 10, 2, 0, 0, 0, 1, 1),
    (3, 1, 2, 'shared tag article', 'content', NULL, 0, 0, 0, 0, 1, 1),
    (4, 1, 1, 'same category article', 'content', NULL, 0, 0, 0, 0, 1, 1),
    (5, 1, 1, 'same series article two', 'content', 10, 3, 0, 0, 0, 1, 1),
    (6, 1, 2, 'recent fallback article', 'content', NULL, 0, 0, 0, 0, 1, 1);
INSERT INTO t_article_tag (id, article_id, tag_id) VALUES
    (1, 1, 1),
    (2, 2, 1), (3, 2, 2),
    (4, 3, 1),
    (5, 4, 3),
    (6, 5, 1),
    (7, 6, 4);
INSERT INTO t_article_daily_metric (article_id, metric_date, views, unique_readers, effective_sessions, total_active_ms, completed_sessions) VALUES (1, CURRENT_DATE, 2, 1, 1, 5000, 1);
SELECT setval(pg_get_serial_sequence('t_talk', 'id'), (SELECT MAX(id) FROM t_talk));`); err != nil {
		t.Fatalf("insert repository fixtures: %v", err)
	}

	xormEngine, err := xorm.NewEngine("postgres", fmt.Sprintf("postgres://integration:integration@%s:%s/integration?sslmode=disable", pgHost, pgPort.Port()))
	if err != nil {
		t.Fatalf("open xorm repository engine: %v", err)
	}
	defer xormEngine.Close()
	authUser, err := NewUserAuthRepo(xormEngine).FindByUsername(ctx, "integration@example.com")
	if err != nil {
		t.Fatalf("read auth repository fixture: %v", err)
	}
	if authUser.Info.Nickname != "integration" || len(authUser.Roles) != 1 || authUser.Roles[0] != "user" {
		t.Fatalf("unexpected auth repository result: %+v", authUser)
	}
	articles, count, err := NewArticleRepo(xormEngine).ListArchives(ctx, 1, 10)
	if err != nil {
		t.Fatalf("read article repository fixture: %v", err)
	}
	if count != 6 || len(articles) != 6 {
		t.Fatalf("unexpected article repository result: count=%d articles=%+v", count, articles)
	}
	topFeatured, err := NewArticleRepo(xormEngine).ListTopAndFeaturedArticles(ctx)
	if err != nil || len(topFeatured) == 0 {
		t.Fatalf("unexpected top and featured article result: articles=%+v err=%v", topFeatured, err)
	}
	site := NewSiteInfoRepo(xormEngine)
	articleCount, err := site.CountArticles(ctx)
	if err != nil || articleCount != 7 {
		t.Fatalf("unexpected site article count: count=%d err=%v", articleCount, err)
	}
	rankedArticles, err := site.ListArticleRank(ctx, []int{1})
	if err != nil || len(rankedArticles) != 1 || rankedArticles[0].ArticleTitle != "integration article" {
		t.Fatalf("unexpected article rank result: articles=%v err=%v", rankedArticles, err)
	}
	relatedArticles, err := NewArticleRepo(xormEngine).ListRelatedArticles(ctx, 1, 1, 10, 3)
	if err != nil || len(relatedArticles) != 3 {
		t.Fatalf("unexpected related article result: articles=%v err=%v", relatedArticles, err)
	}
	relatedIDs := []int{relatedArticles[0].Id, relatedArticles[1].Id, relatedArticles[2].Id}
	if relatedIDs[0] != 3 || relatedIDs[1] != 4 || relatedIDs[2] != 6 {
		t.Fatalf("unexpected related article order: %v", relatedIDs)
	}
	for _, article := range relatedArticles {
		if article.Id == 1 || article.Id == 2 || article.Id == 5 {
			t.Fatalf("current or same-series article leaked into related results: %v", relatedIDs)
		}
	}
	relatedWithoutSeries, err := NewArticleRepo(xormEngine).ListRelatedArticles(ctx, 3, 2, 0, 4)
	relatedWithoutSeriesIDs := make([]int, 0, len(relatedWithoutSeries))
	for _, article := range relatedWithoutSeries {
		relatedWithoutSeriesIDs = append(relatedWithoutSeriesIDs, article.Id)
	}
	if err != nil || len(relatedWithoutSeries) != 4 {
		t.Fatalf("articles without a series should still receive recommendations: ids=%v err=%v", relatedWithoutSeriesIDs, err)
	}
	hasNonSeriesRecommendation := false
	for _, article := range relatedWithoutSeries {
		if article.Id == 6 {
			hasNonSeriesRecommendation = true
		}
	}
	if !hasNonSeriesRecommendation {
		t.Fatalf("non-series article missing from recommendations: %v", relatedWithoutSeries)
	}
	contentAnalytics := NewContentAnalyticsRepo(xormEngine)
	if err := contentAnalytics.RecordView(ctx, 1, time.Now()); err != nil {
		t.Fatalf("record content analytics view: %v", err)
	}
	if err := contentAnalytics.RecordReadSession(ctx, 1, time.Now(), 4200, 95, 1); err != nil {
		t.Fatalf("record content analytics session: %v", err)
	}
	if err := contentAnalytics.RecordReadSession(ctx, 1, time.Now(), 2000, 50, 1); err != nil {
		t.Fatalf("record non-effective content analytics session: %v", err)
	}
	for _, eventType := range []port.ContinuationEventType{
		port.ContinuationEventSeriesImpression,
		port.ContinuationEventRelatedImpression,
		port.ContinuationEventRelatedImpression,
	} {
		if err := contentAnalytics.RecordContinuationEvent(ctx, 1, time.Now(), eventType, nil); err != nil {
			t.Fatalf("record continuation event %s: %v", eventType, err)
		}
	}
	dailyMetrics, err := contentAnalytics.GetArticleDailyMetrics(ctx, 1, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(dailyMetrics) != 1 || dailyMetrics[0].Views != 3 || dailyMetrics[0].CompletedSessions != 2 ||
		dailyMetrics[0].SeriesImpressions != 1 || dailyMetrics[0].SeriesClicks != 0 ||
		dailyMetrics[0].RelatedImpressions != 2 || dailyMetrics[0].RelatedClicks != 0 {
		t.Fatalf("unexpected content analytics daily metrics: metrics=%v err=%v", dailyMetrics, err)
	}
	siteDailyMetrics, err := contentAnalytics.ListDailyMetrics(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(siteDailyMetrics) != 1 || siteDailyMetrics[0].EffectiveSessions != 2 ||
		siteDailyMetrics[0].SeriesImpressions != 1 || siteDailyMetrics[0].RelatedImpressions != 2 {
		t.Fatalf("unexpected site content analytics metrics: metrics=%v err=%v", siteDailyMetrics, err)
	}
	articleMetrics, err := contentAnalytics.ListArticleMetrics(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(articleMetrics) != 1 || articleMetrics[0].ArticleId != 1 || articleMetrics[0].CompletedSessions != 2 ||
		articleMetrics[0].SeriesClicks != 0 || articleMetrics[0].RelatedClicks != 0 {
		t.Fatalf("unexpected article content analytics metrics: metrics=%v err=%v", articleMetrics, err)
	}
	platformRepo := NewPlatformRepo(xormEngine)
	followRepo := NewFollowRepo(xormEngine)
	if err := followRepo.Follow(ctx, 2, 1); err != nil {
		t.Fatalf("follow author: %v", err)
	}
	if err := followRepo.Follow(ctx, 2, 1); err != nil {
		t.Fatalf("repeat follow must be idempotent: %v", err)
	}
	if err := followRepo.Follow(ctx, 1, 1); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("self follow must fail, got %v", err)
	}
	authorCard, err := platformRepo.GetAuthorByHandle(ctx, "integration", 2)
	if err != nil || authorCard.FollowerCount != 1 || !authorCard.IsFollowing {
		t.Fatalf("unexpected followed author card: author=%+v err=%v", authorCard, err)
	}
	published, err := NewArticleRepo(xormEngine).PublishDueScheduledArticles(ctx, time.Now(), 10)
	if err != nil || len(published) != 1 || published[0].ArticleID != 7 || published[0].NotificationState != port.ScheduledNotificationPending {
		t.Fatalf("unexpected scheduled publication result: published=%+v err=%v", published, err)
	}
	if err := NewArticleRepo(xormEngine).MarkScheduledNotificationFailed(ctx, published[0].RecordID, "smtp unavailable", nil); err != nil {
		t.Fatalf("mark scheduled notification failed: %v", err)
	}
	feed, feedCount, err := followRepo.ListFollowFeed(ctx, 2, "", 1, 10)
	if err != nil || feedCount != 1 || len(feed) != 1 || feed[0].ContentId != 7 || feed[0].ContentType != port.FollowContentArticle {
		t.Fatalf("unexpected follow feed: feed=%+v count=%d err=%v", feed, feedCount, err)
	}
	if err := platformRepo.ModerateContent(ctx, "article", 7, 1, true, "integration hidden"); err != nil {
		t.Fatalf("hide followed article: %v", err)
	}
	hiddenUnread, err := followRepo.UnreadNotificationCount(ctx, 2)
	if err != nil || hiddenUnread != 0 {
		t.Fatalf("hidden content must not remain unread: count=%d err=%v", hiddenUnread, err)
	}
	if err := platformRepo.ModerateContent(ctx, "article", 7, 1, false, ""); err != nil {
		t.Fatalf("restore followed article: %v", err)
	}
	notifications, err := followRepo.ListNotifications(ctx, 2, port.NotificationGroupAll, 1, 10)
	if err != nil || notifications.Count != 1 || notifications.UnreadCount != 1 || len(notifications.Records) != 1 {
		t.Fatalf("unexpected follow notifications: notifications=%+v err=%v", notifications, err)
	}
	if err := followRepo.MarkNotificationsRead(ctx, 2, notifications.ReadCursor); err != nil {
		t.Fatalf("mark follow notifications read: %v", err)
	}
	unread, err := followRepo.UnreadNotificationCount(ctx, 2)
	if err != nil || unread != 0 {
		t.Fatalf("unexpected unread notification count: count=%d err=%v", unread, err)
	}
	if _, err := platformRepo.SaveOwnedTalk(ctx, entity.TTalk{UserId: 1, Content: "followed talk", Status: 1, ModerationStatus: "visible"}); err != nil {
		t.Fatalf("publish followed talk: %v", err)
	}
	notifications, err = followRepo.ListNotifications(ctx, 2, port.NotificationGroupAll, 1, 10)
	if err != nil || notifications.Count != 2 || notifications.UnreadCount != 1 {
		t.Fatalf("unexpected talk follow notification: notifications=%+v err=%v", notifications, err)
	}
	following, followingCount, err := followRepo.ListFollowing(ctx, 2, 1, 10)
	if err != nil || followingCount != 1 || len(following) != 1 || following[0].Id != 1 {
		t.Fatalf("unexpected following list: users=%+v count=%d err=%v", following, followingCount, err)
	}
	followers, followerCount, err := followRepo.ListFollowers(ctx, 1, 1, 10)
	if err != nil || followerCount != 1 || len(followers) != 1 || followers[0].Id != 2 {
		t.Fatalf("unexpected follower list: users=%+v count=%d err=%v", followers, followerCount, err)
	}
	commentRepo := NewCommentRepo(xormEngine)
	commentID, err := commentRepo.Create(ctx, entity.TComment{UserId: 2, TopicId: 1, CommentContent: "new comment", Type: 1, IsReview: 1})
	if err != nil || commentID <= 0 {
		t.Fatalf("create approved comment: id=%d err=%v", commentID, err)
	}
	commentNotifications, err := followRepo.ListNotifications(ctx, 1, port.NotificationGroupComment, 1, 10)
	if err != nil || commentNotifications.Count != 1 || commentNotifications.UnreadCount != 1 || len(commentNotifications.Records) != 1 ||
		commentNotifications.Records[0].Type != port.NotificationTypeComment || commentNotifications.Records[0].CommentId != commentID {
		t.Fatalf("unexpected comment notifications: notifications=%+v err=%v", commentNotifications, err)
	}
	if err := commentRepo.Review(ctx, []int{commentID}, 0); err != nil {
		t.Fatalf("hide approved comment notification: %v", err)
	}
	hiddenComments, err := followRepo.ListNotifications(ctx, 1, port.NotificationGroupComment, 1, 10)
	if err != nil || hiddenComments.Count != 0 || hiddenComments.UnreadCount != 0 {
		t.Fatalf("unapproved comment notification must be hidden: notifications=%+v err=%v", hiddenComments, err)
	}
	if err := commentRepo.Review(ctx, []int{commentID}, 1); err != nil {
		t.Fatalf("restore approved comment notification: %v", err)
	}
	restoredComments, err := followRepo.ListNotifications(ctx, 1, port.NotificationGroupComment, 1, 10)
	if err != nil || restoredComments.Count != 1 || restoredComments.UnreadCount != 1 {
		t.Fatalf("restored comment notification must reappear: notifications=%+v err=%v", restoredComments, err)
	}
	secondCommentID, err := commentRepo.Create(ctx, entity.TComment{UserId: 2, TopicId: 1, CommentContent: "second comment", Type: 1, IsReview: 1})
	if err != nil || secondCommentID <= 0 {
		t.Fatalf("create second approved comment: id=%d err=%v", secondCommentID, err)
	}
	secondPage, err := commentRepo.ResolveCommentPage(ctx, 1, 1, secondCommentID, 1)
	if err != nil || secondPage != 1 {
		t.Fatalf("resolve newest comment page: page=%d err=%v", secondPage, err)
	}
	firstPage, err := commentRepo.ResolveCommentPage(ctx, 1, 1, commentID, 1)
	if err != nil || firstPage != 2 {
		t.Fatalf("resolve older comment page: page=%d err=%v", firstPage, err)
	}
	reactionRepo := NewArticleReactionRepo(xormEngine)
	if active, err := reactionRepo.Toggle(ctx, 1, 2, port.ReactionLike); err != nil || !active {
		t.Fatalf("activate article reaction: active=%v err=%v", active, err)
	}
	reactionNotifications, err := followRepo.ListNotifications(ctx, 1, port.NotificationGroupReaction, 1, 10)
	if err != nil || reactionNotifications.Count != 1 || reactionNotifications.UnreadCount != 1 || len(reactionNotifications.Records) != 1 ||
		reactionNotifications.Records[0].Type != port.NotificationTypeLike {
		t.Fatalf("unexpected reaction notifications: notifications=%+v err=%v", reactionNotifications, err)
	}
	if active, err := reactionRepo.Toggle(ctx, 1, 2, port.ReactionLike); err != nil || active {
		t.Fatalf("remove article reaction: active=%v err=%v", active, err)
	}
	reactionNotifications, err = followRepo.ListNotifications(ctx, 1, port.NotificationGroupReaction, 1, 10)
	if err != nil || reactionNotifications.Count != 0 || reactionNotifications.UnreadCount != 0 {
		t.Fatalf("removed reaction must be hidden: notifications=%+v err=%v", reactionNotifications, err)
	}
	if active, err := reactionRepo.Toggle(ctx, 1, 2, port.ReactionLike); err != nil || !active {
		t.Fatalf("reactivate article reaction: active=%v err=%v", active, err)
	}
	reactionNotifications, err = followRepo.ListNotifications(ctx, 1, port.NotificationGroupReaction, 1, 10)
	if err != nil || reactionNotifications.Count != 1 || reactionNotifications.UnreadCount != 1 {
		t.Fatalf("reactivated reaction must restore the original notification: notifications=%+v err=%v", reactionNotifications, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE t_user_info SET notify_interaction = 0 WHERE id = 1`); err != nil {
		t.Fatalf("disable interaction notifications: %v", err)
	}
	if active, err := reactionRepo.Toggle(ctx, 1, 2, port.ReactionFavorite); err != nil || !active {
		t.Fatalf("activate opted-out favorite: active=%v err=%v", active, err)
	}
	reactionNotifications, err = followRepo.ListNotifications(ctx, 1, port.NotificationGroupReaction, 1, 10)
	if err != nil || reactionNotifications.Count != 1 || reactionNotifications.UnreadCount != 1 {
		t.Fatalf("opted-out reactions must not create notifications: notifications=%+v err=%v", reactionNotifications, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE t_user_info SET notify_interaction = 1 WHERE id = 1`); err != nil {
		t.Fatalf("restore interaction notifications: %v", err)
	}
	preview, err := platformRepo.PreviewOwnedContent(ctx, 1, port.StudioContentArticle, port.StudioFilter{Status: 1})
	if err != nil || preview.Count < 1 || preview.MaxID < 1 {
		t.Fatalf("unexpected batch preview: preview=%+v err=%v", preview, err)
	}
	_, err = platformRepo.BatchUpdateOwnedContentStatus(ctx, 1, port.StudioContentArticle, port.StudioBatchScope{
		Mode: port.StudioBatchScopeFilter, Status: 1, MaxID: preview.MaxID, ExpectedCount: preview.Count + 1,
	}, 3, port.StudioAuditActor{UserID: 1, Nickname: "integration"})
	if !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("expected snapshot conflict, got %v", err)
	}
	mutation, err := platformRepo.BatchUpdateOwnedContentStatus(ctx, 1, port.StudioContentArticle, port.StudioBatchScope{
		Mode: port.StudioBatchScopeIDs, IDs: []int{3},
	}, 3, port.StudioAuditActor{UserID: 1, Nickname: "integration", IPAddress: "127.0.0.1"})
	if err != nil || mutation.Affected != 1 || mutation.AuditID <= 0 {
		t.Fatalf("unexpected batch mutation: mutation=%+v err=%v", mutation, err)
	}
	audits, auditCount, err := NewContentAuditRepo(xormEngine).List(ctx, port.ContentAuditFilter{Current: 1, Size: 10})
	if err != nil || auditCount < 2 || len(audits) < 2 || audits[0].Result != "success" || audits[0].AffectedCount != 1 {
		t.Fatalf("unexpected content audits: audits=%+v count=%d err=%v", audits, auditCount, err)
	}
	items, itemCount, err := NewContentAuditRepo(xormEngine).ListItems(ctx, audits[0].ID, 1, 10)
	if err != nil || itemCount != 1 || len(items) != 1 || items[0].ContentID != 3 || items[0].NextStatus != 3 {
		t.Fatalf("unexpected content audit items: items=%+v count=%d err=%v", items, itemCount, err)
	}
	if err := contentAnalytics.RecordContinuationEvent(ctx, 2, time.Now(), port.ContinuationEventSeriesClick, &port.ContinuationTarget{
		Type: port.ContinuationTargetArticle, Id: 3, Placement: port.ContinuationPlacementSeriesNext,
	}); err != nil {
		t.Fatalf("create continuation-only metric row: %v", err)
	}
	continuationOnly, err := contentAnalytics.GetArticleDailyMetrics(ctx, 2, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(continuationOnly) != 1 || continuationOnly[0].Views != 0 || continuationOnly[0].SeriesClicks != 1 {
		t.Fatalf("unexpected continuation-only metrics: metrics=%v err=%v", continuationOnly, err)
	}
	if err := contentAnalytics.RecordContinuationEvent(ctx, 1, time.Now(), port.ContinuationEventRelatedClick, &port.ContinuationTarget{
		Type: port.ContinuationTargetArticle, Id: 3, Placement: port.ContinuationPlacementRelated, Position: 1,
	}); err != nil {
		t.Fatalf("record attributed continuation click: %v", err)
	}
	if err := contentAnalytics.RecordContinuationEvent(ctx, 1, time.Now(), port.ContinuationEventRelatedClick, &port.ContinuationTarget{
		Type: port.ContinuationTargetArticle, Id: 3, Placement: port.ContinuationPlacementRelated, Position: 1,
	}); err != nil {
		t.Fatalf("record attributed continuation retry: %v", err)
	}
	targetMetrics, err := contentAnalytics.ListContinuationTargetMetrics(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"), 1)
	if err != nil || len(targetMetrics) != 1 || targetMetrics[0].TargetId != 3 || targetMetrics[0].Clicks != 2 ||
		targetMetrics[0].ModuleImpressions != 2 || targetMetrics[0].TargetTitle != "shared tag article" {
		t.Fatalf("unexpected continuation target metrics: metrics=%v err=%v", targetMetrics, err)
	}
	roles, err := NewRoleRepository(xormEngine).ListRolesByUserInfoID(ctx, 1)
	if err != nil || len(roles) != 1 || roles[0] != "user" {
		t.Fatalf("unexpected role repository result: roles=%v err=%v", roles, err)
	}
	menus, err := NewMenuRepo(xormEngine).ListByUserInfoID(ctx, 1)
	if err != nil || len(menus) != 1 || menus[0].Name != "integration menu" {
		t.Fatalf("unexpected menu repository result: menus=%v err=%v", menus, err)
	}
	albums, err := NewPhotoAlbumRepository(xormEngine).ListPublic(ctx)
	if err != nil || len(albums) != 1 || albums[0].AlbumName != "integration album" {
		t.Fatalf("unexpected album repository result: albums=%v err=%v", albums, err)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO t_article_daily_metric (article_id, metric_date, views, unique_readers, effective_sessions, total_active_ms, completed_sessions)
    VALUES (2, CURRENT_DATE, 4, 4, 4, 20000, 2)
    ON CONFLICT (article_id, metric_date) DO UPDATE SET unique_readers = 4, effective_sessions = 4;
INSERT INTO t_article_reaction (article_id, user_info_id, reaction) VALUES (4, 1, 'like'), (4, 2, 'like'), (4, 1, 'favorite');
INSERT INTO t_article_reaction (article_id, user_info_id, reaction, create_time) VALUES (6, 1, 'favorite', CURRENT_TIMESTAMP - INTERVAL '30 days');
INSERT INTO t_comment (user_id, topic_id, comment_content, type, is_delete, is_review, create_time) VALUES
    (2, 5, 'hot comment one', 1, 0, 1, CURRENT_TIMESTAMP),
    (1, 5, 'hot comment two', 1, 0, 1, CURRENT_TIMESTAMP),
    (2, 5, 'hot comment three', 1, 0, 1, CURRENT_TIMESTAMP);`); err != nil {
		t.Fatalf("seed discovery ranking signals: %v", err)
	}
	hotArticles, hotCount, err := platformRepo.ListFeedArticlesHot(ctx, 1, 10)
	if err != nil {
		t.Fatalf("list hot feed: %v", err)
	}
	// Article 1 keeps its two approved comments, one like, one favourite and one
	// daily reader; the seeded signals then rank 5, 4 and 2 behind it. Article 3
	// is a draft and article 6 only has an out-of-window reaction, so neither can
	// appear.
	wantHotOrder := []int{1, 5, 4, 2}
	if hotCount != len(wantHotOrder) || len(hotArticles) != len(wantHotOrder) {
		t.Fatalf("unexpected hot feed size: count=%d ids=%v want=%v", hotCount, articleIDs(hotArticles), wantHotOrder)
	}
	for index, article := range hotArticles {
		if article.Id != wantHotOrder[index] {
			t.Fatalf("unexpected hot order at %d: got %d want %d (records=%+v)", index, article.Id, wantHotOrder[index], hotArticles)
		}
	}
	if err := platformRepo.ModerateContent(ctx, "article", 2, 1, true, "integration discovery"); err != nil {
		t.Fatalf("hide hot article: %v", err)
	}
	hiddenHot, hiddenCount, err := platformRepo.ListAuthorArticlesHot(ctx, 1, 1, 10)
	if err != nil {
		t.Fatalf("list hidden author hot feed: %v", err)
	}
	for _, article := range hiddenHot {
		if article.Id == 2 {
			t.Fatalf("hidden article must leave the author hot feed: count=%d records=%+v", hiddenCount, hiddenHot)
		}
	}
	if err := platformRepo.ModerateContent(ctx, "article", 2, 1, false, ""); err != nil {
		t.Fatalf("restore hot article: %v", err)
	}
	overview, err := platformRepo.ListTopicOverview(ctx, 10)
	if err != nil {
		t.Fatalf("list topic overview: %v", err)
	}
	tagCounts := map[string]int{}
	for _, item := range overview.Tags {
		tagCounts[item.Name] = item.ArticleCount
	}
	if len(overview.Tags) == 0 || overview.Tags[0].Name != "integration tag" {
		t.Fatalf("hottest tag must lead the plaza: topics=%+v", overview.Tags)
	}
	if tagCounts["integration tag"] != 3 {
		t.Fatalf("draft article must not count towards its tag: tags=%+v", overview.Tags)
	}
	categoryCounts := map[string]int{}
	for _, item := range overview.Categories {
		categoryCounts[item.Name] = item.ArticleCount
	}
	if categoryCounts["integration category"] != 5 || categoryCounts["second category"] != 1 {
		t.Fatalf("unexpected category counts: categories=%+v", overview.Categories)
	}
	if len(overview.Series) != 1 || overview.Series[0].Id != 10 || overview.Series[0].ArticleCount != 3 {
		t.Fatalf("unexpected series plaza entries: series=%+v", overview.Series)
	}
	activeAuthors, activeCount, err := platformRepo.ListAuthors(ctx, 1, 10, 0, port.AuthorSortActive)
	if err != nil || activeCount != 1 || len(activeAuthors) != 1 || activeAuthors[0].LastPublishedAt == nil {
		t.Fatalf("unexpected active author board: authors=%+v count=%d err=%v", activeAuthors, activeCount, err)
	}
	followerAuthors, followerBoardCount, err := platformRepo.ListAuthors(ctx, 1, 10, 0, port.AuthorSortFollowers)
	if err != nil || followerBoardCount != 1 || len(followerAuthors) != 1 || followerAuthors[0].FollowerCount != 1 {
		t.Fatalf("unexpected follower author board: authors=%+v count=%d err=%v", followerAuthors, followerBoardCount, err)
	}
	articleAuthors, articleBoardCount, err := platformRepo.ListAuthors(ctx, 1, 10, 0, port.AuthorSortArticles)
	if err != nil || articleBoardCount != 1 || len(articleAuthors) != 1 || articleAuthors[0].Id != 1 {
		t.Fatalf("unexpected article author board: authors=%+v count=%d err=%v", articleAuthors, articleBoardCount, err)
	}
	topicRepo := NewTopicSubscriptionRepo(xormEngine)
	if err := topicRepo.Subscribe(ctx, 2, port.TopicTypeTag, "integration tag"); err != nil {
		t.Fatalf("subscribe to topic: %v", err)
	}
	if err := topicRepo.Subscribe(ctx, 2, port.TopicTypeTag, "integration tag"); err != nil {
		t.Fatalf("repeat subscribe must be idempotent: %v", err)
	}
	if err := topicRepo.Subscribe(ctx, 2, "series", "integration tag"); !apperrors.IsKind(err, apperrors.KindValidation) {
		t.Fatalf("series subscriptions are out of scope, got %v", err)
	}
	if err := topicRepo.Subscribe(ctx, 2, port.TopicTypeTag, "missing topic"); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("unknown topic must fail, got %v", err)
	}
	subscriptions, subscriptionCount, err := topicRepo.ListSubscriptions(ctx, 2, 1, 10)
	if err != nil || subscriptionCount != 1 || len(subscriptions) != 1 {
		t.Fatalf("unexpected subscriptions: subscriptions=%+v count=%d err=%v", subscriptions, subscriptionCount, err)
	}
	// Articles 1, 2 and 5 are still public under "integration tag"; article 3 was
	// turned into a draft earlier in this test.
	if subscriptions[0].ArticleCount != 3 || subscriptions[0].HotScore <= 0 || subscriptions[0].UnreadCount != 0 {
		t.Fatalf("unexpected subscription counters: %+v", subscriptions[0])
	}

	// A second author publishing under the same normalised tag must reach the
	// subscriber, and an article the subscriber owns must not notify them.
	readerCategoryID, err := ensureNamedCategoryForTest(ctx, db, "integration reader category")
	if err != nil {
		t.Fatalf("create reader category: %v", err)
	}
	readerArticleID, err := insertPublicArticle(ctx, db, 2, readerCategoryID, "reader topic article", "integration tag")
	if err != nil {
		t.Fatalf("insert reader topic article: %v", err)
	}
	authorArticleID, err := insertPublicArticle(ctx, db, 1, 1, "author topic article", "integration tag")
	if err != nil {
		t.Fatalf("insert author topic article: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO t_author_publish_event (author_id, content_type, content_id, published_at)
		VALUES (1, 'article', $1, CURRENT_TIMESTAMP), (2, 'article', $2, CURRENT_TIMESTAMP)`, authorArticleID, readerArticleID); err != nil {
		t.Fatalf("record topic publish events: %v", err)
	}
	// The topic feed carries every published article that matches a subscription,
	// including the reader's own; only the notification inbox skips self-authored
	// content.
	topicFeed, topicFeedCount, err := topicRepo.ListTopicFeed(ctx, 2, 1, 10)
	if err != nil || topicFeedCount != 2 || len(topicFeed) != 2 {
		t.Fatalf("unexpected topic feed: feed=%+v count=%d err=%v", topicFeed, topicFeedCount, err)
	}
	feedHasAuthorArticle := false
	for _, item := range topicFeed {
		if len(item.Topics) != 1 || item.Topics[0] != "integration tag" {
			t.Fatalf("topic feed must report the matched topic: %+v", item.Topics)
		}
		if item.ContentId == authorArticleID {
			feedHasAuthorArticle = true
		}
	}
	if !feedHasAuthorArticle {
		t.Fatalf("topic feed must carry the cross-author article: %+v", topicFeed)
	}
	topicNotifications, err := followRepo.ListNotifications(ctx, 2, port.NotificationGroupTopic, 1, 10)
	if err != nil || topicNotifications.Count != 1 || topicNotifications.UnreadCount != 1 {
		t.Fatalf("unexpected topic notifications: %+v err=%v", topicNotifications, err)
	}
	if err := topicRepo.SetMuted(ctx, 2, port.TopicTypeTag, "integration tag", true); err != nil {
		t.Fatalf("mute topic subscription: %v", err)
	}
	mutedNotifications, err := followRepo.ListNotifications(ctx, 2, port.NotificationGroupTopic, 1, 10)
	if err != nil || mutedNotifications.Count != 0 {
		t.Fatalf("muted subscription must not notify: %+v err=%v", mutedNotifications, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE t_user_info SET notify_topic = 0 WHERE id = 2`); err != nil {
		t.Fatalf("disable topic notifications: %v", err)
	}
	if err := topicRepo.SetMuted(ctx, 2, port.TopicTypeTag, "integration tag", false); err != nil {
		t.Fatalf("unmute topic subscription: %v", err)
	}
	disabledNotifications, err := followRepo.ListNotifications(ctx, 2, port.NotificationGroupTopic, 1, 10)
	if err != nil || disabledNotifications.Count != 0 {
		t.Fatalf("global topic switch must suppress notifications: %+v err=%v", disabledNotifications, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE t_user_info SET notify_topic = 1 WHERE id = 2`); err != nil {
		t.Fatalf("restore topic notifications: %v", err)
	}
	restoredNotifications, err := followRepo.ListNotifications(ctx, 2, port.NotificationGroupTopic, 1, 10)
	if err != nil || restoredNotifications.Count != 1 || restoredNotifications.UnreadCount != 1 {
		t.Fatalf("restored topic notifications: %+v err=%v", restoredNotifications, err)
	}
	if err := followRepo.MarkNotificationsRead(ctx, 2, restoredNotifications.ReadCursor); err != nil {
		t.Fatalf("mark topic notifications read: %v", err)
	}
	topicUnread, err := followRepo.UnreadNotificationCount(ctx, 2)
	if err != nil || topicUnread != 0 {
		t.Fatalf("topic notifications must be read: count=%d err=%v", topicUnread, err)
	}
	// The named lookup must aggregate the same tag name across both authors.
	namedArticles, namedCount, err := NewArticleRepo(xormEngine).ListArticlesByTagName(ctx, 1, 10, "Integration Tag")
	if err != nil || namedCount != 5 || len(namedArticles) != 5 {
		t.Fatalf("tag name lookup must aggregate every author: count=%d err=%v", namedCount, err)
	}
	authors := map[int]bool{}
	for _, article := range namedArticles {
		authors[article.UserId] = true
	}
	if !authors[1] || !authors[2] {
		t.Fatalf("tag name lookup must cross authors: authors=%v", authors)
	}
	if err := topicRepo.Unsubscribe(ctx, 2, port.TopicTypeTag, "integration tag"); err != nil {
		t.Fatalf("unsubscribe from topic: %v", err)
	}
	if _, count, err := topicRepo.ListTopicFeed(ctx, 2, 1, 10); err != nil || count != 0 {
		t.Fatalf("unsubscribed topic feed must be empty: count=%d err=%v", count, err)
	}
	// Recommendation fixtures cover the account-scoped ranking path, including
	// followed-author exclusion, local reading seeds and persistent feedback.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO t_user_info (id, handle, email, nickname, avatar, is_subscribe, is_disable)
		VALUES (3, 'recommend-author', 'rec-author@example.com', 'Recommend Author', '', 0, 0),
		       (4, 'followed-author', 'followed@example.com', 'Followed Author', '', 0, 0),
		       (5, 'archive-author', 'archive@example.com', 'Archive Author', '', 0, 0);
		INSERT INTO t_tag (id, tag_name) VALUES (5, 'recommendation topic');`); err != nil {
		t.Fatalf("insert recommendation users: %v", err)
	}
	recommendationCategoryID, err := ensureNamedCategoryForTest(ctx, db, "Recommendation Category")
	if err != nil {
		t.Fatalf("create recommendation category: %v", err)
	}
	recArticleA, err := insertPublicArticle(ctx, db, 3, recommendationCategoryID, "recommendation primary", "recommendation topic")
	if err != nil {
		t.Fatalf("insert recommendation primary: %v", err)
	}
	recArticleB, err := insertPublicArticle(ctx, db, 3, recommendationCategoryID, "recommendation secondary", "recommendation topic")
	if err != nil {
		t.Fatalf("insert recommendation secondary: %v", err)
	}
	recArticleC, err := insertPublicArticle(ctx, db, 3, recommendationCategoryID, "recommendation third", "recommendation topic")
	if err != nil {
		t.Fatalf("insert recommendation third: %v", err)
	}
	selfArticle, err := insertPublicArticle(ctx, db, 2, recommendationCategoryID, "recommendation self", "recommendation topic")
	if err != nil {
		t.Fatalf("insert self recommendation article: %v", err)
	}
	followedArticle, err := insertPublicArticle(ctx, db, 4, recommendationCategoryID, "recommendation followed", "recommendation topic")
	if err != nil {
		t.Fatalf("insert followed recommendation article: %v", err)
	}
	archiveArticle, err := insertPublicArticle(ctx, db, 5, recommendationCategoryID, "recommendation archive", "recommendation topic")
	if err != nil {
		t.Fatalf("insert archive recommendation article: %v", err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE t_article SET create_time = CURRENT_TIMESTAMP - INTERVAL '200 days' WHERE id = $1", archiveArticle); err != nil {
		t.Fatalf("age archive recommendation article: %v", err)
	}
	if err := topicRepo.Subscribe(ctx, 2, port.TopicTypeTag, "recommendation topic"); err != nil {
		t.Fatalf("subscribe recommendation topic: %v", err)
	}
	if err := followRepo.Follow(ctx, 2, 4); err != nil {
		t.Fatalf("follow recommendation author: %v", err)
	}
	recommendationRepo := NewRecommendationRepo(xormEngine)
	recFirstPage, err := recommendationRepo.ListRecommendations(ctx, port.RecommendationRequest{
		UserID: 2, Size: 2, Snapshot: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("list recommendations: %v", err)
	}
	if !recFirstPage.Personalized || len(recFirstPage.Items) == 0 || !recFirstPage.HasMore || recFirstPage.NextCursor == nil {
		t.Fatalf("expected a personalized first page with a cursor: %+v", recFirstPage)
	}
	seen := map[int]bool{}
	authorCounts := map[int]int{}
	for _, item := range recFirstPage.Items {
		if item.Id == selfArticle || item.Id == followedArticle {
			t.Fatalf("self and followed articles must be excluded: %+v", item)
		}
		seen[item.Id] = true
		authorCounts[item.UserId]++
	}
	recSecondPage, err := recommendationRepo.ListRecommendations(ctx, port.RecommendationRequest{
		UserID: 2, Size: 2, Snapshot: recFirstPage.NextCursor.Snapshot, Cursor: recFirstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("list recommendation cursor page: %v", err)
	}
	for _, item := range recSecondPage.Items {
		if seen[item.Id] {
			t.Fatalf("cursor returned a duplicate recommendation: %d", item.Id)
		}
	}
	if authorCounts[3] > 2 {
		t.Fatalf("one author must not exceed two items on a page: %v", authorCounts)
	}
	fullPage, err := recommendationRepo.ListRecommendations(ctx, port.RecommendationRequest{
		UserID: 2, Size: 20, Snapshot: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("list complete recommendation page: %v", err)
	}
	foundSubscribed, foundArchive := false, false
	for _, item := range fullPage.Items {
		if item.Id == recArticleA || item.Id == recArticleB || item.Id == recArticleC {
			if item.Reason.Type != port.RecommendationReasonSubscribedTopic {
				t.Fatalf("topic-matched recommendation must explain the subscription: %+v", item)
			}
			foundSubscribed = true
		}
		if item.Id == archiveArticle {
			foundArchive = true
		}
	}
	if !foundSubscribed || !foundArchive {
		t.Fatalf("subscription and archive fallback must appear: %+v", fullPage.Items)
	}
	hidden, err := recommendationRepo.UpsertFeedback(ctx, 2, port.RecommendationFeedbackInput{
		TargetType: port.RecommendationTargetArticle, ArticleID: recArticleA,
	})
	if err != nil || hidden.ID <= 0 {
		t.Fatalf("hide recommendation article: feedback=%+v err=%v", hidden, err)
	}
	withoutHidden, err := recommendationRepo.ListRecommendations(ctx, port.RecommendationRequest{UserID: 2, Size: 20, Snapshot: time.Now().UTC()})
	if err != nil {
		t.Fatalf("list recommendations after hide: %v", err)
	}
	for _, item := range withoutHidden.Items {
		if item.Id == recArticleA {
			t.Fatalf("hidden article was recommended again: %d", item.Id)
		}
	}
	if _, err := recommendationRepo.UpsertFeedback(ctx, 2, port.RecommendationFeedbackInput{
		TargetType: port.RecommendationTargetAuthor, AuthorID: 3,
	}); err != nil {
		t.Fatalf("mute recommendation author: %v", err)
	}
	feedbackItems, feedbackCount, err := recommendationRepo.ListFeedback(ctx, 2, 1, 10)
	if err != nil || feedbackCount != 2 || len(feedbackItems) != 2 {
		t.Fatalf("list recommendation feedback: items=%+v count=%d err=%v", feedbackItems, feedbackCount, err)
	}
	if err := recommendationRepo.DeleteFeedback(ctx, 2, hidden.ID); err != nil {
		t.Fatalf("restore recommendation article: %v", err)
	}
	coldStart, err := recommendationRepo.ListRecommendations(ctx, port.RecommendationRequest{UserID: 5, Size: 10, Snapshot: time.Now().UTC()})
	if err != nil || coldStart.Personalized || len(coldStart.Items) == 0 {
		t.Fatalf("cold-start recommendations must stay available: page=%+v err=%v", coldStart, err)
	}

	collectionRepo := NewCollectionRepo(xormEngine)
	collection, err := collectionRepo.CreateOwned(ctx, 1, "integration-reading-list", port.CollectionSaveInput{
		Title: "Integration reading list", Description: "public article path", Visibility: port.CollectionVisibilityPublic,
	})
	if err != nil || collection.ID <= 0 {
		t.Fatalf("create collection: collection=%+v err=%v", collection, err)
	}
	if _, total, err := collectionRepo.ListPublic(ctx, port.CollectionSortLatest, 1, 10); err != nil || total != 0 {
		t.Fatalf("empty public collection must stay undiscoverable: total=%d err=%v", total, err)
	}
	if err := collectionRepo.AddItem(ctx, 1, collection.ID, 1, "first note"); err != nil {
		t.Fatalf("add first collection item: %v", err)
	}
	if err := collectionRepo.AddItem(ctx, 1, collection.ID, 2, "second note"); err != nil {
		t.Fatalf("add second collection item: %v", err)
	}
	publicCollections, total, err := collectionRepo.ListPublic(ctx, port.CollectionSortHot, 1, 10)
	if err != nil || total != 1 || len(publicCollections) != 1 || publicCollections[0].ID != collection.ID || publicCollections[0].HotScore <= 0 {
		t.Fatalf("unexpected public collection page: items=%+v total=%d err=%v", publicCollections, total, err)
	}
	if err := collectionRepo.Reorder(ctx, 1, collection.ID, []int{2, 1}); err != nil {
		t.Fatalf("reorder collection: %v", err)
	}
	detail, err := collectionRepo.GetPublicBySlug(ctx, collection.Slug)
	if err != nil || len(detail.Items) != 2 || detail.Items[0].ArticleID != 2 || detail.Items[0].Note != "second note" {
		t.Fatalf("unexpected ordered collection: detail=%+v err=%v", detail, err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE t_article SET status = 2 WHERE id = 1"); err != nil {
		t.Fatalf("make collected article private: %v", err)
	}
	if _, total, err := collectionRepo.ListPublic(ctx, port.CollectionSortLatest, 1, 10); err != nil || total != 1 {
		t.Fatalf("collection with one remaining article must stay public: total=%d err=%v", total, err)
	}
	if detail, err := collectionRepo.GetPublicBySlug(ctx, collection.Slug); err != nil || len(detail.Items) != 2 || detail.Items[0].ArticleID != 2 || detail.Items[1].ArticleID != 1 || detail.Items[1].Available {
		t.Fatalf("owner-facing collection detail must retain unavailable references: detail=%+v err=%v", detail, err)
	}
	if _, err := collectionRepo.UpdateOwned(ctx, 1, collection.ID, port.CollectionSaveInput{
		Title: collection.Title, Description: collection.Description, Visibility: port.CollectionVisibilityUnlisted,
	}); err != nil {
		t.Fatalf("make collection unlisted: %v", err)
	}
	if _, total, err := collectionRepo.ListPublic(ctx, port.CollectionSortLatest, 1, 10); err != nil || total != 0 {
		t.Fatalf("unlisted collection must leave discovery: total=%d err=%v", total, err)
	}
	if _, err := collectionRepo.GetPublicBySlug(ctx, collection.Slug); err != nil {
		t.Fatalf("unlisted collection must stay reachable by slug: %v", err)
	}
	if err := platformRepo.ModerateContent(ctx, "collection", collection.ID, 1, true, "integration collection"); err != nil {
		t.Fatalf("hide collection: %v", err)
	}
	if _, err := collectionRepo.GetPublicBySlug(ctx, collection.Slug); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("hidden collection must not be publicly reachable: %v", err)
	}
	if adminCollections, total, err := collectionRepo.ListAdmin(ctx, 1, 10, "hidden", "Integration reading"); err != nil || total != 1 || len(adminCollections) != 1 {
		t.Fatalf("hidden collection must remain visible to admins: items=%+v total=%d err=%v", adminCollections, total, err)
	}
	if err := platformRepo.ModerateContent(ctx, "collection", collection.ID, 1, false, ""); err != nil {
		t.Fatalf("restore collection: %v", err)
	}
	if _, err := collectionRepo.UpdateOwned(ctx, 2, collection.ID, port.CollectionSaveInput{
		Title: "stolen", Description: "", Visibility: port.CollectionVisibilityPublic,
	}); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("non-owner collection update must be rejected: %v", err)
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8.10.0-alpine3.23",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start isolated redis: %v", err)
	}
	t.Cleanup(func() { _ = redisContainer.Terminate(context.Background()) })
	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("resolve redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("resolve redis port: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port())})
	defer client.Close()
	if err := client.Set(ctx, "integration:probe", "ok", time.Minute).Err(); err != nil {
		t.Fatalf("write redis probe: %v", err)
	}
	value, err := client.Get(ctx, "integration:probe").Result()
	if err != nil || value != "ok" {
		t.Fatalf("read redis probe: value=%q err=%v", value, err)
	}
	if err := client.PFAdd(ctx, "content:readers:day-one", "reader-a", "reader-b").Err(); err != nil {
		t.Fatalf("write first reader hyperloglog: %v", err)
	}
	if err := client.PFAdd(ctx, "content:readers:day-two", "reader-b", "reader-c").Err(); err != nil {
		t.Fatalf("write second reader hyperloglog: %v", err)
	}
	uniqueReaders, err := client.PFCount(ctx, "content:readers:day-one", "content:readers:day-two").Result()
	if err != nil || uniqueReaders != 3 {
		t.Fatalf("unexpected hyperloglog union count: count=%d err=%v", uniqueReaders, err)
	}
}

func articleIDs(cards []*port.ArticleCard) []int {
	ids := make([]int, 0, len(cards))
	for _, card := range cards {
		if card != nil {
			ids = append(ids, card.Id)
		}
	}
	return ids
}

// insertPublicArticle creates one public visible article owned by authorID and
// attaches the named tag to it, returning the new article id.
func insertPublicArticle(ctx context.Context, db *sql.DB, authorID, categoryID int, title, tagName string) (int, error) {
	var articleID int
	if err := db.QueryRowContext(ctx, `
		INSERT INTO t_article
			(id, user_id, category_id, article_title, article_content, is_top, is_featured, is_delete, status, type, create_time, update_time)
		VALUES ((SELECT COALESCE(MAX(id), 0) + 1 FROM t_article), $1, $2, $3, 'topic fixture content', 0, 0, 0, 1, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id`, authorID, categoryID, title).Scan(&articleID); err != nil {
		return 0, err
	}
	var tagID int
	if err := db.QueryRowContext(ctx, `
		SELECT id FROM t_tag WHERE lower(btrim(tag_name)) = lower(btrim($1)) ORDER BY id ASC LIMIT 1`, tagName).Scan(&tagID); err != nil {
		return 0, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO t_article_tag (id, article_id, tag_id)
		VALUES ((SELECT COALESCE(MAX(id), 0) + 1 FROM t_article_tag), $1, $2)`, articleID, tagID); err != nil {
		return 0, err
	}
	return articleID, nil
}

func ensureNamedCategoryForTest(ctx context.Context, db *sql.DB, name string) (int, error) {
	var id int
	err := db.QueryRowContext(ctx, `
		SELECT id FROM t_category WHERE lower(btrim(category_name)) = lower(btrim($1)) ORDER BY id ASC LIMIT 1`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	if err := db.QueryRowContext(ctx, `
		INSERT INTO t_category (id, category_name)
		VALUES ((SELECT COALESCE(MAX(id), 0) + 1 FROM t_category), $1)
		RETURNING id`, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
func pingDatabase(ctx context.Context, db *sql.DB) error {
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(deadline); err == nil {
			return nil
		}
		if deadline.Err() != nil {
			return deadline.Err()
		}
		time.Sleep(250 * time.Millisecond)
	}
}
