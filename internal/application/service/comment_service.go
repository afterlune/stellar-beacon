package service

import (
	"container/list"
	"context"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/entity"
	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"strconv"

	"github.com/gin-gonic/gin"
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
	repo    port.CommentRepository
	website StellarBeaconInfoService
}

func NewCommentService(deps CommentServiceDeps) (*MyCommentService, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	return &MyCommentService{repo: deps.Repo, website: deps.Website}, nil
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
	if err := c.commentRepository().Create(ctx.Request.Context(), comment); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
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
