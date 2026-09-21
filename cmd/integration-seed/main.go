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
			(article_id, metric_date, views, unique_readers, effective_sessions, total_active_ms, completed_sessions)
		VALUES ($1, CURRENT_DATE, 3, 2, 2, 40000, 1)
		ON CONFLICT (article_id, metric_date)
		DO UPDATE SET unique_readers = 2, effective_sessions = 2`, articleID); err != nil {
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
