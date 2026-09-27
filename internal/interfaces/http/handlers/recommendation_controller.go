package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func QueryRecommendations(c *gin.Context) {
	c.JSON(http.StatusOK, recommendationService.Query(c))
}

func ListRecommendationFeedback(c *gin.Context) {
	c.JSON(http.StatusOK, recommendationService.ListFeedback(c))
}

func SaveRecommendationFeedback(c *gin.Context) {
	c.JSON(http.StatusOK, recommendationService.SaveFeedback(c))
}

func DeleteRecommendationFeedback(c *gin.Context) {
	c.JSON(http.StatusOK, recommendationService.DeleteFeedback(c))
}
