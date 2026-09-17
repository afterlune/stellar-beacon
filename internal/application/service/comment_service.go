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
	"time"

	"github.com/gin-gonic/gin"
)

// Comment notification policy: one mail per recipient and comment, capped per
// recipient per day so a busy thread cannot flood an inbox.
const (
	commentNotifyDedupeWindow = 24 * time.Hour
	commentNotifyDailyLimit   = 20
)

type CommentService interface {
	ListTopSixComments() model.ResultVO
	ListComments(c *gin.Context) model.ResultVO
	SaveComment(c *gin.Context) model.ResultVO
	ListRepliesByCommentId(c *gin.Context) model.ResultVO
	ListCommentBackDTO(c *gin.Context) model.ResultVO
	UpdateCommentsReview(c *gin.Context) model.ResultVO
	DeleteComments(c *gin.Context) model.ResultVO
}

type MyCommentService struct {
	repo          port.CommentRepository
	website       StellarBeaconInfoService
	users         port.UserInfoRepository
	articles      port.ArticleRepository
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
	filter := port.CommentFilter{Current: current, Size: size, Type: commentVO.Type}
	if commentVO.TopicId != "" {
		topicID, err := strconv.Atoi(commentVO.TopicId)
		if err != nil {
			return model.ResultFailWithMessage("参数校验异常")
		}
		filter.TopicID = &topicID
	}
	commentData, count, err := c.commentRepository().ListComments(ctx.Request.Context(), filter)
	if err != nil {
		return model.ResultFromError(err)
	}
	if count == 0 || len(commentData) == 0 {
		return model.ResultOkWithData(model.PageResultDTO{Records: list.New(), Count: 0})
	}
	commentIDs := make([]int, 0, len(commentData))
	for _, comment := range commentData {
		commentIDs = append(commentIDs, comment.Id)
	}
	replyData, err := c.commentRepository().ListReplies(ctx.Request.Context(), commentIDs)
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
	return model.ResultOkWithData(model.PageResultDTO{Records: commentData, Count: count})
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
	c.notifyComment(ctx.Request.Context(), comment, dto)
	return model.ResultOk()
}

// notifyComment queues the reply/article notification after a comment is
// stored. Delivery is best effort: a mail problem must never fail the write,
// and the queue owns retries.
func (c *MyCommentService) notifyComment(ctx context.Context, created entity.TComment, author model.UserDetailsDTO) {
	if c.notifications == nil || c.users == nil || c.articles == nil || created.Id == 0 {
		return
	}
	// Comments waiting for moderation must not leak their content by email
	// before a moderator approves them.
	if created.IsReview != 1 {
		return
	}
	if !c.commentNoticeEnabled(ctx) {
		return
	}

	recipientID := 0
	replyTo := ""
	switch {
	case created.ParentId != 0:
		parent, err := c.commentRepository().GetByID(ctx, created.ParentId)
		if err != nil {
			slog.WarnContext(ctx, "load parent comment for notification failed", "error", err)
			return
		}
		recipientID = parent.UserId
		replyTo = author.Nickname
	case created.Type == 1 && created.TopicId != 0:
		article, err := c.articles.GetArticleRecord(ctx, created.TopicId)
		if err != nil {
			slog.WarnContext(ctx, "load article for notification failed", "error", err)
			return
		}
		recipientID = article.UserId
	default:
		// Message-board, about, friend-link and talk comments have no owner to
		// notify in this iteration.
		return
	}
	if recipientID <= 0 || recipientID == created.UserId {
		return
	}

	recipient, err := c.users.GetByID(ctx, recipientID)
	if err != nil {
		slog.WarnContext(ctx, "load notification recipient failed", "error", err)
		return
	}
	if recipient.Email == "" || recipient.IsDisable != 0 || recipient.NotifyComment == 0 {
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
	if created.Type == 1 && created.TopicId != 0 {
		article, err := c.articles.GetArticleRecord(ctx, created.TopicId)
		if err == nil && article.Id != 0 {
			notification.ArticleID = article.Id
			notification.ArticleTitle = article.ArticleTitle
			notification.ArticleURL = config.PublicSiteURL + "/articles/" + strconv.Itoa(article.Id)
		}
	}
	if err := c.notifications.EnqueueComment(notification); err != nil {
		slog.WarnContext(ctx, "enqueue comment notification failed", "error", err)
	}
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
	data, err := c.commentRepository().ListReplies(ctx.Request.Context(), []int{commentID})
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
	if err := c.commentRepository().Review(ctx.Request.Context(), vo.Ids, vo.IsReview); err != nil {
		return model.ResultFromError(err)
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

func (c *MyCommentService) checkComment(ctx context.Context, vo model.CommentVO) error {
	if len(TypeHM[vo.Type]) == 0 {
		return apperrors.Invalid("comment.validate", "invalid comment type")
	}
	if vo.Type == Article || vo.Type == Talk {
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
