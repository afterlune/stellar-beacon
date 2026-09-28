package repository

import (
	"context"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"xorm.io/xorm"
)

var _ port.NewsletterRepository = (*MyNewsletterRepo)(nil)

type MyNewsletterRepo struct{ engine *xorm.Engine }

func NewNewsletterRepo(engine *xorm.Engine) *MyNewsletterRepo {
	return &MyNewsletterRepo{engine: engine}
}

func (r *MyNewsletterRepo) session(ctx context.Context, operation string) (*xorm.Session, error) {
	return repoSession(r.engine, ctx, operation)
}

func (r *MyNewsletterRepo) FindSubscriberByEmail(ctx context.Context, email string) (entity.TNewsletterSubscriber, error) {
	session, err := r.session(ctx, "newsletter.find_email")
	if err != nil {
		return entity.TNewsletterSubscriber{}, err
	}
	var subscriber entity.TNewsletterSubscriber
	found, err := session.Where("email = ?", email).Get(&subscriber)
	if err != nil {
		return entity.TNewsletterSubscriber{}, apperrors.Unavailable("newsletter.find_email", err)
	}
	if !found {
		return entity.TNewsletterSubscriber{}, apperrors.NotFound("newsletter.find_email")
	}
	return subscriber, nil
}

func (r *MyNewsletterRepo) FindSubscriberByID(ctx context.Context, id int) (entity.TNewsletterSubscriber, error) {
	session, err := r.session(ctx, "newsletter.find_id")
	if err != nil {
		return entity.TNewsletterSubscriber{}, err
	}
	var subscriber entity.TNewsletterSubscriber
	found, err := session.ID(id).Get(&subscriber)
	if err != nil {
		return entity.TNewsletterSubscriber{}, apperrors.Unavailable("newsletter.find_id", err)
	}
	if !found {
		return entity.TNewsletterSubscriber{}, apperrors.NotFound("newsletter.find_id")
	}
	return subscriber, nil
}

func (r *MyNewsletterRepo) FindSubscriberByToken(ctx context.Context, field, tokenHash string) (entity.TNewsletterSubscriber, error) {
	if field != "confirm_token_hash" && field != "unsubscribe_token_hash" {
		return entity.TNewsletterSubscriber{}, apperrors.Invalid("newsletter.token_field", "unsupported token field")
	}
	session, err := r.session(ctx, "newsletter.find_token")
	if err != nil {
		return entity.TNewsletterSubscriber{}, err
	}
	var subscriber entity.TNewsletterSubscriber
	found, err := session.Where(field+" = ?", tokenHash).Get(&subscriber)
	if err != nil {
		return entity.TNewsletterSubscriber{}, apperrors.Unavailable("newsletter.find_token", err)
	}
	if !found {
		return entity.TNewsletterSubscriber{}, apperrors.NotFound("newsletter.find_token")
	}
	return subscriber, nil
}

func (r *MyNewsletterRepo) SaveSubscriber(ctx context.Context, subscriber entity.TNewsletterSubscriber) error {
	return repoTx(r.engine, ctx, "newsletter.save", func(session *xorm.Session) error {
		if subscriber.Id == 0 {
			_, err := session.Exec(`
				INSERT INTO t_newsletter_subscriber
					(email, status, confirm_token_hash, confirm_token_expires_at, unsubscribe_token_hash, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, subscriber.Email, subscriber.Status, subscriber.ConfirmTokenHash, subscriber.ConfirmTokenExpiresAt, subscriber.UnsubscribeTokenHash)
			if err != nil {
				return apperrors.Unavailable("newsletter.insert", err)
			}
			return nil
		}
		if _, err := session.Exec(`
			UPDATE t_newsletter_subscriber
			SET email = ?, status = ?, confirm_token_hash = ?, confirm_token_expires_at = ?,
				unsubscribe_token_hash = ?, confirmed_at = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, subscriber.Email, subscriber.Status, subscriber.ConfirmTokenHash, subscriber.ConfirmTokenExpiresAt, subscriber.UnsubscribeTokenHash, subscriber.Id); err != nil {
			return apperrors.Unavailable("newsletter.update", err)
		}
		return nil
	})
}

func (r *MyNewsletterRepo) ActivateSubscriber(ctx context.Context, id int, confirmedAt time.Time) error {
	session, err := r.session(ctx, "newsletter.activate")
	if err != nil {
		return err
	}
	if _, err := session.Exec(`UPDATE t_newsletter_subscriber
		SET status = ?, confirmed_at = ?, confirm_token_hash = '', confirm_token_expires_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, port.NewsletterActive, confirmedAt, id); err != nil {
		return apperrors.Unavailable("newsletter.activate", err)
	}
	return nil
}

func (r *MyNewsletterRepo) UnsubscribeSubscriber(ctx context.Context, id int) error {
	session, err := r.session(ctx, "newsletter.unsubscribe")
	if err != nil {
		return err
	}
	if _, err := session.ID(id).Cols("status").Update(&entity.TNewsletterSubscriber{
		Id: id, Status: port.NewsletterUnsubscribed,
	}); err != nil {
		return apperrors.Unavailable("newsletter.unsubscribe", err)
	}
	return nil
}

func (r *MyNewsletterRepo) SetSubscriberStatus(ctx context.Context, id int, status string) error {
	if status != port.NewsletterPending && status != port.NewsletterActive && status != port.NewsletterUnsubscribed {
		return apperrors.Invalid("newsletter.status", "unsupported subscriber status")
	}
	session, err := r.session(ctx, "newsletter.status")
	if err != nil {
		return err
	}
	if _, err := session.ID(id).Cols("status").Update(&entity.TNewsletterSubscriber{Id: id, Status: status}); err != nil {
		return apperrors.Unavailable("newsletter.status", err)
	}
	return nil
}

func (r *MyNewsletterRepo) ListSubscribers(ctx context.Context, filter port.NewsletterFilter) ([]entity.TNewsletterSubscriber, int, error) {
	session, err := r.session(ctx, "newsletter.list_subscribers")
	if err != nil {
		return nil, 0, err
	}
	where, args := subscriberWhere(filter)
	var total int
	if _, err := session.SQL("SELECT COUNT(*) FROM t_newsletter_subscriber"+where, args...).Get(&total); err != nil {
		return nil, 0, apperrors.Unavailable("newsletter.count_subscribers", err)
	}
	limit, offset := page(filter.Current, filter.Size)
	var subscribers []entity.TNewsletterSubscriber
	if err := session.SQL("SELECT * FROM t_newsletter_subscriber"+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, limit, offset)...).Find(&subscribers); err != nil {
		return nil, 0, apperrors.Unavailable("newsletter.list_subscribers", err)
	}
	return subscribers, total, nil
}

func (r *MyNewsletterRepo) Stats(ctx context.Context) (port.NewsletterStats, error) {
	session, err := r.session(ctx, "newsletter.stats")
	if err != nil {
		return port.NewsletterStats{}, err
	}
	stats := port.NewsletterStats{}
	var subscribers []struct {
		Status string
		Count  int64
	}
	if err := session.SQL(`SELECT status, COUNT(*) AS count FROM t_newsletter_subscriber GROUP BY status`).Find(&subscribers); err != nil {
		return port.NewsletterStats{}, apperrors.Unavailable("newsletter.stats.subscribers", err)
	}
	for _, row := range subscribers {
		stats.Total += row.Count
		switch row.Status {
		case port.NewsletterActive:
			stats.Active = row.Count
		case port.NewsletterPending:
			stats.Pending = row.Count
		case port.NewsletterUnsubscribed:
			stats.Unsubscribed = row.Count
		}
	}
	var deliveries []struct {
		Status string
		Count  int64
	}
	if err := session.SQL(`SELECT status, COUNT(*) AS count FROM t_newsletter_delivery GROUP BY status`).Find(&deliveries); err != nil {
		return port.NewsletterStats{}, apperrors.Unavailable("newsletter.stats.deliveries", err)
	}
	for _, row := range deliveries {
		switch row.Status {
		case port.DeliveryQueued:
			stats.Queued = row.Count
		case port.DeliverySending:
			stats.Sending = row.Count
		case port.DeliverySent:
			stats.Sent = row.Count
		case port.DeliveryFailed:
			stats.Failed = row.Count
		}
	}
	return stats, nil
}

func (r *MyNewsletterRepo) DeliveryTrend(ctx context.Context, since time.Time, unit string) ([]port.NewsletterDeliveryTrend, error) {
	session, err := r.session(ctx, "newsletter.delivery_trend")
	if err != nil {
		return nil, err
	}
	format := "YYYY-MM-DD"
	if unit == "month" {
		format = "YYYY-MM"
	}
	query := `
		SELECT period, SUM(sent) AS sent, SUM(failed) AS failed
		FROM (
			SELECT to_char(sent_at, '` + format + `') AS period, COUNT(*) AS sent, 0::bigint AS failed
			FROM t_newsletter_delivery
			WHERE sent_at IS NOT NULL AND sent_at >= ?
			GROUP BY period
			UNION ALL
			SELECT to_char(updated_at, '` + format + `') AS period, 0::bigint AS sent, COUNT(*) AS failed
			FROM t_newsletter_delivery
			WHERE status = ? AND updated_at >= ?
			GROUP BY period
		) delivery_periods
		GROUP BY period
		ORDER BY period ASC`
	var result []port.NewsletterDeliveryTrend
	if err := session.SQL(query, since, port.DeliveryFailed, since).Find(&result); err != nil {
		return nil, apperrors.Unavailable("newsletter.delivery_trend", err)
	}
	return result, nil
}

func subscriberWhere(filter port.NewsletterFilter) (string, []interface{}) {
	where := " WHERE 1 = 1"
	args := make([]interface{}, 0, 2)
	if status := strings.TrimSpace(filter.Status); status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		where += " AND email ILIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	return where, args
}

func page(current, size int) (int, int) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return size, (current - 1) * size
}

func (r *MyNewsletterRepo) ListDeliveries(ctx context.Context, filter port.DeliveryFilter) ([]entity.TNewsletterDelivery, int, error) {
	session, err := r.session(ctx, "newsletter.list_deliveries")
	if err != nil {
		return nil, 0, err
	}
	where := " WHERE 1 = 1"
	args := make([]interface{}, 0, 1)
	if status := strings.TrimSpace(filter.Status); status != "" {
		where += " AND d.status = ?"
		args = append(args, status)
	}
	var total int
	if _, err := session.SQL("SELECT COUNT(*) FROM t_newsletter_delivery d"+where, args...).Get(&total); err != nil {
		return nil, 0, apperrors.Unavailable("newsletter.count_deliveries", err)
	}
	limit, offset := page(filter.Current, filter.Size)
	var deliveries []entity.TNewsletterDelivery
	query := `SELECT d.*, s.email AS subscriber_email, a.article_title AS article_title
		FROM t_newsletter_delivery d
		JOIN t_newsletter_subscriber s ON s.id = d.subscriber_id
		JOIN t_article a ON a.id = d.article_id` + where + " ORDER BY d.id DESC LIMIT ? OFFSET ?"
	if err := session.SQL(query, append(args, limit, offset)...).Find(&deliveries); err != nil {
		return nil, 0, apperrors.Unavailable("newsletter.list_deliveries", err)
	}
	return deliveries, total, nil
}

func (r *MyNewsletterRepo) CreateDeliveriesForArticle(ctx context.Context, articleID int) error {
	session, err := r.session(ctx, "newsletter.enqueue")
	if err != nil {
		return err
	}
	if _, err := session.Exec(`
		INSERT INTO t_newsletter_delivery (subscriber_id, article_id, status, attempts, created_at, updated_at)
		SELECT id, ?, ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM t_newsletter_subscriber
		WHERE status = ?
		ON CONFLICT (subscriber_id, article_id) DO NOTHING
	`, articleID, port.DeliveryQueued, port.NewsletterActive); err != nil {
		return apperrors.Unavailable("newsletter.enqueue", err)
	}
	return nil
}

func (r *MyNewsletterRepo) ClaimNextDelivery(ctx context.Context) (entity.TNewsletterDelivery, bool, error) {
	var delivery entity.TNewsletterDelivery
	found := false
	err := repoTx(r.engine, ctx, "newsletter.claim", func(session *xorm.Session) error {
		var candidate entity.TNewsletterDelivery
		ok, err := session.SQL(`
			SELECT * FROM t_newsletter_delivery
			WHERE (status = ? OR (status = ? AND updated_at < CURRENT_TIMESTAMP - INTERVAL '10 minutes'))
			  AND attempts < 5
			ORDER BY id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, port.DeliveryQueued, port.DeliverySending).Get(&candidate)
		if err != nil {
			return apperrors.Unavailable("newsletter.claim.select", err)
		}
		if !ok {
			return nil
		}
		if _, err := session.Exec("UPDATE t_newsletter_delivery SET status = ?, attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", port.DeliverySending, candidate.Id); err != nil {
			return apperrors.Unavailable("newsletter.claim.update", err)
		}
		candidate.Status = port.DeliverySending
		candidate.Attempts++
		delivery = candidate
		found = true
		return nil
	})
	return delivery, found, err
}

func (r *MyNewsletterRepo) MarkDeliverySent(ctx context.Context, id int, sentAt time.Time) error {
	session, err := r.session(ctx, "newsletter.sent")
	if err != nil {
		return err
	}
	if _, err := session.ID(id).Cols("status", "sent_at", "last_error").Update(&entity.TNewsletterDelivery{Id: id, Status: port.DeliverySent, SentAt: sentAt, LastError: ""}); err != nil {
		return apperrors.Unavailable("newsletter.sent", err)
	}
	return nil
}

func (r *MyNewsletterRepo) MarkDeliveryFailed(ctx context.Context, id int, message string, retry bool) error {
	status := port.DeliveryFailed
	if retry {
		status = port.DeliveryQueued
	}
	if len(message) > 1000 {
		message = message[:1000]
	}
	session, err := r.session(ctx, "newsletter.failed")
	if err != nil {
		return err
	}
	if _, err := session.ID(id).Cols("status", "last_error").Update(&entity.TNewsletterDelivery{Id: id, Status: status, LastError: message}); err != nil {
		return apperrors.Unavailable("newsletter.failed", err)
	}
	return nil
}

func (r *MyNewsletterRepo) RetryDelivery(ctx context.Context, id int) error {
	session, err := r.session(ctx, "newsletter.retry")
	if err != nil {
		return err
	}
	if _, err := session.ID(id).Cols("status", "attempts", "last_error").Update(&entity.TNewsletterDelivery{Id: id, Status: port.DeliveryQueued, Attempts: 0, LastError: ""}); err != nil {
		return apperrors.Unavailable("newsletter.retry", err)
	}
	return nil
}

func (r *MyNewsletterRepo) RetryFailedDeliveries(ctx context.Context) (int, error) {
	var count int64
	err := repoTx(r.engine, ctx, "newsletter.retry_failed", func(session *xorm.Session) error {
		result, err := session.Exec(`
			UPDATE t_newsletter_delivery
			SET status = ?, attempts = 0, last_error = '', updated_at = CURRENT_TIMESTAMP
			WHERE status = ?
		`, port.DeliveryQueued, port.DeliveryFailed)
		if err != nil {
			return apperrors.Unavailable("newsletter.retry_failed", err)
		}
		count, err = result.RowsAffected()
		return err
	})
	return int(count), err
}
