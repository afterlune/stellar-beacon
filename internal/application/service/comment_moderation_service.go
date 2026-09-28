package service

import (
	"container/list"
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	apperrors "github.com/afterlune/stellar-beacon/internal/domain/errors"
	"github.com/afterlune/stellar-beacon/internal/domain/port"
	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"

	"github.com/gin-gonic/gin"
)

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
	if vo.Type == Article || vo.Type == Talk || vo.Type == Collection || vo.Type == ProfileWall {
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
