package service

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/afterlune/stellar-beacon/internal/interfaces/http/model"
)

// notifyModeration raises one in-app moderation notice. Delivery is best-effort:
// the moderation state change has already been committed and must not fail
// because a notice could not be written.
func (c *MyCommentService) notifyModeration(ctx context.Context, recipientID, actorID, commentID, collectionID int, dedupeKey string) {
	if recipientID <= 0 || recipientID == actorID || commentID <= 0 {
		return
	}
	if err := c.commentRepository().CreateModerationNotification(ctx, recipientID, actorID, "collection", collectionID, commentID, dedupeKey); err != nil {
		slog.WarnContext(ctx, "moderation notification failed", "commentId", commentID, "recipient", recipientID, "error", err)
	}
}

func moderationRecipients(items []int) []int {
	seen := make(map[int]bool, len(items))
	out := make([]int, 0, len(items))
	for _, id := range items {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// ReportComment lets a signed-in reader report a visible reading-list comment.
func (c *MyCommentService) ReportComment(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.CommentReportVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	comment, err := c.commentRepository().GetByID(ctx.Request.Context(), vo.CommentId)
	if err != nil {
		return model.ResultFromError(err)
	}
	if comment.Type != 6 || comment.TopicId <= 0 || comment.IsReview != 1 || comment.IsDelete != 0 {
		return model.ResultFailWithMessage("该评论当前不可举报")
	}
	if err := c.commentRepository().CreateCommentReport(ctx.Request.Context(), comment.Id, comment.TopicId, user.UserInfoId, vo.Reason, vo.Detail); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

// AppealComment lets the author of a hidden comment ask for a review.
func (c *MyCommentService) AppealComment(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	var vo model.CommentAppealVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().CreateCommentAppeal(ctx.Request.Context(), user.UserInfoId, vo.CommentId, vo.Reason); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCommentService) EscalateCommentAppeal(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	appealID, err := pathID(ctx, "appealId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	if err := c.commentRepository().EscalateCommentAppeal(ctx.Request.Context(), user.UserInfoId, appealID); err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOk()
}

func (c *MyCommentService) ListMyCommentAppeals(ctx *gin.Context) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	current, size, err := pageParams(ctx)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, total, err := c.commentRepository().ListMyCommentAppeals(ctx.Request.Context(), user.UserInfoId, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) ListOwnerCommentReports(ctx *gin.Context) model.ResultVO {
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
	records, total, err := c.commentRepository().ListOwnerCommentReports(ctx.Request.Context(), user.UserInfoId, collectionID, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) ListAdminCommentReports(ctx *gin.Context) model.ResultVO {
	current, size, err := pageParams(ctx)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, total, err := c.commentRepository().ListAdminCommentReports(ctx.Request.Context(), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) resolveCommentReports(ctx *gin.Context, actorRole string) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	commentID, err := pathID(ctx, "commentId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CommentModerationDecisionVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	collectionID := 0
	if actorRole == "owner" {
		collectionID, err = pathID(ctx, "collectionId")
		if err != nil {
			return model.ResultFailWithMessage("参数格式不正确")
		}
	} else {
		comment, err := c.commentRepository().GetByID(ctx.Request.Context(), commentID)
		if err != nil {
			return model.ResultFromError(err)
		}
		if comment.Type != 6 {
			return model.ResultFailWithMessage("只有书单评论支持举报处理")
		}
		collectionID = comment.TopicId
	}
	comment, err := c.commentRepository().GetByID(ctx.Request.Context(), commentID)
	if err != nil {
		return model.ResultFromError(err)
	}
	reporters, err := c.commentRepository().ResolveCommentReports(ctx.Request.Context(), user.UserInfoId, actorRole, collectionID, commentID, vo.Decision, vo.Reason)
	if err != nil {
		return model.ResultFromError(err)
	}
	for _, reporterID := range moderationRecipients(reporters) {
		c.notifyModeration(ctx.Request.Context(), reporterID, user.UserInfoId, commentID, collectionID,
			"moderation:report:"+vo.Decision+":"+strconv.Itoa(commentID)+":"+strconv.Itoa(reporterID))
	}
	if vo.Decision == "hide" || vo.Decision == "restore" {
		c.notifyModeration(ctx.Request.Context(), comment.UserId, user.UserInfoId, commentID, collectionID,
			"moderation:comment:"+vo.Decision+":"+strconv.Itoa(commentID))
	}
	return model.ResultOk()
}

func (c *MyCommentService) ResolveOwnerCommentReports(ctx *gin.Context) model.ResultVO {
	return c.resolveCommentReports(ctx, "owner")
}

func (c *MyCommentService) ResolveAdminCommentReports(ctx *gin.Context) model.ResultVO {
	return c.resolveCommentReports(ctx, "admin")
}

func (c *MyCommentService) ListOwnerCommentAppeals(ctx *gin.Context) model.ResultVO {
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
	records, total, err := c.commentRepository().ListOwnerCommentAppeals(ctx.Request.Context(), user.UserInfoId, collectionID, current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) ListAdminCommentAppeals(ctx *gin.Context) model.ResultVO {
	current, size, err := pageParams(ctx)
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	records, total, err := c.commentRepository().ListAdminCommentAppeals(ctx.Request.Context(), current, size)
	if err != nil {
		return model.ResultFromError(err)
	}
	return model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size})
}

func (c *MyCommentService) resolveCommentAppeal(ctx *gin.Context, actorRole string) model.ResultVO {
	user, ok := currentUser(ctx)
	if !ok {
		return model.ResultFailWithStatus(model.NO_LOGIN)
	}
	appealID, err := pathID(ctx, "appealId")
	if err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	var vo model.CommentModerationDecisionVO
	if err := ctx.ShouldBindJSON(&vo); err != nil {
		return model.ResultFailWithMessage("参数格式不正确")
	}
	outcome, err := c.commentRepository().ResolveCommentAppeal(ctx.Request.Context(), user.UserInfoId, actorRole, appealID, vo.Decision, vo.Reason)
	if err != nil {
		return model.ResultFromError(err)
	}
	c.notifyModeration(ctx.Request.Context(), outcome.AppellantID, user.UserInfoId, outcome.CommentID, outcome.CollectionID,
		"moderation:appeal:"+actorRole+":"+vo.Decision+":"+strconv.Itoa(appealID)+":"+strconv.Itoa(outcome.AppellantID))
	if actorRole == "admin" {
		c.notifyModeration(ctx.Request.Context(), outcome.CollectionOwnerID, user.UserInfoId, outcome.CommentID, outcome.CollectionID,
			"moderation:appeal:admin:"+vo.Decision+":"+strconv.Itoa(appealID)+":"+strconv.Itoa(outcome.CollectionOwnerID))
	}
	return model.ResultOk()
}

func (c *MyCommentService) ResolveOwnerCommentAppeal(ctx *gin.Context) model.ResultVO {
	return c.resolveCommentAppeal(ctx, "owner")
}

func (c *MyCommentService) ResolveAdminCommentAppeal(ctx *gin.Context) model.ResultVO {
	return c.resolveCommentAppeal(ctx, "admin")
}
