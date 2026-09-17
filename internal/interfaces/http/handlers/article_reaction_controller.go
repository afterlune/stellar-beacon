package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// ToggleArticleReaction
// @Summary		 读者互动
// @Description  点赞或收藏一篇文章（需要登录）
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/reactions [PUT]
func ToggleArticleReaction(c *gin.Context) {
	c.JSON(http.StatusOK, articleReactionService.ToggleArticleReaction(c))
}

// ListMyArticleReactions
// @Summary		 读者互动
// @Description  分页查询当前账号的点赞或收藏
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/reactions [GET]
func ListMyArticleReactions(c *gin.Context) {
	c.JSON(http.StatusOK, articleReactionService.ListMyArticleReactions(c))
}

// ListArticleReactionStates
// @Summary		 读者互动
// @Description  批量查询当前账号对指定文章的互动状态
// @Success		 200	{object} model.ResultVO
// @Router       /v1/auth/me/reactions/state [GET]
func ListArticleReactionStates(c *gin.Context) {
	c.JSON(http.StatusOK, articleReactionService.ListArticleReactionStates(c))
}
