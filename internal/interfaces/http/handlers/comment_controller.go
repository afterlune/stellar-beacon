package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// SaveComment
// @Summary		 评论模块
// @Description  添加评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/comments [POST]
func SaveComment(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.SaveComment(c))
}

// GetComments
// @Summary		 评论模块
// @Description  获取评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/comments [GET]
func GetComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListComments(c))
}

// ListRepliesByCommentId
// @Summary		 评论模块
// @Description  根据commentId获取回复
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/comments/{commentId}/replies [GET]
func ListRepliesByCommentId(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListRepliesByCommentId(c))
}

// ListTopSixComments
// @Summary		 评论模块
// @Description  获取前六个评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/public/comments/top [GET]
func ListTopSixComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListTopSixComments())
}

// ListCommentBackDTO
// @Summary		 评论模块
// @Description  查询后台评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/comments [GET]
func ListCommentBackDTO(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListCommentBackDTO(c))
}

// UpdateCommentsReview
// @Summary		 评论模块
// @Description  审核评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/comments/review [PUT]
func UpdateCommentsReview(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.UpdateCommentsReview(c))
}

// DeleteComments
// @Summary		 评论模块
// @Description  删除评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/comments [DELETE]
func DeleteComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.DeleteComments(c))
}

// PinCollectionComment
// @Summary		 书单评论治理
// @Description  书单作者置顶或取消置顶单条根评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/studio/collections/{collectionId}/comments/{commentId}/pin [PUT]
func PinCollectionComment(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.PinCollectionComment(c))
}

// DeleteOwnedCollectionComment
// @Summary		 书单评论治理
// @Description  书单作者软删除书单内任意评论或回复
// @Success		 200	{object} model.ResultVO
// @Router       /v1/studio/collections/{collectionId}/comments/{commentId} [DELETE]
func DeleteOwnedCollectionComment(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.DeleteOwnedCollectionComment(c))
}

// ListOwnedCollectionComments
// @Summary		 书单评论治理
// @Description  书单作者查看自己书单下的评论治理列表（含已删除）
// @Success		 200	{object} model.ResultVO
// @Router       /v1/studio/collections/{collectionId}/comments [GET]
func ListOwnedCollectionComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListOwnedCollectionComments(c))
}

// BatchModerateCollectionComments
// @Summary		 书单评论治理
// @Description  书单作者批量删除、置顶或取消置顶书单内评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/studio/collections/{collectionId}/comments/batch [POST]
func BatchModerateCollectionComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.BatchModerateCollectionComments(c))
}

// RestoreOwnedCollectionComments
// @Summary		 书单评论治理
// @Description  书单作者恢复被软删除的评论，仅还原同一次删除级联隐藏的回复
// @Success		 200	{object} model.ResultVO
// @Router       /v1/studio/collections/{collectionId}/comments/restore [POST]
func RestoreOwnedCollectionComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.RestoreOwnedCollectionComments(c))
}

// ListCollectionCommentsAdmin
// @Summary		 书单评论治理
// @Description  管理端按书单分页查看评论（含已删除与举报数）
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/collections/{collectionId}/comments [GET]
func ListCollectionCommentsAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListCollectionCommentsAdmin(c))
}

// RestoreCommentsAdmin
// @Summary		 书单评论治理
// @Description  管理员恢复被软删除的书单评论
// @Success		 200	{object} model.ResultVO
// @Router       /v1/admin/comments/{commentId}/restore [PUT]
func RestoreCommentsAdmin(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.RestoreCommentsAdmin(c))
}
