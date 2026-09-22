package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ToggleCollectionReaction likes or unlikes a public/unlisted collection.
// @Router /v1/auth/me/collection-reactions [PUT]
func ToggleCollectionReaction(c *gin.Context) {
	c.JSON(http.StatusOK, collectionReactionService.ToggleCollectionReaction(c))
}

// GetCollectionReactionState returns the current user's like state.
// @Router /v1/auth/me/collection-reactions/state [GET]
func GetCollectionReactionState(c *gin.Context) {
	c.JSON(http.StatusOK, collectionReactionService.GetCollectionReactionState(c))
}
