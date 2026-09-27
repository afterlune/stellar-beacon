package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
)

func (c *MyCommentService) notifyComment(ctx context.Context, created entity.TComment) {
	if c.notifications == nil || c.users == nil || c.articles == nil || created.Id == 0 {
		return
	}
	if created.IsReview != 1 {
		return
	}

	actorNickname := ""
	if actor, err := c.users.GetByID(ctx, created.UserId); err == nil {
		actorNickname = actor.Nickname
	}

	recipientID := 0
	replyTo := ""
	contentType := ""
	contentID := 0
	switch {
	case created.ParentId != 0:
		if created.ReplyUserId <= 0 {
			return
		}
		recipientID = created.ReplyUserId
		replyTo = actorNickname
		contentType, contentID = commentTarget(created.Type, created.TopicId)
	case created.Type == 1 && created.TopicId != 0:
		article, err := c.articles.GetArticleRecord(ctx, created.TopicId)
		if err != nil {
			slog.WarnContext(ctx, "load article for notification failed", "error", err)
			return
		}
		recipientID = article.UserId
		contentType = port.FollowContentArticle
		contentID = article.Id
	case created.Type == 5 && created.TopicId != 0:
		if c.talks == nil {
			return
		}
		talk, err := c.talks.Get(ctx, created.TopicId)
		if err != nil {
			slog.WarnContext(ctx, "load talk for notification failed", "error", err)
			return
		}
		recipientID = talk.UserId
		contentType = port.FollowContentTalk
		contentID = talk.Id
	case created.Type == commentTypeCollection && created.TopicId != 0:
		if c.collections == nil {
			return
		}
		collection, err := c.collections.GetPublicByID(ctx, created.TopicId)
		if err != nil {
			slog.WarnContext(ctx, "load collection for notification failed", "error", err)
			return
		}
		if collection.Owner == nil {
			return
		}
		recipientID = collection.Owner.Id
		contentType = port.FollowContentCollection
		contentID = collection.ID
	case created.Type == commentTypeProfileWall && created.TopicId != 0:
		recipientID = created.TopicId
		contentType = port.FollowContentProfile
		contentID = created.TopicId
	default:
		return
	}
	if recipientID <= 0 || recipientID == created.UserId || contentType == "" || contentID <= 0 {
		return
	}

	recipient, err := c.users.GetByID(ctx, recipientID)
	if err != nil {
		slog.WarnContext(ctx, "load notification recipient failed", "error", err)
		return
	}
	if recipient.Email == "" || recipient.IsDisable != 0 || recipient.NotifyComment == 0 || !c.commentNoticeEnabled(ctx) {
		return
	}
	if !c.notificationAllowed(ctx, recipientID, created.Id) {
		return
	}

	notification := port.CommentNotification{
		CommentID:   created.Id,
		RecipientID: recipientID,
		Recipient:   recipient.Email,
		Nickname:    recipient.Nickname,
		ReplyAuthor: replyTo,
		CommentBody: created.CommentContent,
	}
	switch contentType {
	case port.FollowContentArticle:
		article, err := c.articles.GetArticleRecord(ctx, contentID)
		if err != nil {
			slog.WarnContext(ctx, "load article notification payload failed", "error", err)
			return
		}
		notification.ArticleID = article.Id
		notification.ArticleTitle = article.ArticleTitle
		notification.ArticleURL = config.PublicSiteURL + "/articles/" + strconv.Itoa(article.Id)
	case port.FollowContentTalk:
		talk, err := c.talks.Get(ctx, contentID)
		if err != nil {
			slog.WarnContext(ctx, "load talk notification payload failed", "error", err)
			return
		}
		notification.ArticleID = talk.Id
		notification.ArticleTitle = commentExcerpt(talk.Content, 80)
		notification.ArticleURL = config.PublicSiteURL + "/talks/" + strconv.Itoa(talk.Id)
	case port.FollowContentProfile:
		notification.ArticleID = recipient.Id
		notification.ArticleTitle = recipient.Nickname + " 的主页留言"
		notification.ArticleURL = config.PublicSiteURL + "/u/" + recipient.Handle + "#comments"
	case port.FollowContentCollection:
		if c.collections == nil {
			return
		}
		collection, err := c.collections.GetPublicByID(ctx, contentID)
		if err != nil {
			slog.WarnContext(ctx, "load collection notification payload failed", "error", err)
			return
		}
		notification.ArticleID = collection.ID
		notification.ArticleTitle = collection.Title
		notification.ArticleURL = config.PublicSiteURL + "/collections/" + collection.Slug
	default:
		return
	}
	if err := c.notifications.EnqueueComment(notification); err != nil {
		slog.WarnContext(ctx, "enqueue comment notification failed", "error", err)
	}
}

func commentTarget(commentType, topicID int) (string, int) {
	if topicID <= 0 {
		return "", 0
	}
	switch commentType {
	case commentTypeArticle:
		return port.FollowContentArticle, topicID
	case commentTypeTalk:
		return port.FollowContentTalk, topicID
	case commentTypeCollection:
		return port.FollowContentCollection, topicID
	case commentTypeProfileWall:
		return port.FollowContentProfile, topicID
	default:
		return "", 0
	}
}
func commentExcerpt(value string, limit int) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if limit > 0 && len([]rune(value)) > limit {
		value = string([]rune(value)[:limit]) + "…"
	}
	return value
}
func (c *MyCommentService) commentNoticeEnabled(ctx context.Context) bool {
	result := c.websiteService().GetWebsiteConfig(ctx)
	config, ok := result.Data.(model.WebsiteConfigDTO)
	if !ok {
		return false
	}
	return config.IsEmailNotice == 1
}

// notificationAllowed applies the per-comment dedupe and the per-recipient
// daily cap. Rate-limit failures are treated as "do not send".
func (c *MyCommentService) notificationAllowed(ctx context.Context, recipientID, commentID int) bool {
	allowed, err := allowRateLimit(ctx, c.limiter, "comment-notify:", fmt.Sprintf("%d:%d", recipientID, commentID), 1, commentNotifyDedupeWindow)
	if err != nil {
		slog.WarnContext(ctx, "comment notification dedupe failed", "error", err)
		return false
	}
	if !allowed {
		return false
	}
	allowed, err = allowRateLimit(ctx, c.limiter, "comment-notify-daily:", strconv.Itoa(recipientID), commentNotifyDailyLimit, commentNotifyDedupeWindow)
	if err != nil {
		slog.WarnContext(ctx, "comment notification daily cap failed", "error", err)
		return false
	}
	return allowed
}
