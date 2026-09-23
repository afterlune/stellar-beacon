package service

import (
	"container/list"
	"context"
	"fmt"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/config"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Comment notification policy: one mail per recipient and comment, capped per
// recipient per day so a busy thread cannot flood an inbox.
const (
	commentNotifyDedupeWindow = 24 * time.Hour
	commentNotifyDailyLimit   = 20
)

// Comment types carried by t_comment.type. Public count surfaces reuse these
// so a counter can never disagree with the target a comment was accepted for.
const (
	commentTypeArticle    = 1
	commentTypeTalk       = 5
	commentTypeCollection = 6
)

type CommentService interface {
	ListTopSixComments() model.ResultVO
	ListComments(c *gin.Context) model.ResultVO
	SaveComment(c *gin.Context) model.ResultVO
	ListRepliesByCommentId(c *gin.Context) model.ResultVO
	ListCommentBackDTO(c *gin.Context) model.ResultVO
	UpdateCommentsReview(c *gin.Context) model.ResultVO
	DeleteComments(c *gin.Context) model.ResultVO
	PinCollectionComment(c *gin.Context) model.ResultVO
	DeleteOwnedCollectionComment(c *gin.Context) model.ResultVO
	ListOwnedCollectionComments(c *gin.Context) model.ResultVO
	ListCollectionCommentsAdmin(c *gin.Context) model.ResultVO
	RestoreCommentsAdmin(c *gin.Context) model.ResultVO
	BatchModerateCollectionComments(c *gin.Context) model.ResultVO
	RestoreOwnedCollectionComments(c *gin.Context) model.ResultVO
}

type MyCommentService struct {
	repo          port.CommentRepository
	website       StellarBeaconInfoService
	users         port.UserInfoRepository
	articles      port.ArticleRepository
	talks         port.TalkRepository
	collections   port.CollectionPublicReader
	notifications port.CommentNotifier
	limiter       port.RateLimiter
}

func NewCommentService(deps CommentServiceDeps) (*MyCommentService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCommentService{
		repo:          deps.Repo,
		website:       deps.Website,
		users:         deps.Users,
		articles:      deps.Articles,
		talks:         deps.Talks,
		collections:   deps.Collections,
		notifications: deps.Notifications,
		limiter:       deps.Limiter,
	}, nil
}

func (c *MyCommentService) commentRepository() port.CommentRepository {
	return c.repo
}

func (c *MyCommentService) websiteService() StellarBeaconInfoService {
	return c.website
}

func (c *MyCommentService) ListTopSixComments() model.ResultVO {
	data, err := c.commentRepository().ListTopSixComments(context.Background())
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (c *MyCommentService) ListComments(ctx *gin.Context) model.ResultVO {
	current, err := strconv.Atoi(ctx.Query("current"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	size, err := strconv.Atoi(ctx.Query("size"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var commentVO model.CommentVO
	if err := ctx.ShouldBind(&commentVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CommentFilter{Current: current, Size: size, Type: commentVO.Type, ViewerID: optionalUserID(ctx)}
	if commentVO.TopicId != "" {
		topicID, err := strconv.Atoi(commentVO.TopicId)
		if err != nil {
			return model.ResultFailWithMessage("参数校验异常")
		}
		filter.TopicID = &topicID
	}
	if rawFocusCommentID := strings.TrimSpace(ctx.Query("focusCommentId")); rawFocusCommentID != "" {
		focusCommentID, parseErr := strconv.Atoi(rawFocusCommentID)
		if parseErr != nil || focusCommentID <= 0 || filter.TopicID == nil {
			return model.ResultFailWithMessage("参数格式不正确")
		}
		current, err = c.commentRepository().ResolveCommentPage(ctx.Request.Context(), filter.Type, *filter.TopicID, focusCommentID, filter.Size)
		if err != nil {
			return model.ResultFromError(err)
		}
		filter.Current = current
	}
	commentData, count, err := c.commentRepository().ListComments(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 || len(commentData) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0, Page: current, PageSize: size})
	}
	commentIDs := make([]int, 0, len(commentData))
	for _, comment := range commentData {
		commentIDs = append(commentIDs, comment.Id)
	}
	replyData, err := c.commentRepository().ListReplies(ctx.Request.Context(), commentIDs, filter.ViewerID)
	if err != nil {
		return model.ResultFromError(err)
	}
	replyMap := make(map[int][]model.ReplyDTO)
	for _, reply := range replyData {
		replyMap[reply.ParentId] = append(replyMap[reply.ParentId], *reply)
	}
	for _, comment := range commentData {
		comment.ReplyDTOs = replyMap[comment.Id]
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: commentData, Count: count, Page: current, PageSize: size})
}

func (c *MyCommentService) SaveComment(ctx *gin.Context) model.ResultVO {
	var commentVO model.CommentVO
	if err := ctx.ShouldBind(&commentVO); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.checkComment(ctx.Request.Context(), commentVO); err != nil {
		if apperrors.IsKind(err, apperrors.KindValidation) {
			return model.ResultFailWithMessage("参数校验异常")
		}
		return model.ResultFromError(err)
	}
	websiteConfigResult := c.websiteService().GetWebsiteConfig(ctx.Request.Context())
	websiteConfig, ok := websiteConfigResult.Data.(model.WebsiteConfigDTO)
	if !ok {
		return websiteConfigResult
	}
	isReview := 0
	if websiteConfig.IsCommentReview == False {
		isReview = 1
	}
	topicID := 0
	if commentVO.TopicId != "" {
		var err error
		topicID, err = strconv.Atoi(commentVO.TopicId)
		if err != nil {
			return model.ResultFailWithMessage("参数校验异常")
		}
	}
	value, ok := ctx.Get("userInfo")
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment.user", nil))
	}
	dto, ok := value.(model.UserDetailsDTO)
	if !ok {
		return model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "comment.user", nil))
	}
	comment := entity.TComment{
		UserId:         dto.UserInfoId,
		ReplyUserId:    commentVO.ReplyUserId,
		TopicId:        topicID,
		CommentContent: commentVO.CommentContent,
		ParentId:       commentVO.ParentId,
		Type:           commentVO.Type,
		IsReview:       isReview,
	}
	commentID, err := c.commentRepository().Create(ctx.Request.Context(), comment)
	if err != nil {
		return model.ResultFromError(err)
	}
	comment.Id = commentID
	if comment.IsReview == 1 {
		c.notifyComment(ctx.Request.Context(), comment)
		if err := c.commentRepository().MarkNotificationDispatched(ctx.Request.Context(), comment.Id); err != nil {
			slog.WarnContext(ctx.Request.Context(), "mark comment notification dispatched failed", "commentId", comment.Id, "error", err)
		}
	}
	return model.ResultOk()
}

// notifyComment queues the reply or content-owner email after a comment is
// approved. In-app notifications are written transactionally by the repository;
// email remains best effort and is retried by the bounded queue.
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
	if contentType == port.FollowContentArticle {
		article, err := c.articles.GetArticleRecord(ctx, contentID)
		if err != nil {
			slog.WarnContext(ctx, "load article notification payload failed", "error", err)
			return
		}
		notification.ArticleID = article.Id
		notification.ArticleTitle = article.ArticleTitle
		notification.ArticleURL = config.PublicSiteURL + "/articles/" + strconv.Itoa(article.Id)
	} else if contentType == port.FollowContentTalk {
		talk, err := c.talks.Get(ctx, contentID)
		if err != nil {
			slog.WarnContext(ctx, "load talk notification payload failed", "error", err)
			return
		}
		notification.ArticleID = talk.Id
		notification.ArticleTitle = commentExcerpt(talk.Content, 80)
		notification.ArticleURL = config.PublicSiteURL + "/talks/" + strconv.Itoa(talk.Id)
	} else {
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

func (c *MyCommentService) ListRepliesByCommentId(ctx *gin.Context) model.ResultVO {
	commentID, err := strconv.Atoi(ctx.Param("commentId"))
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	data, err := c.commentRepository().ListReplies(ctx.Request.Context(), []int{commentID}, optionalUserID(ctx))
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(data)
}

func (c *MyCommentService) ListCommentBackDTO(ctx *gin.Context) model.ResultVO {
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	filter := port.CommentFilter{
		Current:  vo.Current,
		Size:     vo.Size,
		Keywords: vo.Keywords,
		Type:     vo.Type,
		IsReview: vo.IsReview,
	}
	count, err := c.commentRepository().CountComments(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	data, err := c.commentRepository().ListCommentsAdmin(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: data, Count: int(count)})
}

func (c *MyCommentService) UpdateCommentsReview(ctx *gin.Context) model.ResultVO {
	var vo model.ReviewVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	pending := make([]entity.TComment, 0, len(vo.Ids))
	if vo.IsReview == 1 {
		for _, id := range vo.Ids {
			comment, err := c.commentRepository().GetByID(ctx.Request.Context(), id)
			if err != nil {
				continue
			}
			if comment.IsReview != 1 && comment.NotificationDispatchedAt == nil {
				pending = append(pending, comment)
			}
		}
	}
	if err := c.commentRepository().Review(ctx.Request.Context(), vo.Ids, vo.IsReview); err != nil {
		return model.ResultFromError(err)
	}
	for _, comment := range pending {
		comment.IsReview = 1
		c.notifyComment(ctx.Request.Context(), comment)
		if err := c.commentRepository().MarkNotificationDispatched(ctx.Request.Context(), comment.Id); err != nil {
			slog.WarnContext(ctx.Request.Context(), "mark reviewed comment notification dispatched failed", "commentId", comment.Id, "error", err)
		}
	}
	return model.ResultOk()
}

func (c *MyCommentService) DeleteComments(ctx *gin.Context) model.ResultVO {
	var ids []int
	if err := ctx.ShouldBind(&ids); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().Delete(ctx.Request.Context(), ids); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCommentService) PinCollectionComment(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	commentID, err := pathID(ctx, "commentId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CommentPinVO
	if err := ctx.ShouldBind(&vo); err != nil || vo.Pinned == nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().SetPinned(ctx.Request.Context(), user.UserInfoId, collectionID, commentID, *vo.Pinned); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCommentService) DeleteOwnedCollectionComment(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	commentID, err := pathID(ctx, "commentId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().SoftDeleteOwned(ctx.Request.Context(), user.UserInfoId, collectionID, commentID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCommentService) ListOwnedCollectionComments(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	current, size, err := pageParams(ctx)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	includeDeleted := ctx.Query("includeDeleted") == "1" || strings.EqualFold(ctx.Query("includeDeleted"), "true")
	records, total, err := c.commentRepository().ListOwnedCollectionComments(ctx.Request.Context(), user.UserInfoId, collectionID, current, size, ctx.Query("keywords"), includeDeleted)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) BatchModerateCollectionComments(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CommentBatchVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	result, err := c.commentRepository().BatchModerateOwned(ctx.Request.Context(), user.UserInfoId, collectionID, strings.TrimSpace(vo.Action), vo.CommentIds)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(result)
}

func (c *MyCommentService) RestoreOwnedCollectionComments(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CommentRestoreVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	result, err := c.commentRepository().RestoreOwned(ctx.Request.Context(), user.UserInfoId, collectionID, vo.CommentIds)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(result)
}

func (c *MyCommentService) ListCollectionCommentsAdmin(ctx *gin.Context) model.ResultVO {
	collectionID, err := pathID(ctx, "collectionId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.ConditionVO
	if err := ctx.ShouldBind(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if vo.Current <= 0 {
		vo.Current = 1
	}
	if vo.Size <= 0 || vo.Size > 100 {
		vo.Size = 12
	}
	filter := port.CommentFilter{
		Current:      vo.Current,
		Size:         vo.Size,
		Keywords:     vo.Keywords,
		Type:         6,
		IsReview:     vo.IsReview,
		CollectionID: collectionID,
	}
	total, err := c.commentRepository().CountComments(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	records, err := c.commentRepository().ListCommentsAdmin(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if records == nil {
		records = []*port.CommentAdmin{}
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(total), Page: vo.Current, PageSize: vo.Size})
}

func (c *MyCommentService) RestoreCommentsAdmin(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	commentID, err := pathID(ctx, "commentId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	comment, err := c.commentRepository().GetByID(ctx.Request.Context(), commentID)
	if err != nil {
		return model.ResultFromError(err)
	}
	if comment.Type != 6 || comment.TopicId <= 0 {
		return model.ResultFailWithMessage("只有书单评论支持恢复")
	}
	result, err := c.commentRepository().RestoreAsAdmin(ctx.Request.Context(), user.UserInfoId, comment.TopicId, []int{commentID})
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(result)
}
func (c *MyCommentService) checkComment(ctx context.Context, vo model.CommentVO) error {
	if len(TypeHM[vo.Type]) == 0 {
		return apperrors.Invalid("comment.validate", "invalid comment type")
	}
	if vo.Type == Article || vo.Type == Talk || vo.Type == Collection {
		if vo.TopicId == "" {
			return apperrors.Invalid("comment.validate", "topic is required")
		}
		topicID, err := strconv.Atoi(vo.TopicId)
		if err != nil {
			return apperrors.Invalid("comment.validate", "invalid topic")
		}
		if err := c.commentRepository().ValidateTarget(ctx, vo.Type, topicID); err != nil {
			return err
		}
	}
	if (vo.Type == Link || vo.Type == Abouts || vo.Type == Message) && vo.TopicId != "" {
		return apperrors.Invalid("comment.validate", "topic must be empty")
	}
	if vo.ParentId == 0 && vo.ReplyUserId != 0 {
		return apperrors.Invalid("comment.validate", "reply user requires parent")
	}
	if vo.ParentId != 0 {
		if vo.ReplyUserId == 0 {
			return apperrors.Invalid("comment.validate", "reply user is required")
		}
		return c.commentRepository().ValidateReply(ctx, vo.Type, vo.ParentId, vo.ReplyUserId)
	}
	return nil
}

var _ CommentService = (*MyCommentService)(nil)
