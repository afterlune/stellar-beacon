package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListContentAudits(c *gin.Context) {
	c.JSON(http.StatusOK, contentAuditService.ListContentAudits(c))
}

func ListContentAuditItems(c *gin.Context) {
	c.JSON(http.StatusOK, contentAuditService.ListContentAuditItems(c))
}
