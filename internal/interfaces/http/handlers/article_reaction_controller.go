package api

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ToggleArticleReaction
// @Summary         读者互动
// @Description    点赞或收藏一篇文章（需要登录）
// @Success        200 {object} model.ResultVO
// @Router         /v1/auth/me/reactions [PUT]
func ToggleArticleReaction(c *gin.Context) {
	var request model.ReactionToggleVO
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, _ := authenticatedUserInfoID(c)
	result, err := articleReactionService.ToggleArticleReaction(c.Request.Context(), userID, request.ArticleId, request.Reaction, request.Active)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.ReactionToggleDTO{
		Active: result.Active, LikeCount: result.LikeCount, FavoriteCount: result.FavoriteCount,
	}))
}

// ListMyArticleReactions
// @Summary         读者互动
// @Description    分页查询当前账号的点赞或收藏
// @Success        200 {object} model.ResultVO
// @Router         /v1/auth/me/reactions [GET]
func ListMyArticleReactions(c *gin.Context) {
	current, err := strconv.Atoi(c.Query("current"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	userID, _ := authenticatedUserInfoID(c)
	records, total, err := articleReactionService.ListMyArticleReactions(c.Request.Context(), userID, current, size, c.Query("reaction"))
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: int(total)}))
}

// ListArticleReactionStates
// @Summary         读者互动
// @Description    批量查询当前账号对指定文章的互动状态
// @Success        200 {object} model.ResultVO
// @Router         /v1/auth/me/reactions/state [GET]
func ListArticleReactionStates(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("articleIds"))
	var articleIDs []int
	if raw != "" {
		parts := strings.Split(raw, ",")
		articleIDs = make([]int, 0, len(parts))
		for _, part := range parts {
			id, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				c.JSON(http.StatusOK, model.ResultFromError(apperrors.Invalid("article_reaction.states", "invalid article id")))
				return
			}
			articleIDs = append(articleIDs, id)
		}
	}
	userID, _ := authenticatedUserInfoID(c)
	states, err := articleReactionService.ListArticleReactionStates(c.Request.Context(), userID, articleIDs)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	dtos, err := mapResponseDTO[[]model.ReactionStateDTO](states)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(dtos))
}
