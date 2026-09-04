package repository

import (
	"context"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.VideoRepository = (*MyVideoRepository)(nil)

type MyVideoRepository struct {
	engine *xorm.Engine
}

func NewVideoRepository(engine *xorm.Engine) *MyVideoRepository {
	return &MyVideoRepository{engine: engine}
}

type videoRow struct {
	ID          string    `xorm:"id"`
	Title       string    `xorm:"title"`
	Description string    `xorm:"description"`
	Source      string    `xorm:"source"`
	URL         string    `xorm:"url"`
	EmbedURL    string    `xorm:"embed_url"`
	MIMEType    string    `xorm:"mime_type"`
	SizeBytes   int64     `xorm:"size_bytes"`
	Published   bool      `xorm:"published"`
	Deleted     bool      `xorm:"deleted"`
	CreatedAt   time.Time `xorm:"created_at"`
	UpdatedAt   time.Time `xorm:"updated_at"`
}

const videoColumns = `id, title, description, source, url, embed_url, mime_type,
size_bytes, published, deleted, created_at, updated_at`

func (r *MyVideoRepository) Create(ctx context.Context, input port.Video) error {
	if r == nil {
		return apperrors.Unavailable("video.create.database", nil)
	}
	if strings.TrimSpace(input.ID) == "" {
		input.ID = uuid.NewString()
	}
	if err := port.ValidateVideo(input); err != nil {
		return apperrors.Invalid("video.create", err.Error())
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	return repoTx(r.engine, ctx, "video.create", func(session *xorm.Session) error {
		_, err := session.Exec(`
INSERT INTO t_agent_video
    (id, title, description, source, url, embed_url, mime_type, size_bytes,
     published, deleted, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			input.ID,
			input.Title,
			input.Description,
			string(input.Source),
			input.URL,
			input.EmbedURL,
			input.MIMEType,
			input.SizeBytes,
			input.Published,
			input.Deleted,
			input.CreatedAt,
			input.UpdatedAt,
		)
		if err != nil && isUniqueViolation(err) {
			return apperrors.Conflict("video.create", "video already exists")
		}
		return err
	})
}

func (r *MyVideoRepository) ListPublic(ctx context.Context, current, size int) ([]port.Video, int, error) {
	if r == nil {
		return nil, 0, apperrors.Unavailable("video.list.database", nil)
	}
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > port.MaxVideoPageSize {
		size = port.MaxVideoPageSize
	}
	session, err := repoSession(r.engine, ctx, "video.list")
	if err != nil {
		return nil, 0, err
	}
	defer session.Close()
	var countRow struct {
		Count int `xorm:"count"`
	}
	found, err := session.SQL("SELECT COUNT(1) AS count FROM t_agent_video WHERE published = TRUE AND deleted = FALSE").Get(&countRow)
	if err != nil {
		return nil, 0, apperrors.Unavailable("video.list.count", err)
	}
	if !found {
		return []port.Video{}, 0, nil
	}
	var rows []videoRow
	if err := session.SQL("SELECT "+videoColumns+" FROM t_agent_video WHERE published = TRUE AND deleted = FALSE ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", size, (current-1)*size).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("video.list", err)
	}
	videos := make([]port.Video, 0, len(rows))
	for _, row := range rows {
		video, err := row.video()
		if err != nil {
			return nil, 0, apperrors.Unavailable("video.list.decode", err)
		}
		videos = append(videos, video)
	}
	return videos, countRow.Count, nil
}

func (r *MyVideoRepository) Delete(ctx context.Context, id string) error {
	if r == nil {
		return apperrors.Unavailable("video.delete.database", nil)
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxVideoIDLength || strings.ContainsAny(id, "\r\n\x00") {
		return apperrors.Invalid("video.delete", "video id is invalid")
	}
	session, err := repoSession(r.engine, ctx, "video.delete")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec("UPDATE t_agent_video SET deleted = TRUE, published = FALSE, updated_at = ? WHERE id = ? AND deleted = FALSE", time.Now().UTC(), id)
	if err != nil {
		return apperrors.Unavailable("video.delete", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("video.delete.rows", err)
	}
	if affected == 0 {
		// Soft deletion is idempotent. Do not reveal whether a missing ID ever
		// existed to an admin caller or turn retries into an error loop.
		return nil
	}
	return nil
}

func (r videoRow) video() (port.Video, error) {
	source, err := port.NormalizeVideoSource(r.Source)
	if err != nil {
		return port.Video{}, err
	}
	video := port.Video{
		ID:          r.ID,
		Title:       r.Title,
		Description: r.Description,
		Source:      source,
		URL:         r.URL,
		EmbedURL:    r.EmbedURL,
		MIMEType:    r.MIMEType,
		SizeBytes:   r.SizeBytes,
		Published:   r.Published,
		Deleted:     r.Deleted,
		CreatedAt:   r.CreatedAt.UTC(),
		UpdatedAt:   r.UpdatedAt.UTC(),
	}
	if err := port.ValidateVideo(video); err != nil {
		return port.Video{}, err
	}
	return video, nil
}
