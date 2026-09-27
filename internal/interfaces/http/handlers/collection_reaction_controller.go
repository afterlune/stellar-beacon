package api

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/eternallyzzz/stellar-beacon/internal/domain/errors"
	"github.com/eternallyzzz/stellar-beacon/internal/domain/port"
	"github.com/eternallyzzz/stellar-beacon/internal/interfaces/http/model"
	"github.com/gin-gonic/gin"
)

// ToggleCollectionReaction likes or unlikes a public/unlisted collection.
// @Router /v1/auth/me/collection-reactions [PUT]
func ToggleCollectionReaction(c *gin.Context) {
	var request model.CollectionReactionToggleVO
	if err := c.ShouldBind(&request); err != nil || request.CollectionId <= 0 {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.Invalid("collection_reaction.toggle", "invalid collection")))
		return
	}
	userID, _ := authenticatedUserInfoID(c)
	result, err := collectionReactionService.ToggleCollectionReaction(c.Request.Context(), userID, request.CollectionId, request.Reaction, request.Active)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.CollectionReactionToggleDTO{
		Active: result.Active, LikeCount: result.LikeCount, FavoriteCount: result.FavoriteCount,
	}))
}

// GetCollectionReactionState returns the current user's like state.
// @Router /v1/auth/me/collection-reactions/state [GET]
func GetCollectionReactionState(c *gin.Context) {
	collectionID, err := strconv.Atoi(c.Query("collectionId"))
	if err != nil || collectionID <= 0 {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.Invalid("collection_reaction.state", "invalid collection")))
		return
	}
	userID, _ := authenticatedUserInfoID(c)
	state, err := collectionReactionService.GetCollectionReactionState(c.Request.Context(), userID, collectionID)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.CollectionReactionStateDTO{
		CollectionId: collectionID, Like: state.Like, Favorite: state.Favorite,
	}))
}

// ListMyCollectionReactions lists collections saved by the current account.
// @Router /v1/auth/me/collection-reactions [GET]
func ListMyCollectionReactions(c *gin.Context) {
	reaction := strings.TrimSpace(c.Query("reaction"))
	if reaction != port.ReactionFavorite {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.Invalid("collection_reaction.list", "unsupported reaction")))
		return
	}
	userID, ok := authenticatedUserInfoID(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFromError(apperrors.New(apperrors.KindUnauthorized, "collection_reaction.user", nil)))
		return
	}
	current, size, ok := requestPageParams(c)
	if !ok {
		c.JSON(http.StatusOK, model.ResultFailWithMessage("参数格式不正确"))
		return
	}
	records, total, err := collectionReactionService.ListMyCollectionReactions(c.Request.Context(), userID, current, size, reaction)
	if err != nil {
		c.JSON(http.StatusOK, model.ResultFromError(err))
		return
	}
	c.JSON(http.StatusOK, model.ResultOkWithData(model.PageResultDTO{Records: records, Count: total, Page: current, PageSize: size}))
}
