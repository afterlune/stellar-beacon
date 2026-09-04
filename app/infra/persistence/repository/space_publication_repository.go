package repository

import (
	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"
	"benetnasch/app/infra/persistence/pgsql"
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.SpacePublicationRepository = (*MySpacePublicationRepository)(nil)

type MySpacePublicationRepository struct {
	engine *xorm.Engine
}

func NewSpacePublicationRepository(engine *xorm.Engine) *MySpacePublicationRepository {
	return &MySpacePublicationRepository{engine: engine}
}

type spacePublicationRow struct {
	ID              string    `xorm:"id"`
	AgentID         string    `xorm:"agent_id"`
	ContentType     string    `xorm:"content_type"`
	Title           string    `xorm:"title"`
	Body            string    `xorm:"body"`
	MediaURL        string    `xorm:"media_url"`
	SourceSessionID string    `xorm:"source_session_id"`
	SourceRunID     string    `xorm:"source_run_id"`
	IdempotencyKey  string    `xorm:"idempotency_key"`
	RequestDigest   string    `xorm:"request_digest"`
	PublishedAt     time.Time `xorm:"published_at"`
	CreatedAt       time.Time `xorm:"created_at"`
}

const spacePublicationColumns = `id, agent_id, content_type, title, body, media_url,
source_session_id, source_run_id, idempotency_key, request_digest, published_at, created_at`

func (r *MySpacePublicationRepository) Create(ctx context.Context, input port.SpacePublication) (publication port.SpacePublication, existing bool, err error) {
	if r == nil {
		return port.SpacePublication{}, false, apperrors.Unavailable("space.publication.create.database", nil)
	}
	if strings.TrimSpace(input.ID) == "" {
		input.ID = "publication-" + uuid.NewString()
	}
	if input.PublishedAt.IsZero() {
		input.PublishedAt = time.Now().UTC()
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = input.PublishedAt
	}
	if strings.TrimSpace(input.RequestDigest) == "" {
		return port.SpacePublication{}, false, apperrors.Invalid("space.publication.create", "request digest is required")
	}
	if _, err := port.NormalizeSpacePublicationType(string(input.Type)); err != nil {
		return port.SpacePublication{}, false, apperrors.Invalid("space.publication.create", err.Error())
	}

	err = repoTx(r.engine, ctx, "space.publication.create", func(session *xorm.Session) error {
		result, execErr := session.Exec(`
INSERT INTO t_space_publication
    (id, agent_id, content_type, title, body, media_url, source_session_id,
     source_run_id, idempotency_key, request_digest, published_at, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (idempotency_key) DO NOTHING`,
			input.ID,
			input.AgentID,
			string(input.Type),
			input.Title,
			input.Body,
			input.MediaURL,
			input.SourceSessionID,
			input.SourceRunID,
			input.IdempotencyKey,
			input.RequestDigest,
			input.PublishedAt,
			input.CreatedAt,
		)
		if execErr != nil {
			return execErr
		}
		affected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if affected == 1 {
			publication = input
			return nil
		}

		var row spacePublicationRow
		found, getErr := session.SQL(
			"SELECT "+spacePublicationColumns+" FROM t_space_publication WHERE idempotency_key = ?",
			input.IdempotencyKey,
		).Get(&row)
		if getErr != nil {
			return getErr
		}
		if !found {
			return apperrors.Conflict("space.publication.create", "publication idempotency record is unavailable")
		}
		if row.RequestDigest != input.RequestDigest {
			return apperrors.Conflict("space.publication.create", "idempotency key was already used for different content")
		}
		decoded, decodeErr := row.publication()
		if decodeErr != nil {
			return decodeErr
		}
		publication = decoded
		existing = true
		return nil
	})
	if err != nil {
		return port.SpacePublication{}, false, err
	}
	return publication, existing, nil
}

func (r *MySpacePublicationRepository) GetPublic(ctx context.Context, contentType port.SpaceContentType, id string) (port.SpacePublication, error) {
	if r == nil {
		return port.SpacePublication{}, apperrors.Unavailable("space.publication.get.database", nil)
	}
	normalizedType, err := port.NormalizeSpacePublicationType(string(contentType))
	if err != nil {
		return port.SpacePublication{}, apperrors.Invalid("space.publication.get", err.Error())
	}
	id = strings.TrimSpace(id)
	if id == "" || len(id) > port.MaxSpaceContentID || strings.ContainsAny(id, "\r\n\x00") {
		return port.SpacePublication{}, apperrors.Invalid("space.publication.get", "publication id is invalid")
	}
	session, err := repoSession(r.engine, ctx, "space.publication.get")
	if err != nil {
		return port.SpacePublication{}, err
	}
	defer session.Close()
	var row spacePublicationRow
	found, err := session.SQL(
		"SELECT "+spacePublicationColumns+" FROM t_space_publication WHERE content_type = ? AND id = ?",
		string(normalizedType), id,
	).Get(&row)
	if err != nil {
		return port.SpacePublication{}, apperrors.Unavailable("space.publication.get", err)
	}
	if !found {
		return port.SpacePublication{}, apperrors.NotFound("space.publication.get")
	}
	return row.publication()
}

func (r *MySpacePublicationRepository) ListPublic(ctx context.Context, query port.SpacePublicationQuery) ([]port.SpacePublication, int, error) {
	if r == nil {
		return nil, 0, apperrors.Unavailable("space.publication.list.database", nil)
	}
	normalized, err := query.Normalize()
	if err != nil {
		return nil, 0, apperrors.Invalid("space.publication.list", err.Error())
	}
	where := " WHERE 1 = 1"
	args := make([]interface{}, 0, 2)
	if normalized.Type != "" {
		where += " AND content_type = ?"
		args = append(args, normalized.Type)
	}
	if normalized.Query != "" {
		where += " AND (LOWER(title) LIKE LOWER(?) ESCAPE '\\' OR LOWER(body) LIKE LOWER(?) ESCAPE '\\' OR LOWER(media_url) LIKE LOWER(?) ESCAPE '\\')"
		pattern := pgsql.ContainsPattern(normalized.Query)
		args = append(args, pattern, pattern, pattern)
	}
	session, err := repoSession(r.engine, ctx, "space.publication.list")
	if err != nil {
		return nil, 0, err
	}
	defer session.Close()
	var count int
	if _, err := session.SQL("SELECT COUNT(1) FROM t_space_publication"+where, args...).Get(&count); err != nil {
		return nil, 0, apperrors.Unavailable("space.publication.list.count", err)
	}
	listArgs := append(append([]interface{}{}, args...), normalized.Limit)
	var rows []spacePublicationRow
	if err := session.SQL(
		"SELECT "+spacePublicationColumns+" FROM t_space_publication"+where+" ORDER BY published_at DESC, id DESC LIMIT ?",
		listArgs...,
	).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("space.publication.list", err)
	}
	publications := make([]port.SpacePublication, 0, len(rows))
	for _, row := range rows {
		publication, decodeErr := row.publication()
		if decodeErr != nil {
			return nil, 0, apperrors.Unavailable("space.publication.list.decode", decodeErr)
		}
		publications = append(publications, publication)
	}
	return publications, count, nil
}

func (r spacePublicationRow) publication() (port.SpacePublication, error) {
	contentType, err := port.NormalizeSpacePublicationType(r.ContentType)
	if err != nil {
		return port.SpacePublication{}, err
	}
	return port.SpacePublication{
		ID:              r.ID,
		AgentID:         r.AgentID,
		Type:            contentType,
		Title:           r.Title,
		Body:            r.Body,
		MediaURL:        r.MediaURL,
		SourceSessionID: r.SourceSessionID,
		SourceRunID:     r.SourceRunID,
		IdempotencyKey:  r.IdempotencyKey,
		RequestDigest:   r.RequestDigest,
		PublishedAt:     r.PublishedAt.UTC(),
		CreatedAt:       r.CreatedAt.UTC(),
	}, nil
}
