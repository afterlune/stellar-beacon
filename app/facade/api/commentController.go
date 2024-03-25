package api

import (
	"benetnasch/app/application/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

// SaveComment
// @Summary		 评论模块
// @Description  添加评论
// @Success		 200	{object}	model.ResultVO
// @Router       /comments/save [POST]
func SaveComment(c *gin.Context) {
	res := service.SaveComment(c)
	c.JSON(http.StatusOK, res)
}

// GetComments
// @Summary		 评论模块
// @Description  获取评论
// @Success		 200	{object}	model.ResultVO
// @Router       /comments [GET]
func GetComments(c *gin.Context) {
	res := service.ListComments(c)
	c.JSON(http.StatusOK, res)
}

// ListRepliesByCommentId
// @Summary		 评论模块
// @Description  根据commentId获取回复
// @Success		 200	{object}	model.ResultVO
// @Router       /comments/:commentId/replies [GET]
func ListRepliesByCommentId(c *gin.Context) {
	res := service.ListRepliesByCommentId(c)
	c.JSON(http.StatusOK, res)
}

// ListTopSixComments
// @Summary		 评论模块
// @Description  获取前六个评论
// @Success		 200	{object}	model.ResultVO
// @Router       /comments/topSix [GET]
func ListTopSixComments(c *gin.Context) {
	res := service.ListTopSixComments()
	c.JSON(http.StatusOK, res)
}

// ListCommentBackDTO
// @Summary		 评论模块
// @Description  查询后台评论
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/comments [GET]
func ListCommentBackDTO(c *gin.Context) {
	res := service.ListCommentBackDTO(c)
	c.JSON(http.StatusOK, res)
}

// UpdateCommentsReview
// @Summary		 评论模块
// @Description  审核评论
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/comments/review [PUT]
func UpdateCommentsReview(c *gin.Context) {
	res := service.UpdateCommentsReview(c)
	c.JSON(http.StatusOK, res)
}

// DeleteComments
// @Summary		 评论模块
// @Description  删除评论
// @Success		 200	{object}	model.ResultVO
// @Router       /admin/comments [DELETE]
func DeleteComments(c *gin.Context) {
	res := service.DeleteComments(c)
	c.JSON(http.StatusOK, res)
}
