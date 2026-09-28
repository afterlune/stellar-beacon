package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type userSeed struct {
	email    string
	password string
	nickname string
	handle   string
	roleID   int
}

const (
	fixtureArticleTitle  = "Integration notification fixture"
	fixtureArticleBody   = "Stable public content used by the isolated frontend/backend integration suite."
	fixtureAuthorComment = "Integration author root comment"
	// Both seeded authors publish under this normalised topic name so the suite
	// can prove that topic aggregation and subscription cross author boundaries.
	fixtureTopicName          = "Integration Topic"
	fixtureReaderArticleTitle = "Integration reader topic fixture"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := envOr("INTEGRATION_DB_HOST", "127.0.0.1")
	port := envOr("INTEGRATION_DB_PORT", "15432")
	database := envOr("INTEGRATION_DB_NAME", "stellar_beacon")
	username := envOr("INTEGRATION_DB_USER", "postgres")
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		fail("POSTGRES_PASSWORD is required")
	}

	dsn := (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   host + ":" + port,
		Path:   "/" + database,
		RawQuery: url.Values{
			"sslmode": []string{"disable"},
		}.Encode(),
	}).String()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fail("open database: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		fail("ping database: %v", err)
	}

	seeds := []userSeed{
		{
			email:    required("E2E_ADMIN_EMAIL"),
			password: required("E2E_ADMIN_PASSWORD"),
			nickname: "integration-admin",
			handle:   "e2e-admin",
			roleID:   1,
		},
		{
			email:    required("E2E_USER_EMAIL"),
			password: required("E2E_USER_PASSWORD"),
			nickname: "integration-user",
			handle:   "e2e-user",
			roleID:   2,
		},
	}
	userIDs := make([]int, 0, len(seeds))
	for _, seed := range seeds {
		userID, err := ensureUser(ctx, db, seed)
		if err != nil {
			fail("seed %s: %v", seed.email, err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := ensureIntegrationFixture(ctx, db, userIDs[0], userIDs[1]); err != nil {
		fail("seed integration fixture: %v", err)
	}
	if err := ensureTopicFixture(ctx, db, userIDs[0], userIDs[1]); err != nil {
		fail("seed topic fixture: %v", err)
	}
	if err := ensureTalkFixture(ctx, db, userIDs[0]); err != nil {
		fail("seed talk fixture: %v", err)
	}
	fmt.Println("integration users, public fixture and notification preferences are ready")
}

func ensureUser(ctx context.Context, db *sql.DB, seed userSeed) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(seed.password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var authID, userInfoID int
	err = tx.QueryRowContext(ctx, `SELECT id, user_info_id FROM t_user_auth WHERE username = $1 ORDER BY id LIMIT 1`, seed.email).Scan(&authID, &userInfoID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		userInfoID, err = insertUserInfo(ctx, tx, seed)
		if err != nil {
			return 0, err
		}
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO t_user_auth
				(user_info_id, username, password, login_type, ip_address, ip_source, create_time)
			VALUES ($1, $2, $3, 1, '127.0.0.1', 'integration', NOW())
			RETURNING id`, userInfoID, seed.email, string(hash)).Scan(&authID); err != nil {
			return 0, fmt.Errorf("insert auth: %w", err)
		}
	case err != nil:
		return 0, fmt.Errorf("find auth: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_user_info
			SET email = $1, nickname = $2, handle = $3, is_disable = 0,
			    notify_comment = 1, notify_interaction = 1, update_time = NOW()
			WHERE id = $4`, seed.email, seed.nickname, seed.handle, userInfoID); err != nil {
			return 0, fmt.Errorf("update user info: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_user_auth
			SET password = $1, login_type = 1, update_time = NOW()
			WHERE id = $2`, string(hash), authID); err != nil {
			return 0, fmt.Errorf("update auth: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO t_user_role (user_id, role_id)
		SELECT $1, $2
		WHERE NOT EXISTS (
			SELECT 1 FROM t_user_role WHERE user_id = $1 AND role_id = $2
		)`, userInfoID, seed.roleID); err != nil {
		return 0, fmt.Errorf("ensure role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return userInfoID, nil
}

func insertUserInfo(ctx context.Context, tx *sql.Tx, seed userSeed) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO t_user_info
			(email, handle, nickname, avatar, intro, website, is_subscribe, notify_comment, notify_interaction, is_disable, create_time)
		VALUES ($1, $2, $3, '', 'integration test user', '', 0, 1, 1, 0, NOW())
		RETURNING id`, seed.email, seed.handle, seed.nickname).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert user info: %w", err)
	}
	return id, nil
}

func ensureIntegrationFixture(ctx context.Context, db *sql.DB, authorID, readerID int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fixture transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var categoryID int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MIN(id), 0) FROM t_category WHERE user_id = $1`, authorID).Scan(&categoryID); err != nil {
		return fmt.Errorf("find fixture category: %w", err)
	}

	var articleID int
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM t_article
		WHERE user_id = $1 AND article_title = $2 AND is_delete = 0
		ORDER BY id ASC
		LIMIT 1`, authorID, fixtureArticleTitle).Scan(&articleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO t_article
				(user_id, category_id, article_cover, article_title, article_content, article_content_html,
				 series_id, series_order, is_top, is_featured, is_delete, status, moderation_status,
				 type, password, original_url, create_time, update_time)
			VALUES ($1, $2, '', $3, $4, $5, 0, 0, 0, 0, 0, 1, 'visible', 1, '', '', NOW(), NOW())
			RETURNING id`, authorID, categoryID, fixtureArticleTitle, fixtureArticleBody, "<p>"+fixtureArticleBody+"</p>").Scan(&articleID); err != nil {
			return fmt.Errorf("insert fixture article: %w", err)
		}
	case err != nil:
		return fmt.Errorf("find fixture article: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_article
			SET category_id = $1, article_content = $2, article_content_html = $3,
			    status = 1, moderation_status = 'visible', moderation_reason = '',
			    moderated_by = 0, moderated_at = NULL, is_delete = 0, update_time = NOW()
			WHERE id = $4`, categoryID, fixtureArticleBody, "<p>"+fixtureArticleBody+"</p>", articleID); err != nil {
			return fmt.Errorf("update fixture article: %w", err)
		}
	}

	var rootCommentID int
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM t_comment
		WHERE user_id = $1 AND type = 1 AND topic_id = $2 AND comment_content = $3
		ORDER BY id ASC
		LIMIT 1`, authorID, articleID, fixtureAuthorComment).Scan(&rootCommentID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO t_comment
				(user_id, topic_id, comment_content, reply_user_id, parent_id, type,
				 is_delete, is_review, notification_dispatched_at, create_time, update_time)
			VALUES ($1, $2, $3, 0, 0, 1, 0, 1, NOW(), NOW(), NOW())
			RETURNING id`, authorID, articleID, fixtureAuthorComment).Scan(&rootCommentID); err != nil {
			return fmt.Errorf("insert fixture root comment: %w", err)
		}
	case err != nil:
		return fmt.Errorf("find fixture root comment: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_comment
			SET is_delete = 0, is_review = 1, notification_dispatched_at = COALESCE(notification_dispatched_at, NOW()), update_time = NOW()
			WHERE id = $1`, rootCommentID); err != nil {
			return fmt.Errorf("update fixture root comment: %w", err)
		}
	}

	// A deterministic read signal keeps the public "hot" ranking non-empty and
	// comparable across runs, even before the E2E flow adds reactions.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO t_article_daily_metric
			(article_id, metric_date, views, unique_readers, effective_sessions, total_active_ms, completed_sessions, create_time, update_time)
		VALUES ($1, CURRENT_DATE, 3, 2, 2, 40000, 1, NOW(), NOW())
		ON CONFLICT (article_id, metric_date)
		DO UPDATE SET unique_readers = 2, effective_sessions = 2, update_time = NOW()`, articleID); err != nil {
		return fmt.Errorf("seed fixture discovery metric: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM t_user_notification WHERE content_type = 'article' AND content_id = $1`, articleID); err != nil {
		return fmt.Errorf("reset fixture notifications: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM t_article_reaction WHERE article_id = $1`, articleID); err != nil {
		return fmt.Errorf("reset fixture reactions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM t_comment WHERE type = 1 AND topic_id = $1 AND id <> $2`, articleID, rootCommentID); err != nil {
		return fmt.Errorf("reset fixture comments: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM t_user_follow WHERE follower_id = $1 AND author_id = $2`, readerID, authorID); err != nil {
		return fmt.Errorf("reset fixture follow: %w", err)
	}

	var rawConfig string
	config := map[string]interface{}{}
	err = tx.QueryRowContext(ctx, `SELECT config FROM t_website_config WHERE id = 1`).Scan(&rawConfig)
	if err == nil && rawConfig != "" {
		if unmarshalErr := json.Unmarshal([]byte(rawConfig), &config); unmarshalErr != nil {
			config = map[string]interface{}{}
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read website config: %w", err)
	}
	config["isCommentReview"] = 1
	config["isEmailNotice"] = 1
	encodedConfig, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode website config: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE t_website_config SET config = $1, update_time = NOW() WHERE id = 1`, string(encodedConfig))
	if err != nil {
		return fmt.Errorf("update website config: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read website config update result: %w", err)
	}
	if affected == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO t_website_config (config, create_time, update_time) VALUES ($1, NOW(), NOW())`, string(encodedConfig)); err != nil {
			return fmt.Errorf("insert website config: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fixture transaction: %w", err)
	}
	return nil
}

func ensureTalkFixture(ctx context.Context, db *sql.DB, authorID int) error {
	const content = "Integration readonly talk fixture"
	var talkID int
	err := db.QueryRowContext(ctx, `
		SELECT id FROM t_talk
		WHERE user_id = $1 AND content = $2
		ORDER BY id ASC LIMIT 1`, authorID, content).Scan(&talkID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := db.QueryRowContext(ctx, `
			INSERT INTO t_talk
				(user_id, content, images, is_top, status, moderation_status, create_time, update_time)
			VALUES ($1, $2, '', 0, 1, 'visible', NOW(), NOW())
			RETURNING id`, authorID, content).Scan(&talkID); err != nil {
			return fmt.Errorf("insert readonly talk fixture: %w", err)
		}
	case err != nil:
		return fmt.Errorf("find readonly talk fixture: %w", err)
	default:
		if _, err := db.ExecContext(ctx, `
			UPDATE t_talk
			SET status = 1, moderation_status = 'visible', update_time = NOW()
			WHERE id = $1`, talkID); err != nil {
			return fmt.Errorf("refresh readonly talk fixture: %w", err)
		}
	}
	return nil
}

// ensureNamedCategory returns the author's category with the given name,
// creating it when missing. Categories and tags are owned per account, so the
// shared topic name has to exist once per author.
func ensureNamedCategory(ctx context.Context, tx *sql.Tx, userID int, name string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM t_category
		WHERE user_id = $1 AND lower(btrim(category_name)) = lower(btrim($2))
		ORDER BY id ASC LIMIT 1`, userID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("find category %q: %w", name, err)
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO t_category (user_id, category_name, create_time, update_time)
		VALUES ($1, $2, NOW(), NOW()) RETURNING id`, userID, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert category %q: %w", name, err)
	}
	return id, nil
}

func ensureNamedTag(ctx context.Context, tx *sql.Tx, userID int, name string) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM t_tag
		WHERE user_id = $1 AND lower(btrim(tag_name)) = lower(btrim($2))
		ORDER BY id ASC LIMIT 1`, userID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("find tag %q: %w", name, err)
	}
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO t_tag (user_id, tag_name, create_time, update_time)
		VALUES ($1, $2, NOW(), NOW()) RETURNING id`, userID, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert tag %q: %w", name, err)
	}
	return id, nil
}

func ensureArticleTag(ctx context.Context, tx *sql.Tx, articleID, tagID int) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO t_article_tag (article_id, tag_id)
		SELECT $1, $2
		WHERE NOT EXISTS (SELECT 1 FROM t_article_tag WHERE article_id = $1 AND tag_id = $2)`,
		articleID, tagID); err != nil {
		return fmt.Errorf("attach tag %d to article %d: %w", tagID, articleID, err)
	}
	return nil
}

func recordFixturePublishEvent(ctx context.Context, tx *sql.Tx, authorID, articleID int) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO t_author_publish_event (author_id, content_type, content_id, published_at)
		VALUES ($1, 'article', $2, NOW())
		ON CONFLICT (content_type, content_id) DO NOTHING`, authorID, articleID); err != nil {
		return fmt.Errorf("record publish event for article %d: %w", articleID, err)
	}
	return nil
}

// ensureTopicFixture gives both seeded authors a public article that shares the
// same normalised category and tag name, which is what the topic subscription
// and topic plaza assertions rely on.
func ensureTopicFixture(ctx context.Context, db *sql.DB, authorID, readerID int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin topic fixture transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var authorArticleID int
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM t_article
		WHERE user_id = $1 AND article_title = $2 AND is_delete = 0
		ORDER BY id ASC LIMIT 1`, authorID, fixtureArticleTitle).Scan(&authorArticleID); err != nil {
		return fmt.Errorf("find author fixture article: %w", err)
	}
	authorCategoryID, err := ensureNamedCategory(ctx, tx, authorID, fixtureTopicName)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE t_article SET category_id = $1, update_time = NOW()
		WHERE id = $2 AND category_id <> $1`, authorCategoryID, authorArticleID); err != nil {
		return fmt.Errorf("move author fixture article: %w", err)
	}
	authorTagID, err := ensureNamedTag(ctx, tx, authorID, fixtureTopicName)
	if err != nil {
		return err
	}
	if err := ensureArticleTag(ctx, tx, authorArticleID, authorTagID); err != nil {
		return err
	}
	if err := recordFixturePublishEvent(ctx, tx, authorID, authorArticleID); err != nil {
		return err
	}

	readerCategoryID, err := ensureNamedCategory(ctx, tx, readerID, fixtureTopicName)
	if err != nil {
		return err
	}
	readerTagID, err := ensureNamedTag(ctx, tx, readerID, fixtureTopicName)
	if err != nil {
		return err
	}
	var readerArticleID int
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM t_article
		WHERE user_id = $1 AND article_title = $2 AND is_delete = 0
		ORDER BY id ASC LIMIT 1`, readerID, fixtureReaderArticleTitle).Scan(&readerArticleID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO t_article
				(user_id, category_id, article_cover, article_title, article_content, article_content_html,
				 series_id, series_order, is_top, is_featured, is_delete, status, moderation_status,
				 type, password, original_url, create_time, update_time)
			VALUES ($1, $2, '', $3, $4, $5, 0, 0, 0, 0, 0, 1, 'visible', 1, '', '', NOW(), NOW())
			RETURNING id`, readerID, readerCategoryID, fixtureReaderArticleTitle, fixtureArticleBody, "<p>"+fixtureArticleBody+"</p>").Scan(&readerArticleID); err != nil {
			return fmt.Errorf("insert reader topic article: %w", err)
		}
	case err != nil:
		return fmt.Errorf("find reader topic article: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_article
			SET category_id = $1, status = 1, moderation_status = 'visible', is_delete = 0, update_time = NOW()
			WHERE id = $2`, readerCategoryID, readerArticleID); err != nil {
			return fmt.Errorf("update reader topic article: %w", err)
		}
	}
	if err := ensureArticleTag(ctx, tx, readerArticleID, readerTagID); err != nil {
		return err
	}
	if err := recordFixturePublishEvent(ctx, tx, readerID, readerArticleID); err != nil {
		return err
	}
	return tx.Commit()
}

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		fail("%s is required", name)
	}
	return value
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
