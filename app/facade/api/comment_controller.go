package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// SaveComment
// @Summary		 评论模块
// @Description  添加评论
// @Success		 200	{object} model.ResultVO
// @Router       /comments/save [POST]
func SaveComment(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.SaveComment(c))
}

// GetComments
// @Summary		 评论模块
// @Description  获取评论
// @Success		 200	{object} model.ResultVO
// @Router       /comments [GET]
func GetComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListComments(c))
}

// ListRepliesByCommentId
// @Summary		 评论模块
// @Description  根据commentId获取回复
// @Success		 200	{object} model.ResultVO
// @Router       /comments/:commentId/replies [GET]
func ListRepliesByCommentId(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListRepliesByCommentId(c))
}

// ListTopSixComments
// @Summary		 评论模块
// @Description  获取前六个评论
// @Success		 200	{object} model.ResultVO
// @Router       /comments/topSix [GET]
func ListTopSixComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListTopSixComments())
}

// ListCommentBackDTO
// @Summary		 评论模块
// @Description  查询后台评论
// @Success		 200	{object} model.ResultVO
// @Router       /admin/comments [GET]
func ListCommentBackDTO(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.ListCommentBackDTO(c))
}

// UpdateCommentsReview
// @Summary		 评论模块
// @Description  审核评论
// @Success		 200	{object} model.ResultVO
// @Router       /admin/comments/review [PUT]
func UpdateCommentsReview(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.UpdateCommentsReview(c))
}

// DeleteComments
// @Summary		 评论模块
// @Description  删除评论
// @Success		 200	{object} model.ResultVO
// @Router       /admin/comments [DELETE]
func DeleteComments(c *gin.Context) {
	c.JSON(http.StatusOK, commentService.DeleteComments(c))
}
