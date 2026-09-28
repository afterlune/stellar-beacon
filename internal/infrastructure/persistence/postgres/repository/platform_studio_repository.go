package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"xorm.io/xorm"
)

func (r *MyPlatformRepo) StudioDashboard(ctx context.Context, userID int) (port.StudioDashboard, error) {
	session, err := repoSession(r.engine, ctx, "platform.studio.dashboard")
	if err != nil {
		return port.StudioDashboard{}, err
	}
	var dashboard port.StudioDashboard
	queries := []struct {
		sql    string
		target *int
	}{
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0`, &dashboard.ArticleCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 3`, &dashboard.DraftCount},
		{`SELECT count(1) FROM t_article WHERE user_id = ? AND is_delete = 0 AND status = 2`, &dashboard.PrivateCount},
		{`SELECT count(1) FROM t_talk WHERE user_id = ?`, &dashboard.TalkCount},
		{`SELECT count(1) FROM t_series WHERE user_id = ? AND is_delete = 0`, &dashboard.SeriesCount},
		{`SELECT count(1) FROM t_article_reaction WHERE user_info_id = ? AND reaction = 'favorite'`, &dashboard.FavoriteCount},
		{`SELECT count(1) FROM t_user_follow follow JOIN t_user_info follower ON follower.id = follow.follower_id AND follower.is_disable = 0 WHERE follow.author_id = ?`, &dashboard.FollowerCount},
		{`SELECT count(1) FROM t_user_follow follow JOIN t_user_info author ON author.id = follow.author_id AND author.is_disable = 0 WHERE follow.follower_id = ?`, &dashboard.FollowingCount},
	}
	for _, item := range queries {
		if _, err := session.SQL(item.sql, userID).Get(item.target); err != nil {
			return port.StudioDashboard{}, apperrors.Unavailable("platform.studio.dashboard", err)
		}
	}
	activation, err := loadStudioActivation(session, userID)
	if err != nil {
		return port.StudioDashboard{}, err
	}
	dashboard.Activation = activation
	return dashboard, nil
}

type studioActivationRow struct {
	UserId              int        `xorm:"user_id"`
	Collapsed           int        `xorm:"collapsed"`
	StartedAt           *time.Time `xorm:"started_at"`
	IdentityCompletedAt *time.Time `xorm:"identity_completed_at"`
	ContentCompletedAt  *time.Time `xorm:"content_completed_at"`
	ProfileVisitedAt    *time.Time `xorm:"profile_visited_at"`
	CompletedAt         *time.Time `xorm:"completed_at"`
}

func emptyStudioActivation() port.StudioActivation {
	return port.StudioActivation{}
}

func activationTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func studioActivationFromRow(row studioActivationRow) port.StudioActivation {
	return port.StudioActivation{
		StartedAt:           activationTime(row.StartedAt),
		Collapsed:           row.Collapsed != 0,
		IdentityCompletedAt: activationTime(row.IdentityCompletedAt),
		ContentCompletedAt:  activationTime(row.ContentCompletedAt),
		ProfileVisitedAt:    activationTime(row.ProfileVisitedAt),
		CompletedAt:         activationTime(row.CompletedAt),
	}
}

func activationTimeValue(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return *value
}

func loadStudioActivation(session *xorm.Session, userID int) (port.StudioActivation, error) {
	var row studioActivationRow
	found, err := session.SQL(`
		SELECT user_id, collapsed, started_at, identity_completed_at,
		       content_completed_at, profile_visited_at, completed_at
		FROM t_studio_activation
		WHERE user_id = ?
	`, userID).Get(&row)
	if err != nil {
		return emptyStudioActivation(), apperrors.Unavailable("platform.studio.activation.get", err)
	}
	if !found {
		return emptyStudioActivation(), nil
	}
	return studioActivationFromRow(row), nil
}

func (r *MyPlatformRepo) SyncStudioActivation(ctx context.Context, userID int, update port.StudioActivationUpdate) (port.StudioActivation, error) {
	var result port.StudioActivation
	err := repoTx(r.engine, ctx, "platform.studio.activation.sync", func(session *xorm.Session) error {
		collapsed := 0
		if update.Collapsed {
			collapsed = 1
		}
		if _, err := session.Exec(`
			INSERT INTO t_studio_activation (user_id, collapsed)
			VALUES (?, ?)
			ON CONFLICT (user_id) DO NOTHING
		`, userID, collapsed); err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		var row studioActivationRow
		found, err := session.SQL(`
			SELECT user_id, collapsed, started_at, identity_completed_at,
			       content_completed_at, profile_visited_at, completed_at
			FROM t_studio_activation
			WHERE user_id = ?
			FOR UPDATE
		`, userID).Get(&row)
		if err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		if !found {
			return apperrors.Unavailable("platform.studio.activation.sync", nil)
		}
		now := time.Now().UTC()
		if update.Started && row.StartedAt == nil {
			row.StartedAt = &now
		}
		if update.IdentityComplete && row.IdentityCompletedAt == nil {
			row.IdentityCompletedAt = &now
		}
		if update.ContentComplete && row.ContentCompletedAt == nil {
			row.ContentCompletedAt = &now
		}
		if update.ProfileVisited && row.ProfileVisitedAt == nil {
			row.ProfileVisitedAt = &now
		}
		if (update.Completed || (update.IdentityComplete && update.ContentComplete && update.ProfileVisited)) && row.CompletedAt == nil {
			row.CompletedAt = &now
		}
		row.Collapsed = collapsed
		if _, err := session.Exec(`
			UPDATE t_studio_activation
			SET collapsed = ?, started_at = ?, identity_completed_at = ?,
			    content_completed_at = ?, profile_visited_at = ?, completed_at = ?,
			    update_time = CURRENT_TIMESTAMP
			WHERE user_id = ?
		`, row.Collapsed, activationTimeValue(row.StartedAt), activationTimeValue(row.IdentityCompletedAt),
			activationTimeValue(row.ContentCompletedAt), activationTimeValue(row.ProfileVisitedAt),
			activationTimeValue(row.CompletedAt), userID); err != nil {
			return apperrors.Unavailable("platform.studio.activation.sync", err)
		}
		result = studioActivationFromRow(row)
		return nil
	})
	return result, err
}

func (r *MyPlatformRepo) CreateDueStudioActivationReminders(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	session, err := repoSession(r.engine, ctx, "platform.studio.activation_reminders")
	if err != nil {
		return 0, err
	}
	result, err := session.Exec(`
		INSERT INTO t_user_notification (
			recipient_id, actor_id, type, content_type, content_id, comment_id,
			dedupe_key, title, excerpt, action_url
		)
		SELECT
			activation.user_id,
			NULL,
			'studio_activation',
			'studio',
			0,
			0,
			'studio-activation:' || CASE WHEN activation.started_at <= ? THEN '72h' ELSE '24h' END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '完成公开身份'
				WHEN activation.content_completed_at IS NULL THEN '写下第一条内容'
				ELSE '预览你的公开主页'
			END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '补齐 Handle、头像、昵称和简介，让公开主页可以访问。'
				WHEN activation.content_completed_at IS NULL THEN '写一篇文章或发布一条随想，完成你的第一次表达。'
				ELSE '检查主页在公共空间里的最终呈现，完成创作者激活。'
			END,
			CASE
				WHEN activation.identity_completed_at IS NULL THEN '/studio/profile'
				WHEN activation.content_completed_at IS NULL THEN '/studio/dashboard#activation'
				WHEN recipient.handle <> '' THEN '/u/' || recipient.handle
				ELSE '/studio/profile'
			END
		FROM t_studio_activation activation
		JOIN t_user_info recipient ON recipient.id = activation.user_id
			AND recipient.is_disable = 0 AND recipient.notify_studio_activation = 1
		WHERE activation.started_at IS NOT NULL
		  AND activation.completed_at IS NULL
		  AND activation.started_at <= ?
		  AND (activation.identity_completed_at IS NULL
		       OR activation.content_completed_at IS NULL
		       OR activation.profile_visited_at IS NULL)
		ORDER BY activation.started_at ASC
		LIMIT ?
		ON CONFLICT (recipient_id, dedupe_key) DO NOTHING
	`, now.Add(-72*time.Hour), now.Add(-24*time.Hour), limit)
	if err != nil {
		return 0, apperrors.Unavailable("platform.studio.activation_reminders", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, apperrors.Unavailable("platform.studio.activation_reminders", err)
	}
	return int(affected), nil
}

func (r *MyPlatformRepo) GetStudioProfile(ctx context.Context, userID int) (port.StudioProfile, error) {
	session, err := repoSession(r.engine, ctx, "platform.profile.get")
	if err != nil {
		return port.StudioProfile{}, err
	}
	var user entity.TUserInfo
	found, err := session.ID(userID).Get(&user)
	if err != nil {
		return port.StudioProfile{}, apperrors.Unavailable("platform.profile.get", err)
	}
	if !found {
		return port.StudioProfile{}, apperrors.NotFound("platform.profile.get")
	}
	return port.StudioProfile{
		Handle: user.Handle, Nickname: user.Nickname, Avatar: user.Avatar,
		Intro: user.Intro, Website: user.Website, About: user.About,
		Links: decodeProfileLinks(user.ProfileLinksJSON),
	}, nil
}

func (r *MyPlatformRepo) UpdateAuthorProfile(ctx context.Context, userID int, profile port.StudioProfile) error {
	profile.Handle = strings.ToLower(strings.TrimSpace(profile.Handle))
	profile.Nickname = strings.TrimSpace(profile.Nickname)
	if !validPublicHandle(profile.Handle) {
		return apperrors.Invalid("platform.profile.handle", "handle is invalid")
	}
	if profile.Nickname == "" {
		return apperrors.Invalid("platform.profile.nickname", "nickname is required")
	}
	linksJSON, err := json.Marshal(profile.Links)
	if err != nil {
		return apperrors.Invalid("platform.profile.links", "links could not be encoded")
	}
	return ormInit.WithEngineTx(r.engine, ctx, func(session *xorm.Session) error {
		var existing entity.TUserInfo
		found, err := session.Where("lower(handle) = lower(?) AND id <> ?", profile.Handle, userID).Get(&existing)
		if err != nil {
			return apperrors.Unavailable("platform.profile.handle.unique", err)
		}
		if found {
			return apperrors.Conflict("platform.profile.handle", "handle already exists")
		}
		affected, err := session.ID(userID).Cols("handle", "nickname", "intro", "website", "about", "profile_links_json").Update(&entity.TUserInfo{
			Handle: profile.Handle, Nickname: profile.Nickname, Intro: profile.Intro,
			Website: profile.Website, About: profile.About, ProfileLinksJSON: string(linksJSON),
		})
		if err != nil {
			return apperrors.Unavailable("platform.profile.update", err)
		}
		if affected == 0 {
			return apperrors.NotFound("platform.profile.update")
		}
		return nil
	})
}

func validPublicHandle(value string) bool {
	if len(value) < 3 || len(value) > 40 {
		return false
	}
	for index, r := range value {
		if index == 0 && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
