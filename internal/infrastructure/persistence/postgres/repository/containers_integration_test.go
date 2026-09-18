//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

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
    email VARCHAR(50), nickname VARCHAR(50) NOT NULL, avatar VARCHAR(1024) NOT NULL,
    intro VARCHAR(255), website VARCHAR(255), is_subscribe SMALLINT DEFAULT 0,
    notify_comment SMALLINT NOT NULL DEFAULT 1,
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
CREATE TABLE t_talk (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, content TEXT NOT NULL, images TEXT, is_top SMALLINT NOT NULL, status SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_comment (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, topic_id INTEGER, comment_content TEXT NOT NULL, reply_user_id INTEGER, parent_id INTEGER, type SMALLINT NOT NULL, is_delete SMALLINT NOT NULL, is_review SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
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
CREATE TABLE t_article (
    id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, category_id INTEGER, article_cover VARCHAR(1024),
    article_title VARCHAR(50) NOT NULL, article_content TEXT NOT NULL, article_content_html TEXT,
    series_id INTEGER, series_order INTEGER,
    is_top SMALLINT NOT NULL,
    is_featured SMALLINT NOT NULL, is_delete SMALLINT NOT NULL, status SMALLINT NOT NULL,
    type SMALLINT NOT NULL, password VARCHAR(255), original_url VARCHAR(255),
    create_time TIMESTAMP, update_time TIMESTAMP
);
CREATE TABLE t_article_tag (id INTEGER PRIMARY KEY, article_id INTEGER NOT NULL, tag_id INTEGER NOT NULL);
CREATE TABLE t_article_daily_metric (
    id SERIAL PRIMARY KEY, article_id INTEGER NOT NULL REFERENCES t_article(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL, views BIGINT NOT NULL DEFAULT 0, unique_readers BIGINT NOT NULL DEFAULT 0,
    effective_sessions BIGINT NOT NULL DEFAULT 0, total_active_ms BIGINT NOT NULL DEFAULT 0,
    completed_sessions BIGINT NOT NULL DEFAULT 0, create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, UNIQUE (article_id, metric_date)
);`); err != nil {
		t.Fatalf("create repository fixtures: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO t_user_info (id, email, nickname, avatar, is_subscribe, is_disable) VALUES (1, 'integration@example.com', 'integration', '', 0, 0);
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
INSERT INTO t_article (id, user_id, category_id, article_title, article_content, series_id, series_order, is_top, is_featured, is_delete, status, type) VALUES (1, 1, 1, 'integration article', 'content', 10, 1, 0, 0, 0, 1, 1);
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
INSERT INTO t_article_daily_metric (article_id, metric_date, views, unique_readers, effective_sessions, total_active_ms, completed_sessions) VALUES (1, CURRENT_DATE, 2, 1, 1, 5000, 1);`); err != nil {
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
	site := NewSiteInfoRepo(xormEngine)
	articleCount, err := site.CountArticles(ctx)
	if err != nil || articleCount != 6 {
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
	dailyMetrics, err := contentAnalytics.GetArticleDailyMetrics(ctx, 1, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(dailyMetrics) != 1 || dailyMetrics[0].Views != 3 || dailyMetrics[0].CompletedSessions != 2 {
		t.Fatalf("unexpected content analytics daily metrics: metrics=%v err=%v", dailyMetrics, err)
	}
	siteDailyMetrics, err := contentAnalytics.ListDailyMetrics(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(siteDailyMetrics) != 1 || siteDailyMetrics[0].EffectiveSessions != 2 {
		t.Fatalf("unexpected site content analytics metrics: metrics=%v err=%v", siteDailyMetrics, err)
	}
	articleMetrics, err := contentAnalytics.ListArticleMetrics(ctx, time.Now().Format("2006-01-02"), time.Now().Format("2006-01-02"))
	if err != nil || len(articleMetrics) != 1 || articleMetrics[0].ArticleId != 1 || articleMetrics[0].CompletedSessions != 2 {
		t.Fatalf("unexpected article content analytics metrics: metrics=%v err=%v", articleMetrics, err)
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
