package main

import (
	"context"
	"database/sql"
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
	roleID   int
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host := envOr("INTEGRATION_DB_HOST", "127.0.0.1")
	port := envOr("INTEGRATION_DB_PORT", "15432")
	database := envOr("INTEGRATION_DB_NAME", "benetnasch")
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
			roleID:   1,
		},
		{
			email:    required("E2E_USER_EMAIL"),
			password: required("E2E_USER_PASSWORD"),
			nickname: "integration-user",
			roleID:   2,
		},
	}
	for _, seed := range seeds {
		if err := ensureUser(ctx, db, seed); err != nil {
			fail("seed %s: %v", seed.email, err)
		}
	}
	if err := ensureArticleVisibilityFixtures(ctx, db); err != nil {
		fail("seed article visibility fixtures: %v", err)
	}
	fmt.Println("integration users and article visibility fixtures are ready")
}

type articleVisibilityFixture struct {
	id       int
	title    string
	content  string
	isDelete int
	status   int
}

// These rows deliberately contain unique sentinel terms. They make the
// integration retrieval gate prove that private and deleted source rows are
// not projected into the public article_chunks index. The IDs are outside the
// normal fixture range and the operation is idempotent.
func ensureArticleVisibilityFixtures(ctx context.Context, db *sql.DB) error {
	fixtures := []articleVisibilityFixture{
		{
			id:       1000001,
			title:    "integration private index sentinel",
			content:  "q9z7private20260901xk",
			isDelete: 0,
			status:   2,
		},
		{
			id:       1000002,
			title:    "integration deleted index sentinel",
			content:  "q9z7deleted20260901xk",
			isDelete: 1,
			status:   1,
		},
	}

	for _, fixture := range fixtures {
		_, err := db.ExecContext(ctx, `
			INSERT INTO t_article
				(id, user_id, category_id, article_cover, article_title, article_content,
				 is_top, is_featured, is_delete, status, type, password, original_url,
				 create_time, update_time)
			VALUES ($1, 1, NULL, '', $2, $3, 0, 0, $4, $5, 1, NULL, NULL, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				article_title = EXCLUDED.article_title,
				article_content = EXCLUDED.article_content,
				is_delete = EXCLUDED.is_delete,
				status = EXCLUDED.status,
				update_time = NOW()`,
			fixture.id, fixture.title, fixture.content, fixture.isDelete, fixture.status)
		if err != nil {
			return fmt.Errorf("upsert article %d: %w", fixture.id, err)
		}
	}
	return nil
}

func ensureUser(ctx context.Context, db *sql.DB, seed userSeed) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(seed.password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
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
			return err
		}
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO t_user_auth
				(user_info_id, username, password, login_type, ip_address, ip_source, create_time)
			VALUES ($1, $2, $3, 1, '127.0.0.1', 'integration', NOW())
			RETURNING id`, userInfoID, seed.email, string(hash)).Scan(&authID); err != nil {
			return fmt.Errorf("insert auth: %w", err)
		}
	case err != nil:
		return fmt.Errorf("find auth: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_user_info
			SET email = $1, nickname = $2, is_disable = 0, update_time = NOW()
			WHERE id = $3`, seed.email, seed.nickname, userInfoID); err != nil {
			return fmt.Errorf("update user info: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE t_user_auth
			SET password = $1, login_type = 1, update_time = NOW()
			WHERE id = $2`, string(hash), authID); err != nil {
			return fmt.Errorf("update auth: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO t_user_role (user_id, role_id)
		SELECT $1, $2
		WHERE NOT EXISTS (
			SELECT 1 FROM t_user_role WHERE user_id = $1 AND role_id = $2
		)`, userInfoID, seed.roleID); err != nil {
		return fmt.Errorf("ensure role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func insertUserInfo(ctx context.Context, tx *sql.Tx, seed userSeed) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO t_user_info
			(email, nickname, avatar, intro, website, is_subscribe, is_disable, create_time)
		VALUES ($1, $2, '', 'integration test user', '', 0, 0, NOW())
		RETURNING id`, seed.email, seed.nickname).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert user info: %w", err)
	}
	return id, nil
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
