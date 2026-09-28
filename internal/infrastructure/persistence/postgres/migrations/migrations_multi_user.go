package migrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"xorm.io/xorm"
)

// applyMultiUserSchema introduces public author identities, owner-scoped
// taxonomy, moderation metadata and the public/private content boundary.
func applyMultiUserSchema(ctx context.Context, engine *xorm.Engine) error {
	checkSession := engine.NewSession().Context(ctx)
	defer checkSession.Close()
	var applied bool
	if _, err := checkSession.SQL("SELECT EXISTS (SELECT 1 FROM " + migrationTable + " WHERE version = 15)").Get(&applied); err != nil {
		return fmt.Errorf("check multi-user migration: %w", err)
	}
	if applied {
		return nil
	}

	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin multi-user migration: %w", err)
	}
	defer session.Rollback()

	for _, statement := range []string{
		`ALTER TABLE t_user_info ADD COLUMN IF NOT EXISTS handle VARCHAR(40)`,
		`ALTER TABLE t_category ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_tag ADD COLUMN IF NOT EXISTS user_id INTEGER`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 1`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_series ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_article ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(16) NOT NULL DEFAULT 'visible'`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderation_reason VARCHAR(255) NULL`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderated_by INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE t_talk ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ NULL`,
		`UPDATE t_article SET status = 1 WHERE status = 2 AND COALESCE(password, '') <> ''`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("apply multi-user schema: %w", err)
		}
	}

	var ownerID int
	if _, err := session.SQL(`
		SELECT ui.id
		FROM t_user_info ui
		JOIN t_user_role ur ON ur.user_id = ui.id
		JOIN t_role r ON r.id = ur.role_id
		WHERE r.role_name = 'admin'
		ORDER BY ui.id ASC
		LIMIT 1`).Get(&ownerID); err != nil {
		return fmt.Errorf("find legacy content owner: %w", err)
	}
	if ownerID == 0 {
		if _, err := session.SQL(`SELECT id FROM t_user_info ORDER BY id ASC LIMIT 1`).Get(&ownerID); err != nil {
			return fmt.Errorf("find fallback content owner: %w", err)
		}
	}

	var users []entity.TUserInfo
	if err := session.Cols("id", "handle", "nickname", "email").OrderBy("id ASC").Find(&users); err != nil {
		return fmt.Errorf("load users for handles: %w", err)
	}
	usedHandles := make(map[string]struct{}, len(users))
	for _, user := range users {
		if handle := strings.TrimSpace(strings.ToLower(user.Handle)); validHandle(handle) {
			usedHandles[handle] = struct{}{}
		}
	}
	for _, user := range users {
		if strings.TrimSpace(user.Handle) != "" {
			continue
		}
		base := handleBase(user.Nickname)
		if base == "" {
			base = handleBase(strings.Split(user.Email, "@")[0])
		}
		if base == "" {
			base = fmt.Sprintf("user-%d", user.Id)
		}
		base = truncateHandle(base, 32)
		handle := base
		for suffix := 2; ; suffix++ {
			if _, exists := usedHandles[handle]; !exists {
				break
			}
			handle = fmt.Sprintf("%s-%d", base, suffix)
		}
		usedHandles[handle] = struct{}{}
		if _, err := session.ID(user.Id).Cols("handle").Update(&entity.TUserInfo{Handle: handle}); err != nil {
			return fmt.Errorf("backfill user handle %d: %w", user.Id, err)
		}
	}

	if ownerID > 0 {
		for _, statement := range []string{
			`UPDATE t_article SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_series SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_category SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
			`UPDATE t_tag SET user_id = ? WHERE user_id IS NULL OR user_id <= 0`,
		} {
			if _, err := session.Exec(statement, ownerID); err != nil {
				return fmt.Errorf("backfill content owner: %w", err)
			}
		}
	}

	type duplicateName struct {
		UserId int    `xorm:"user_id"`
		Name   string `xorm:"normalized_name"`
		Id     int    `xorm:"id"`
	}
	var categoryDuplicates []duplicateName
	if err := session.SQL(`
		SELECT user_id, lower(btrim(category_name)) AS normalized_name, id
		FROM (
			SELECT id, user_id, category_name,
			       row_number() OVER (PARTITION BY user_id, lower(btrim(category_name)) ORDER BY id) AS rn
			FROM t_category
		) ranked
		WHERE rn > 1`).Find(&categoryDuplicates); err != nil {
		return fmt.Errorf("find duplicate categories: %w", err)
	}
	for _, duplicate := range categoryDuplicates {
		var keepID int
		if _, err := session.SQL(`
			SELECT id FROM t_category
			WHERE user_id = ? AND lower(btrim(category_name)) = ?
			ORDER BY id ASC LIMIT 1`, duplicate.UserId, duplicate.Name).Get(&keepID); err != nil {
			return fmt.Errorf("resolve duplicate category: %w", err)
		}
		if keepID == 0 || keepID == duplicate.Id {
			continue
		}
		if _, err := session.Exec(`UPDATE t_article SET category_id = ? WHERE category_id = ?`, keepID, duplicate.Id); err != nil {
			return fmt.Errorf("remap duplicate category: %w", err)
		}
		if _, err := session.Exec(`DELETE FROM t_category WHERE id = ?`, duplicate.Id); err != nil {
			return fmt.Errorf("delete duplicate category: %w", err)
		}
	}

	var tagDuplicates []duplicateName
	if err := session.SQL(`
		SELECT user_id, lower(btrim(tag_name)) AS normalized_name, id
		FROM (
			SELECT id, user_id, tag_name,
			       row_number() OVER (PARTITION BY user_id, lower(btrim(tag_name)) ORDER BY id) AS rn
			FROM t_tag
		) ranked
		WHERE rn > 1`).Find(&tagDuplicates); err != nil {
		return fmt.Errorf("find duplicate tags: %w", err)
	}
	for _, duplicate := range tagDuplicates {
		var keepID int
		if _, err := session.SQL(`
			SELECT id FROM t_tag
			WHERE user_id = ? AND lower(btrim(tag_name)) = ?
			ORDER BY id ASC LIMIT 1`, duplicate.UserId, duplicate.Name).Get(&keepID); err != nil {
			return fmt.Errorf("resolve duplicate tag: %w", err)
		}
		if keepID == 0 || keepID == duplicate.Id {
			continue
		}
		if _, err := session.Exec(`UPDATE t_article_tag SET tag_id = ? WHERE tag_id = ?`, keepID, duplicate.Id); err != nil {
			return fmt.Errorf("remap duplicate tag: %w", err)
		}
		if _, err := session.Exec(`DELETE FROM t_tag WHERE id = ?`, duplicate.Id); err != nil {
			return fmt.Errorf("delete duplicate tag: %w", err)
		}
	}
	if _, err := session.Exec(`
		DELETE FROM t_article_tag
		WHERE id IN (
			SELECT id FROM (
				SELECT id, row_number() OVER (PARTITION BY article_id, tag_id ORDER BY id) AS rn
				FROM t_article_tag
			) ranked WHERE rn > 1
		)`); err != nil {
		return fmt.Errorf("deduplicate article tags: %w", err)
	}

	for _, statement := range []string{
		`ALTER TABLE t_user_info ALTER COLUMN handle SET NOT NULL`,
		`ALTER TABLE t_category ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_series ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_tag ALTER COLUMN user_id SET NOT NULL`,
		`ALTER TABLE t_series DROP CONSTRAINT IF EXISTS t_series_series_name_key`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_user_info_handle_lower ON t_user_info (lower(handle))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_series_owner_name ON t_series (user_id, lower(btrim(series_name))) WHERE is_delete = 0`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_category_owner_name ON t_category (user_id, lower(btrim(category_name)))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_tag_owner_name ON t_tag (user_id, lower(btrim(tag_name)))`,
		`CREATE INDEX IF NOT EXISTS idx_article_public_owner ON t_article(status, moderation_status, user_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_talk_public_owner ON t_talk(status, moderation_status, user_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_series_public_owner ON t_series(status, moderation_status, user_id, id DESC)`,
	} {
		if _, err := session.Exec(statement); err != nil {
			return fmt.Errorf("finalize multi-user schema: %w", err)
		}
	}

	if _, err := session.Exec("INSERT INTO "+migrationTable+" (version, name) VALUES (?, ?)", 15, "multi-user-platform"); err != nil {
		return fmt.Errorf("record multi-user migration: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit multi-user migration: %w", err)
	}
	return nil
}
