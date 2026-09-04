//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/migration"

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
CREATE TABLE t_menu (id INTEGER GENERATED BY DEFAULT AS IDENTITY (START WITH 1000) PRIMARY KEY, name VARCHAR(50) NOT NULL, path VARCHAR(100) NOT NULL, component VARCHAR(100) NOT NULL, icon VARCHAR(50) NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP, order_num SMALLINT NOT NULL, parent_id INTEGER, is_hidden SMALLINT NOT NULL);
CREATE TABLE t_resource (id INTEGER GENERATED BY DEFAULT AS IDENTITY (START WITH 1000) PRIMARY KEY, resource_name VARCHAR(50) NOT NULL, url VARCHAR(255), request_method VARCHAR(10), parent_id INTEGER, is_anonymous SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_role_menu (id INTEGER GENERATED BY DEFAULT AS IDENTITY (START WITH 1000) PRIMARY KEY, role_id INTEGER, menu_id INTEGER);
CREATE TABLE t_role_resource (id INTEGER GENERATED BY DEFAULT AS IDENTITY (START WITH 1000) PRIMARY KEY, role_id INTEGER, resource_id INTEGER);
CREATE TABLE t_photo_album (id INTEGER PRIMARY KEY, album_name VARCHAR(50) NOT NULL, album_desc VARCHAR(100) NOT NULL, album_cover VARCHAR(255) NOT NULL, is_delete SMALLINT NOT NULL, status SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_photo (id INTEGER PRIMARY KEY, album_id INTEGER NOT NULL, photo_name VARCHAR(50) NOT NULL, photo_desc VARCHAR(100), photo_src VARCHAR(255) NOT NULL, is_delete SMALLINT NOT NULL, create_time TIMESTAMP, update_time TIMESTAMP);
CREATE TABLE t_article (
    id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, category_id INTEGER, article_cover VARCHAR(1024),
    article_title VARCHAR(50) NOT NULL, article_content TEXT NOT NULL, is_top SMALLINT NOT NULL,
    is_featured SMALLINT NOT NULL, is_delete SMALLINT NOT NULL, status SMALLINT NOT NULL,
    type SMALLINT NOT NULL, password VARCHAR(255), original_url VARCHAR(255),
    create_time TIMESTAMP, update_time TIMESTAMP
);`); err != nil {
		t.Fatalf("create repository fixtures: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO t_user_info (id, email, nickname, avatar, is_subscribe, is_disable) VALUES (1, 'integration@example.com', 'integration', '', 0, 0);
INSERT INTO t_user_auth (id, user_info_id, username, password, login_type) VALUES (1, 1, 'integration@example.com', 'hashed-password', 1);
INSERT INTO t_role (id, role_name, is_disable) VALUES (1, 'user', 0);
INSERT INTO t_user_role (id, user_id, role_id) VALUES (1, 1, 1);
INSERT INTO t_category (id, category_name) VALUES (1, 'integration category');
INSERT INTO t_tag (id, tag_name) VALUES (1, 'integration tag');
INSERT INTO t_talk (id, user_id, content, is_top, status) VALUES (1, 1, 'integration talk', 0, 1);
INSERT INTO t_website_config (id, config) VALUES (1, '{"name":"integration"}');
INSERT INTO t_about (id, content) VALUES (1, '{"content":"integration"}');
INSERT INTO t_menu (id, name, path, component, icon, order_num, parent_id, is_hidden) VALUES (1, 'integration menu', '/', 'Layout', '', 1, 0, 0);
INSERT INTO t_resource (id, resource_name, url, request_method, parent_id, is_anonymous) VALUES (1, 'integration resource', '/integration', 'GET', 0, 0);
INSERT INTO t_role_menu (id, role_id, menu_id) VALUES (1, 1, 1);
INSERT INTO t_role_resource (id, role_id, resource_id) VALUES (1, 1, 1);
INSERT INTO t_photo_album (id, album_name, album_desc, album_cover, is_delete, status) VALUES (1, 'integration album', 'integration', '', 0, 1);
INSERT INTO t_photo (id, album_id, photo_name, photo_src, is_delete) VALUES (1, 1, 'integration photo', 'https://example.com/photo.jpg', 0);
INSERT INTO t_article (id, user_id, article_title, article_content, is_top, is_featured, is_delete, status, type) VALUES
    (1, 1, 'integration article', 'content', 0, 0, 0, 1, 1),
    (2, 1, 'private article', 'private content', 0, 0, 0, 2, 1),
    (3, 1, 'deleted article', 'deleted content', 0, 0, 1, 1, 1);`); err != nil {
		t.Fatalf("insert repository fixtures: %v", err)
	}
	if err := verifyMigrationFailureRecovery(ctx, db, pgHost, pgPort.Port()); err != nil {
		t.Fatalf("verify migration failure recovery: %v", err)
	}

	xormEngine, err := xorm.NewEngine("postgres", fmt.Sprintf("postgres://integration:integration@%s:%s/integration?sslmode=disable", pgHost, pgPort.Port()))
	if err != nil {
		t.Fatalf("open xorm repository engine: %v", err)
	}
	defer xormEngine.Close()
	if err := migration.NewRunner(xormEngine).Up(ctx); err != nil {
		t.Fatalf("apply isolated migrations: %v", err)
	}
	if err := migration.NewRunner(xormEngine).Up(ctx); err != nil {
		t.Fatalf("re-run isolated migrations for idempotency: %v", err)
	}
	if err := verifyAIReviewCreate(ctx, xormEngine); err != nil {
		t.Fatalf("verify AI review create lifecycle: %v", err)
	}
	if err := verifyAgentMemoryHistoryAndConflictLifecycle(ctx, xormEngine); err != nil {
		t.Fatalf("verify agent memory history/conflict lifecycle: %v", err)
	}
	profileRepository := NewAgentProfileRepository(xormEngine)
	profile := port.AgentProfile{
		ID:            "integration-profile",
		Name:          "Integration Agent",
		PromptVersion: "v1",
		SystemPrompt:  "保持公开资料边界",
		Opening:       "你好",
		RhythmPrompts: map[port.AgentRhythmPhase]string{
			port.AgentRhythmAwake: "清醒",
			port.AgentRhythmDusk:  "黄昏",
			port.AgentRhythmNight: "深夜",
		},
		Enabled: true,
	}
	if err := profileRepository.Save(ctx, profile); err != nil {
		t.Fatalf("save agent profile: %v", err)
	}
	storedProfile, err := profileRepository.Get(ctx, profile.ID)
	if err != nil || storedProfile.SystemPrompt != profile.SystemPrompt || storedProfile.RhythmPrompts[port.AgentRhythmNight] != "深夜" || !storedProfile.Enabled {
		t.Fatalf("stored agent profile = %+v, error=%v", storedProfile, err)
	}
	profile.Enabled = false
	if err := profileRepository.Save(ctx, profile); err != nil {
		t.Fatalf("disable agent profile: %v", err)
	}
	storedProfile, err = profileRepository.Get(ctx, profile.ID)
	if err != nil || storedProfile.Enabled {
		t.Fatalf("disabled agent profile = %+v, error=%v", storedProfile, err)
	}
	policyRepository := NewAgentReviewPolicyRepository(xormEngine)
	policy := port.DefaultAgentReviewPolicy()
	policy.ID = "integration-policy"
	policy.ReviewTTL = 2 * time.Hour
	if err := policyRepository.Save(ctx, policy); err != nil {
		t.Fatalf("save agent review policy: %v", err)
	}
	storedPolicy, err := policyRepository.Get(ctx, policy.ID)
	if err != nil || storedPolicy.Version != 1 || storedPolicy.ReviewTTL != 2*time.Hour || !storedPolicy.ReviewRequired {
		t.Fatalf("stored agent review policy = %+v, error=%v", storedPolicy, err)
	}
	policy.Version = storedPolicy.Version
	policy.ReviewTTL = 3 * time.Hour
	if err := policyRepository.Save(ctx, policy); err != nil {
		t.Fatalf("update agent review policy: %v", err)
	}
	storedPolicy, err = policyRepository.Get(ctx, policy.ID)
	if err != nil || storedPolicy.Version != 2 || storedPolicy.ReviewTTL != 3*time.Hour {
		t.Fatalf("updated agent review policy = %+v, error=%v", storedPolicy, err)
	}
	videoRepository := NewVideoRepository(xormEngine)
	if err := videoRepository.Create(ctx, port.Video{
		ID:        "integration-video-1",
		Title:     "integration video",
		Source:    port.VideoSourceExternal,
		URL:       "https://video.example/embed/1",
		EmbedURL:  "https://video.example/embed/1",
		Published: true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create video repository fixture: %v", err)
	}
	videos, videoCount, err := videoRepository.ListPublic(ctx, 1, 10)
	if err != nil || videoCount != 1 || len(videos) != 1 || videos[0].ID != "integration-video-1" {
		t.Fatalf("unexpected video repository result: count=%d videos=%+v err=%v", videoCount, videos, err)
	}
	if err := videoRepository.Delete(ctx, "integration-video-1"); err != nil {
		t.Fatalf("delete video repository fixture: %v", err)
	}
	if videos, videoCount, err := videoRepository.ListPublic(ctx, 1, 10); err != nil || videoCount != 0 || len(videos) != 0 {
		t.Fatalf("deleted video remained public: count=%d videos=%+v err=%v", videoCount, videos, err)
	}
	aiJobs := NewAIJobRepository(xormEngine)
	job := port.AIJob{
		ID:             "integration-ai-job-1",
		Kind:           port.AIJobKindArticleIndex,
		IdempotencyKey: "integration:article:1:index:1",
		Payload:        []byte(`{"articleId":1,"action":"upsert"}`),
		RunAfter:       time.Now().UTC().Add(-time.Second),
		MaxAttempts:    port.DefaultAIJobMaxAttempts,
	}
	if err := aiJobs.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue AI job: %v", err)
	}
	job.ID = "integration-ai-job-duplicate"
	if err := aiJobs.Enqueue(ctx, job); err != nil {
		t.Fatalf("enqueue duplicate AI job: %v", err)
	}
	claimed, ok, err := aiJobs.Claim(ctx, "integration-worker", time.Now().UTC(), time.Minute)
	if err != nil || !ok {
		t.Fatalf("claim AI job: job=%+v ok=%v err=%v", claimed, ok, err)
	}
	if claimed.ID != "integration-ai-job-1" || claimed.Attempts != 1 || claimed.Status != port.AIJobRunning {
		t.Fatalf("unexpected claimed AI job: %+v", claimed)
	}
	if err := aiJobs.Complete(ctx, claimed.ID, "integration-worker", port.AIJobResult{RunID: "integration-run-1", Payload: []byte(`{"indexed":true}`)}); err != nil {
		t.Fatalf("complete AI job: %v", err)
	}
	backfillStates := NewArticleIndexBackfillRepository(xormEngine)
	backfill, err := backfillStates.CreateOrGet(ctx, port.ArticleIndexBackfillState{
		ID:                 "integration-article-backfill",
		IndexUID:           "article_chunks_v1",
		IndexVersion:       "v1",
		Provider:           "openai",
		Model:              "text-embedding-3-small",
		ModelVersion:       "integration",
		Dimension:          3,
		EmbeddingBatchSize: 2,
		PageSize:           10,
		Status:             port.ArticleIndexBackfillPending,
	})
	if err != nil || backfill.Status != port.ArticleIndexBackfillPending {
		t.Fatalf("create backfill state: state=%+v err=%v", backfill, err)
	}
	if err := backfillStates.RequestPause(ctx, backfill.ID); err != nil {
		t.Fatalf("request backfill pause: %v", err)
	}
	if err := backfillStates.Resume(ctx, backfill.ID); err != nil {
		t.Fatalf("resume pending backfill: %v", err)
	}
	claimedBackfill, ok, err := backfillStates.Claim(ctx, backfill.ID, "integration-backfill-worker", time.Now().UTC(), time.Minute)
	if err != nil || !ok || claimedBackfill.Status != port.ArticleIndexBackfillRunning {
		t.Fatalf("claim backfill state: state=%+v ok=%v err=%v", claimedBackfill, ok, err)
	}
	if err := backfillStates.Checkpoint(ctx, backfill.ID, "integration-backfill-worker", 1, 1, 2, time.Now().UTC(), time.Minute); err != nil {
		t.Fatalf("checkpoint backfill state: %v", err)
	}
	if err := backfillStates.RequestPause(ctx, backfill.ID); err != nil {
		t.Fatalf("request running backfill pause: %v", err)
	}
	if err := backfillStates.Pause(ctx, backfill.ID, "integration-backfill-worker", time.Now().UTC()); err != nil {
		t.Fatalf("pause backfill state: %v", err)
	}
	pausedBackfill, err := backfillStates.Get(ctx, backfill.ID)
	if err != nil || pausedBackfill.Status != port.ArticleIndexBackfillPaused || pausedBackfill.Cursor != 1 || pausedBackfill.ProcessedArticles != 1 {
		t.Fatalf("unexpected paused backfill state: state=%+v err=%v", pausedBackfill, err)
	}
	if err := backfillStates.Resume(ctx, backfill.ID); err != nil {
		t.Fatalf("resume paused backfill: %v", err)
	}
	if _, ok, err := backfillStates.Claim(ctx, backfill.ID, "integration-backfill-worker-2", time.Now().UTC(), time.Minute); err != nil || !ok {
		t.Fatalf("reclaim paused backfill: ok=%v err=%v", ok, err)
	}
	if err := backfillStates.Complete(ctx, backfill.ID, "integration-backfill-worker-2", time.Now().UTC()); err != nil {
		t.Fatalf("complete backfill state: %v", err)
	}
	authUser, err := NewUserAuthRepo(xormEngine).FindByUsername(ctx, "integration@example.com")
	if err != nil {
		t.Fatalf("read auth repository fixture: %v", err)
	}
	if authUser.Info.Nickname != "integration" || len(authUser.Roles) != 1 || authUser.Roles[0] != "user" {
		t.Fatalf("unexpected auth repository result: %+v", authUser)
	}
	articleRepo := NewArticleRepo(xormEngine)
	articles, count, err := articleRepo.ListArchives(ctx, 1, 10)
	if err != nil {
		t.Fatalf("read article repository fixture: %v", err)
	}
	if count != 1 || len(articles) != 1 || articles[0].ArticleTitle != "integration article" {
		t.Fatalf("unexpected article repository result: count=%d articles=%+v", count, articles)
	}
	publicCards, publicCount, err := articleRepo.ListArticles(ctx, 1, 10)
	if err != nil {
		t.Fatalf("read public article list fixture: %v", err)
	}
	if publicCount != 1 || len(publicCards) != 1 || publicCards[0].ArticleTitle != "integration article" {
		t.Fatalf("public article list returned private content: count=%d articles=%+v", publicCount, publicCards)
	}
	indexSources, err := articleRepo.ListPublicArticleIndexSources(ctx, 0, 10)
	if err != nil {
		t.Fatalf("read article index source fixture: %v", err)
	}
	if len(indexSources) != 1 || indexSources[0].Article.ArticleContent != "content" || indexSources[0].Article.Id != 1 {
		t.Fatalf("unexpected article index source result: %+v", indexSources)
	}
	site := NewSiteInfoRepo(xormEngine)
	articleCount, err := site.CountArticles(ctx)
	if err != nil || articleCount != 1 {
		t.Fatalf("unexpected site article count: count=%d err=%v", articleCount, err)
	}
	rankedArticles, err := site.ListArticleRank(ctx, []int{1})
	if err != nil || len(rankedArticles) != 1 || rankedArticles[0].ArticleTitle != "integration article" {
		t.Fatalf("unexpected article rank result: articles=%v err=%v", rankedArticles, err)
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
}

func verifyAIReviewCreate(ctx context.Context, engine *xorm.Engine) error {
	repository := NewAIReviewRepository(engine)
	now := time.Now().UTC().Truncate(time.Microsecond)
	content := "integration review preview"
	review := port.AIReview{
		ID:            "integration-ai-review-create",
		TargetType:    "draft",
		TargetID:      "integration-draft-1",
		SessionID:     "admin:1",
		ContentDigest: port.ReviewContentDigest(content),
		BoundAt:       now,
		Operation:     "vision",
		Content:       content,
		Status:        port.ReviewPending,
		RunID:         "integration-vision-run",
		ReviewerID:    "1",
		CreatedAt:     now,
		UpdatedAt:     now,
		ExpiresAt:     now.Add(time.Hour),
	}
	if err := repository.Create(ctx, review); err != nil {
		return fmt.Errorf("create review: %w", err)
	}
	stored, err := repository.Get(ctx, review.ID)
	if err != nil {
		return fmt.Errorf("get created review: %w", err)
	}
	if stored.ID != review.ID || stored.Status != port.ReviewPending || stored.Operation != review.Operation || stored.Content != content || stored.SessionID != review.SessionID || stored.ContentDigest != review.ContentDigest {
		return fmt.Errorf("created review does not match input: %+v", stored)
	}
	reviews, count, err := repository.List(ctx, port.ReviewFilter{TargetID: review.TargetID, Current: 1, Size: 10})
	if err != nil {
		return fmt.Errorf("list created review: %w", err)
	}
	if count != 1 || len(reviews) != 1 || reviews[0].ID != review.ID {
		return fmt.Errorf("created review was not listed: count=%d reviews=%+v", count, reviews)
	}
	return nil
}

func verifyAgentMemoryHistoryAndConflictLifecycle(ctx context.Context, engine *xorm.Engine) error {
	repository := NewAgentMemoryRepository(engine)
	base := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	revisionFirst := port.AgentMemoryAssertion{
		ID: "integration-memory-revision-a", SubjectKey: "user:7", Predicate: "preferred_editor", Object: "Vim",
		SourceType: "manual", SourceID: "integration-revision-a", Version: 1, Confidence: 0.8,
		Status: port.AgentMemoryActive, ValidFrom: base, CreatedAt: base, UpdatedAt: base,
	}
	revisionPeer := port.AgentMemoryAssertion{
		ID: "integration-memory-revision-b", SubjectKey: "user:7", Predicate: "preferred_editor", Object: "Vim",
		SourceType: "manual", SourceID: "integration-revision-b", Version: 1, Confidence: 0.7,
		Status: port.AgentMemoryActive, ValidFrom: base, CreatedAt: base, UpdatedAt: base.Add(time.Minute),
	}
	if err := repository.Upsert(ctx, revisionFirst); err != nil {
		return fmt.Errorf("insert revision assertion: %w", err)
	}
	if err := repository.Upsert(ctx, revisionPeer); err != nil {
		return fmt.Errorf("insert same-object peer assertion: %w", err)
	}
	revisionFirst.Object = "Emacs"
	revisionFirst.UpdatedAt = base.Add(2 * time.Minute)
	if err := repository.Upsert(ctx, revisionFirst); err != nil {
		return fmt.Errorf("revise assertion into conflict: %w", err)
	}
	revisionConflicts, err := repository.ListConflicts(ctx, port.AgentMemoryConflictFilter{SubjectKey: "user:7", Predicate: "preferred_editor"})
	if err != nil || len(revisionConflicts) != 1 || len(revisionConflicts[0].Members) != 2 {
		return fmt.Errorf("revision conflict = %+v, err=%v", revisionConflicts, err)
	}
	revisionHistory, err := repository.History(ctx, revisionFirst.ID)
	if err != nil || len(revisionHistory) != 3 || revisionHistory[1].Reason != "revised_previous" || revisionHistory[2].Status != port.AgentMemoryConflicted {
		return fmt.Errorf("revised assertion history = %+v, err=%v", revisionHistory, err)
	}
	if err := repository.ResolveConflict(ctx, revisionConflicts[0].ID, revisionFirst.ID, "operator_selected_revision"); err != nil {
		return fmt.Errorf("resolve revised memory conflict: %w", err)
	}

	first := port.AgentMemoryAssertion{
		ID: "integration-memory-a", SubjectKey: "user:7", Predicate: "favorite_language", Object: "Go",
		SourceType: "manual", SourceID: "integration-note-a", Version: 1, Confidence: 0.8,
		Status: port.AgentMemoryActive, ValidFrom: base, CreatedAt: base, UpdatedAt: base,
	}
	if err := repository.Upsert(ctx, first); err != nil {
		return fmt.Errorf("insert first assertion: %w", err)
	}
	history, err := repository.History(ctx, first.ID)
	if err != nil || len(history) != 1 || history[0].Reason != "created" || history[0].Status != port.AgentMemoryActive {
		return fmt.Errorf("first assertion history = %+v, err=%v", history, err)
	}
	if err := repository.Upsert(ctx, first); err != nil {
		return fmt.Errorf("idempotent assertion upsert: %w", err)
	}
	history, err = repository.History(ctx, first.ID)
	if err != nil || len(history) != 1 {
		return fmt.Errorf("idempotent assertion history = %+v, err=%v", history, err)
	}
	second := port.AgentMemoryAssertion{
		ID: "integration-memory-b", SubjectKey: "user:7", Predicate: "favorite_language", Object: "Rust",
		SourceType: "conversation", SourceID: "integration-note-b", Version: 1, Confidence: 0.7,
		Status: port.AgentMemoryActive, ValidFrom: base, CreatedAt: base, UpdatedAt: base.Add(time.Minute),
	}
	if err := repository.Upsert(ctx, second); err != nil {
		return fmt.Errorf("insert conflicting assertion: %w", err)
	}
	conflicts, err := repository.ListConflicts(ctx, port.AgentMemoryConflictFilter{SubjectKey: "user:7", Predicate: "favorite_language"})
	if err != nil || len(conflicts) != 1 || conflicts[0].Status != port.AgentMemoryConflictOpen || len(conflicts[0].Members) != 2 {
		return fmt.Errorf("open memory conflicts = %+v, err=%v", conflicts, err)
	}
	if err := repository.ResolveConflict(ctx, conflicts[0].ID, first.ID, "operator_selected_manual_source"); err != nil {
		return fmt.Errorf("resolve memory conflict: %w", err)
	}
	resolved, err := repository.ListConflicts(ctx, port.AgentMemoryConflictFilter{SubjectKey: "user:7", Predicate: "favorite_language", Status: port.AgentMemoryConflictResolved})
	if err != nil || len(resolved) != 1 || resolved[0].WinnerAssertionID != first.ID || resolved[0].Members[0].Role == "" {
		return fmt.Errorf("resolved memory conflicts = %+v, err=%v", resolved, err)
	}
	active, err := repository.List(ctx, port.AgentMemoryFilter{SubjectKey: "user:7", Predicate: "favorite_language", Status: port.AgentMemoryActive})
	if err != nil || len(active) != 1 || active[0].ID != first.ID {
		return fmt.Errorf("active memory assertions after resolution = %+v, err=%v", active, err)
	}
	firstHistory, err := repository.History(ctx, first.ID)
	if err != nil || len(firstHistory) != 3 || firstHistory[len(firstHistory)-1].Reason != "conflict_resolved" {
		return fmt.Errorf("resolved assertion history = %+v, err=%v", firstHistory, err)
	}
	secondHistory, err := repository.History(ctx, second.ID)
	if err != nil || len(secondHistory) != 2 || secondHistory[len(secondHistory)-1].Status != port.AgentMemoryStale {
		return fmt.Errorf("losing assertion history = %+v, err=%v", secondHistory, err)
	}
	rejectFirst := port.AgentMemoryAssertion{
		ID: "integration-memory-reject-a", SubjectKey: "user:7", Predicate: "preferred_shell", Object: "fish",
		SourceType: "manual", SourceID: "integration-reject-a", Version: 1, Confidence: 0.8,
		Status: port.AgentMemoryActive, ValidFrom: base, CreatedAt: base, UpdatedAt: base,
	}
	rejectSecond := rejectFirst
	rejectSecond.ID = "integration-memory-reject-b"
	rejectSecond.SourceID = "integration-reject-b"
	rejectSecond.Object = "zsh"
	rejectSecond.UpdatedAt = base.Add(time.Minute)
	if err := repository.Upsert(ctx, rejectFirst); err != nil {
		return fmt.Errorf("insert rejection candidate: %w", err)
	}
	if err := repository.Upsert(ctx, rejectSecond); err != nil {
		return fmt.Errorf("insert second rejection candidate: %w", err)
	}
	rejectionConflicts, err := repository.ListConflicts(ctx, port.AgentMemoryConflictFilter{SubjectKey: "user:7", Predicate: "preferred_shell"})
	if err != nil || len(rejectionConflicts) != 1 || len(rejectionConflicts[0].Members) != 2 {
		return fmt.Errorf("rejection conflict = %+v, err=%v", rejectionConflicts, err)
	}
	if err := repository.RejectConflict(ctx, rejectionConflicts[0].ID, "operator_rejected_both"); err != nil {
		return fmt.Errorf("reject memory conflict: %w", err)
	}
	rejected, err := repository.List(ctx, port.AgentMemoryFilter{SubjectKey: "user:7", Predicate: "preferred_shell", Status: port.AgentMemoryRetracted})
	if err != nil || len(rejected) != 2 {
		return fmt.Errorf("retracted conflict candidates = %+v, err=%v", rejected, err)
	}
	return nil
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

func verifyMigrationFailureRecovery(ctx context.Context, db *sql.DB, host, port string) error {
	const schema = "migration_failure_probe"
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		return fmt.Errorf("create probe schema: %w", err)
	}

	engine, err := xorm.NewEngine("postgres", fmt.Sprintf("postgres://integration:integration@%s:%s/integration?sslmode=disable&search_path=%s", host, port, schema))
	if err != nil {
		return fmt.Errorf("open probe engine: %w", err)
	}
	defer engine.Close()

	failingMigrations := fstest.MapFS{
		"migrations/0001_probe.sql": &fstest.MapFile{Data: []byte("CREATE TABLE migration_failure_probe_ok (id INTEGER PRIMARY KEY);")},
		"migrations/0002_fail.sql":  &fstest.MapFile{Data: []byte("CREATE TABLE migration_failure_probe_rolled_back (id INTEGER PRIMARY KEY); SELECT * FROM migration_failure_probe_missing;")},
	}
	if err := migration.NewRunnerWithFS(engine, failingMigrations).Up(ctx); err == nil {
		return fmt.Errorf("failing migration unexpectedly succeeded")
	}
	applied, err := engine.QueryString("SELECT version, name FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("read probe migration state: %w", err)
	}
	if len(applied) != 1 || applied[0]["version"] != "1" || applied[0]["name"] != "probe" {
		return fmt.Errorf("failed migration was not rolled back cleanly: %+v", applied)
	}

	recoveredMigrations := fstest.MapFS{
		"migrations/0001_probe.sql":     &fstest.MapFile{Data: []byte("CREATE TABLE migration_failure_probe_ok (id INTEGER PRIMARY KEY);")},
		"migrations/0002_recovered.sql": &fstest.MapFile{Data: []byte("CREATE TABLE migration_failure_probe_recovered (id INTEGER PRIMARY KEY);")},
	}
	if err := migration.NewRunnerWithFS(engine, recoveredMigrations).Up(ctx); err != nil {
		return fmt.Errorf("re-run recovered migrations: %w", err)
	}
	if err := migration.NewRunnerWithFS(engine, recoveredMigrations).Up(ctx); err != nil {
		return fmt.Errorf("re-run recovered migrations for idempotency: %w", err)
	}
	applied, err = engine.QueryString("SELECT version, name FROM schema_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("read recovered migration state: %w", err)
	}
	if len(applied) != 2 || applied[1]["version"] != "2" || applied[1]["name"] != "recovered" {
		return fmt.Errorf("recovered migration state = %+v", applied)
	}
	return nil
}
