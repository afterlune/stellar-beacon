package repository

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/url"
	"strings"
	"time"

	apperrors "benetnasch/app/domain/errors"
	"benetnasch/app/domain/port"

	"github.com/google/uuid"
	"xorm.io/xorm"
)

var _ port.DreamRepository = (*MyDreamRepository)(nil)

type MyDreamRepository struct {
	engine *xorm.Engine
}

func NewDreamRepository(engine *xorm.Engine) *MyDreamRepository {
	return &MyDreamRepository{engine: engine}
}

type dreamEntryRow struct {
	ID               string     `xorm:"id"`
	ReviewID         string     `xorm:"review_id"`
	Title            string     `xorm:"title"`
	Content          string     `xorm:"content"`
	ImagePrompt      string     `xorm:"image_prompt"`
	SourceArticleIDs string     `xorm:"source_article_ids"`
	Status           string     `xorm:"status"`
	ImageStatus      string     `xorm:"image_status"`
	ImageURL         string     `xorm:"image_url"`
	ImageError       string     `xorm:"image_error"`
	ImageLeaseOwner  string     `xorm:"image_lease_owner"`
	ImageLeaseUntil  *time.Time `xorm:"image_lease_until"`
	CreatedAt        time.Time  `xorm:"created_at"`
	UpdatedAt        time.Time  `xorm:"updated_at"`
}

const dreamEntryColumns = `id, review_id, title, content, image_prompt,
COALESCE(source_article_ids, '[]') AS source_article_ids, status,
COALESCE(image_status, 'pending') AS image_status, COALESCE(image_url, '') AS image_url,
COALESCE(image_error, '') AS image_error, COALESCE(image_lease_owner, '') AS image_lease_owner, image_lease_until,
created_at, updated_at`

func (r *MyDreamRepository) Create(ctx context.Context, input port.DreamEntry) error {
	entry, sourceIDs, err := normalizeDreamEntry(input)
	if err != nil {
		return apperrors.Invalid("dream.create", err.Error())
	}
	return repoTx(r.engine, ctx, "dream.create", func(session *xorm.Session) error {
		var existing dreamEntryRow
		found, err := session.SQL("SELECT "+dreamEntryColumns+" FROM t_dream_entry WHERE review_id = ?", entry.ReviewID).Get(&existing)
		if err != nil {
			return err
		}
		if found {
			current, decodeErr := existing.dreamEntry()
			if decodeErr != nil {
				return decodeErr
			}
			if current.ID == entry.ID && current.Title == entry.Title && current.Content == entry.Content && current.ImagePrompt == entry.ImagePrompt && sameIntSlice(current.SourceArticleIDs, entry.SourceArticleIDs) {
				return nil
			}
			return apperrors.Conflict("dream.create", "review already has another dream entry")
		}
		_, err = session.Exec(`
INSERT INTO t_dream_entry
    (id, review_id, title, content, image_prompt, source_article_ids, status, image_status, image_url, image_error, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			entry.ID,
			entry.ReviewID,
			entry.Title,
			entry.Content,
			entry.ImagePrompt,
			sourceIDs,
			string(entry.Status),
			string(entry.ImageStatus),
			entry.ImageURL,
			entry.ImageError,
			entry.CreatedAt,
			entry.UpdatedAt,
		)
		return err
	})
}

func (r *MyDreamRepository) GetByReview(ctx context.Context, reviewID string) (port.DreamEntry, error) {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" || len(reviewID) > 64 {
		return port.DreamEntry{}, apperrors.Invalid("dream.get", "review id is invalid")
	}
	session, err := r.session(ctx, "dream.get")
	if err != nil {
		return port.DreamEntry{}, err
	}
	defer session.Close()
	var row dreamEntryRow
	found, err := session.SQL("SELECT "+dreamEntryColumns+" FROM t_dream_entry WHERE review_id = ?", reviewID).Get(&row)
	if err != nil {
		return port.DreamEntry{}, apperrors.Unavailable("dream.get", err)
	}
	if !found {
		return port.DreamEntry{}, apperrors.NotFound("dream.get")
	}
	entry, err := row.dreamEntry()
	if err != nil {
		return port.DreamEntry{}, apperrors.Unavailable("dream.get.decode", err)
	}
	return entry, nil
}

func (r *MyDreamRepository) Approve(ctx context.Context, reviewID, content string, now time.Time) error {
	reviewID = strings.TrimSpace(reviewID)
	content = strings.TrimSpace(content)
	if reviewID == "" || len(reviewID) > 64 {
		return apperrors.Invalid("dream.approve", "review id is invalid")
	}
	if content == "" || len([]rune(content)) > 100_000 {
		return apperrors.Invalid("dream.approve", "approved dream content is invalid")
	}
	now = normalizeTime(now)
	session, err := r.session(ctx, "dream.approve")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_dream_entry
SET status = ?, content = ?, image_status = CASE WHEN image_status IN (?, ?) THEN image_status ELSE ? END,
    image_error = '', image_lease_owner = NULL, image_lease_until = NULL, updated_at = ?
WHERE review_id = ? AND status = ?`,
		string(port.DreamApproved),
		content,
		string(port.DreamImageReady),
		string(port.DreamImagePlaceholder),
		string(port.DreamImagePending),
		now,
		reviewID,
		string(port.DreamPendingReview),
	)
	if err != nil {
		return apperrors.Unavailable("dream.approve", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("dream.approve.rows", err)
	}
	if affected == 1 {
		return nil
	}
	entry, getErr := r.GetByReview(ctx, reviewID)
	if getErr != nil {
		return getErr
	}
	if entry.Status == port.DreamApproved {
		return nil
	}
	return apperrors.Conflict("dream.approve", "dream entry is not pending review")
}

func (r *MyDreamRepository) SyncReviewStatus(ctx context.Context, reviewID string, status port.DreamStatus, now time.Time) error {
	reviewID = strings.TrimSpace(reviewID)
	if reviewID == "" || len(reviewID) > 64 {
		return apperrors.Invalid("dream.sync_status", "review id is invalid")
	}
	if status != port.DreamRejected && status != port.DreamExpired {
		return apperrors.Invalid("dream.sync_status", "dream status can only be rejected or expired")
	}
	now = normalizeTime(now)
	session, err := r.session(ctx, "dream.sync_status")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_dream_entry
SET status = ?, image_lease_owner = NULL, image_lease_until = NULL, updated_at = ?
WHERE review_id = ? AND status = ?`, string(status), now, reviewID, string(port.DreamPendingReview))
	if err != nil {
		return apperrors.Unavailable("dream.sync_status", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("dream.sync_status.rows", err)
	}
	if affected == 1 {
		return nil
	}
	entry, getErr := r.GetByReview(ctx, reviewID)
	if getErr != nil {
		return getErr
	}
	if entry.Status == status {
		return nil
	}
	return apperrors.Conflict("dream.sync_status", "dream entry is no longer pending review")
}

func (r *MyDreamRepository) ListPublic(ctx context.Context, current, size int) ([]port.DreamEntry, int, error) {
	current, size, err := normalizeDreamPage(current, size)
	if err != nil {
		return nil, 0, apperrors.Invalid("dream.list", err.Error())
	}
	session, err := r.session(ctx, "dream.list")
	if err != nil {
		return nil, 0, err
	}
	defer session.Close()
	var countRow struct {
		Count int `xorm:"count"`
	}
	found, err := session.SQL("SELECT COUNT(1) AS count FROM t_dream_entry WHERE status = ?", string(port.DreamApproved)).Get(&countRow)
	if err != nil {
		return nil, 0, apperrors.Unavailable("dream.list.count", err)
	}
	if !found {
		return []port.DreamEntry{}, 0, nil
	}
	var rows []dreamEntryRow
	if err := session.SQL("SELECT "+dreamEntryColumns+" FROM t_dream_entry WHERE status = ? ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?", string(port.DreamApproved), size, (current-1)*size).Find(&rows); err != nil {
		return nil, 0, apperrors.Unavailable("dream.list", err)
	}
	entries := make([]port.DreamEntry, 0, len(rows))
	for _, row := range rows {
		entry, decodeErr := row.dreamEntry()
		if decodeErr != nil {
			return nil, 0, apperrors.Unavailable("dream.list.decode", decodeErr)
		}
		entries = append(entries, entry)
	}
	return entries, countRow.Count, nil
}

func (r *MyDreamRepository) ListPendingImages(ctx context.Context, limit int, now time.Time) ([]port.DreamEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	now = normalizeTime(now)
	session, err := r.session(ctx, "dream.list_images")
	if err != nil {
		return nil, err
	}
	defer session.Close()
	var rows []dreamEntryRow
	if err := session.SQL("SELECT "+dreamEntryColumns+" FROM t_dream_entry WHERE status = ? AND image_status IN (?, ?) AND (image_lease_until IS NULL OR image_lease_until <= ?) ORDER BY updated_at ASC, id ASC LIMIT ?", string(port.DreamApproved), string(port.DreamImagePending), string(port.DreamImageFailed), now, limit).Find(&rows); err != nil {
		return nil, apperrors.Unavailable("dream.list_images", err)
	}
	entries := make([]port.DreamEntry, 0, len(rows))
	for _, row := range rows {
		entry, decodeErr := row.dreamEntry()
		if decodeErr != nil {
			return nil, apperrors.Unavailable("dream.list_images.decode", decodeErr)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (r *MyDreamRepository) ClaimImage(ctx context.Context, id, worker string, now time.Time, lease time.Duration) (entry port.DreamEntry, claimed bool, err error) {
	id = strings.TrimSpace(id)
	worker = strings.TrimSpace(worker)
	if id == "" || len(id) > 64 || worker == "" || len(worker) > 128 || strings.ContainsAny(worker, "\r\n\x00") {
		return port.DreamEntry{}, false, apperrors.Invalid("dream.claim_image", "image claim identity is invalid")
	}
	if lease <= 0 || lease > 24*time.Hour {
		return port.DreamEntry{}, false, apperrors.Invalid("dream.claim_image", "image lease is invalid")
	}
	now = normalizeTime(now)
	err = repoTx(r.engine, ctx, "dream.claim_image", func(session *xorm.Session) error {
		var row dreamEntryRow
		found, getErr := session.SQL(`
UPDATE t_dream_entry
SET image_status = ?, image_lease_owner = ?, image_lease_until = ?, updated_at = ?
WHERE id = ? AND status = ? AND image_status IN (?, ?)
  AND (image_lease_until IS NULL OR image_lease_until <= ?)
RETURNING `+dreamEntryColumns,
			string(port.DreamImageProcessing), worker, now.Add(lease), now, id,
			string(port.DreamApproved), string(port.DreamImagePending), string(port.DreamImageFailed), now,
		).Get(&row)
		if getErr != nil {
			return getErr
		}
		if !found {
			return nil
		}
		decoded, decodeErr := row.dreamEntry()
		if decodeErr != nil {
			return decodeErr
		}
		entry = decoded
		claimed = true
		return nil
	})
	if err != nil {
		return port.DreamEntry{}, false, err
	}
	return entry, claimed, nil
}

func (r *MyDreamRepository) CompleteImage(ctx context.Context, id, worker string, status port.DreamImageStatus, imageURL, imageError string, now time.Time) error {
	id = strings.TrimSpace(id)
	worker = strings.TrimSpace(worker)
	imageURL = strings.TrimSpace(imageURL)
	imageError = strings.TrimSpace(imageError)
	if id == "" || len(id) > 64 || worker == "" || len(worker) > 128 || strings.ContainsAny(worker, "\r\n\x00") {
		return apperrors.Invalid("dream.complete_image", "image completion identity is invalid")
	}
	if err := validateDreamImageResult(status, imageURL, imageError); err != nil {
		return apperrors.Invalid("dream.complete_image", err.Error())
	}
	now = normalizeTime(now)
	session, err := r.session(ctx, "dream.complete_image")
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Exec(`
UPDATE t_dream_entry
SET image_status = ?, image_url = ?, image_error = ?, image_lease_owner = NULL,
    image_lease_until = NULL, updated_at = ?
WHERE id = ? AND status = ? AND image_status = ? AND image_lease_owner = ?`,
		string(status), imageURL, imageError, now, id, string(port.DreamApproved), string(port.DreamImageProcessing), worker)
	if err != nil {
		return apperrors.Unavailable("dream.complete_image", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Unavailable("dream.complete_image.rows", err)
	}
	if affected != 1 {
		return apperrors.Conflict("dream.complete_image", "image claim is no longer owned by the worker")
	}
	return nil
}

func (r *MyDreamRepository) session(ctx context.Context, operation string) (*xorm.Session, error) {
	if r == nil {
		return nil, apperrors.Unavailable(operation+".database", nil)
	}
	return repoSession(r.engine, ctx, operation)
}

func normalizeDreamEntry(input port.DreamEntry) (port.DreamEntry, string, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.ReviewID = strings.TrimSpace(input.ReviewID)
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	input.ImagePrompt = strings.TrimSpace(input.ImagePrompt)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.ImageError = strings.TrimSpace(input.ImageError)
	input.ImageLeaseOwner = strings.TrimSpace(input.ImageLeaseOwner)
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Status == "" {
		input.Status = port.DreamPendingReview
	}
	if input.ImageStatus == "" {
		input.ImageStatus = port.DreamImagePending
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now().UTC()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	input.CreatedAt = input.CreatedAt.UTC()
	input.UpdatedAt = input.UpdatedAt.UTC()
	if input.ID == "" || len(input.ID) > 64 || input.ReviewID == "" || len(input.ReviewID) > 64 || input.Title == "" || len([]rune(input.Title)) > 255 || input.Content == "" || len([]rune(input.Content)) > 100_000 || len([]rune(input.ImagePrompt)) > 10_000 || len(input.ImageURL) > 2048 || len(input.ImageError) > 255 || len(input.ImageLeaseOwner) > 128 || strings.ContainsAny(input.ID+input.ReviewID+input.ImageLeaseOwner, "\r\n\x00") {
		return port.DreamEntry{}, "", stderrors.New("dream entry identity or content is invalid")
	}
	if !validDreamStatus(input.Status) || !validDreamImageStatus(input.ImageStatus) {
		return port.DreamEntry{}, "", stderrors.New("dream entry status is invalid")
	}
	if len(input.SourceArticleIDs) == 0 || len(input.SourceArticleIDs) > port.MaxDreamSeedArticles {
		return port.DreamEntry{}, "", stderrors.New("dream source article count is invalid")
	}
	seen := make(map[int]struct{}, len(input.SourceArticleIDs))
	for _, articleID := range input.SourceArticleIDs {
		if articleID <= 0 {
			return port.DreamEntry{}, "", stderrors.New("dream source article id must be positive")
		}
		if _, exists := seen[articleID]; exists {
			return port.DreamEntry{}, "", stderrors.New("dream source articles must be unique")
		}
		seen[articleID] = struct{}{}
	}
	data, err := json.Marshal(input.SourceArticleIDs)
	if err != nil {
		return port.DreamEntry{}, "", err
	}
	return input, string(data), nil
}

func normalizeDreamPage(current, size int) (int, int, error) {
	if current <= 0 {
		current = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return current, size, nil
}

func validDreamStatus(status port.DreamStatus) bool {
	switch status {
	case port.DreamPendingReview, port.DreamApproved, port.DreamRejected, port.DreamExpired:
		return true
	default:
		return false
	}
}

func validDreamImageStatus(status port.DreamImageStatus) bool {
	switch status {
	case port.DreamImagePending, port.DreamImageProcessing, port.DreamImageReady, port.DreamImagePlaceholder, port.DreamImageFailed:
		return true
	default:
		return false
	}
}

func validateDreamImageResult(status port.DreamImageStatus, imageURL, imageError string) error {
	if !validDreamImageStatus(status) || status == port.DreamImagePending || status == port.DreamImageProcessing {
		return stderrors.New("dream image completion status is invalid")
	}
	if len(imageError) > 255 || strings.ContainsAny(imageError, "\r\n\x00") {
		return stderrors.New("dream image error is invalid")
	}
	switch status {
	case port.DreamImageReady:
		parsed, err := url.ParseRequestURI(imageURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return stderrors.New("ready dream image url must be http or https")
		}
		if imageError != "" {
			return stderrors.New("ready dream image cannot contain an error")
		}
	case port.DreamImagePlaceholder:
		if imageURL == "" || imageError == "" {
			return stderrors.New("placeholder dream image requires url and error")
		}
	case port.DreamImageFailed:
		if imageError == "" {
			return stderrors.New("failed dream image requires an error")
		}
		if imageURL != "" {
			return stderrors.New("failed dream image cannot contain a url")
		}
	}
	return nil
}

func (row dreamEntryRow) dreamEntry() (port.DreamEntry, error) {
	ids, err := decodeDreamArticleIDs(row.SourceArticleIDs)
	if err != nil {
		return port.DreamEntry{}, err
	}
	entry := port.DreamEntry{
		ID:               row.ID,
		ReviewID:         row.ReviewID,
		Title:            row.Title,
		Content:          row.Content,
		ImagePrompt:      row.ImagePrompt,
		SourceArticleIDs: ids,
		Status:           port.DreamStatus(row.Status),
		ImageStatus:      port.DreamImageStatus(row.ImageStatus),
		ImageURL:         row.ImageURL,
		ImageError:       row.ImageError,
		ImageLeaseOwner:  row.ImageLeaseOwner,
		CreatedAt:        row.CreatedAt.UTC(),
		UpdatedAt:        row.UpdatedAt.UTC(),
	}
	if row.ImageLeaseUntil != nil {
		entry.ImageLeaseUntil = row.ImageLeaseUntil.UTC()
	}
	return entry, nil
}

func sameIntSlice(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func decodeDreamArticleIDs(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, stderrors.New("dream source article ids are empty")
	}
	var ids []int
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	if len(ids) == 0 || len(ids) > port.MaxDreamSeedArticles {
		return nil, stderrors.New("dream source article count is invalid")
	}
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, stderrors.New("dream source article id must be positive")
		}
		if _, exists := seen[id]; exists {
			return nil, stderrors.New("dream source articles must be unique")
		}
		seen[id] = struct{}{}
	}
	return ids, nil
}
