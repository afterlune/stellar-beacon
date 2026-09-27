package service

import (
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"

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
	commentTypeArticle     = 1
	commentTypeTalk        = 5
	commentTypeCollection  = 6
	commentTypeProfileWall = 7
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
	ReportComment(c *gin.Context) model.ResultVO
	AppealComment(c *gin.Context) model.ResultVO
	EscalateCommentAppeal(c *gin.Context) model.ResultVO
	ListMyCommentAppeals(c *gin.Context) model.ResultVO
	ListOwnerCommentReports(c *gin.Context) model.ResultVO
	ResolveOwnerCommentReports(c *gin.Context) model.ResultVO
	ListOwnerCommentAppeals(c *gin.Context) model.ResultVO
	ResolveOwnerCommentAppeal(c *gin.Context) model.ResultVO
	ListAdminCommentReports(c *gin.Context) model.ResultVO
	ResolveAdminCommentReports(c *gin.Context) model.ResultVO
	ListAdminCommentAppeals(c *gin.Context) model.ResultVO
	ResolveAdminCommentAppeal(c *gin.Context) model.ResultVO
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
