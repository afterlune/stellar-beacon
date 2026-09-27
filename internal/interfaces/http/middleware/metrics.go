package middlewares

import (
	"time"

	"github.com/eternallyzzz/stellar-beacon/internal/infrastructure/monitoring"
	"github.com/gin-gonic/gin"
)

// RequestMetrics records bounded aggregate HTTP measurements for the system
// monitor. Request details are deliberately not retained.
func RequestMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()
		monitoring.RecordHTTP(c.FullPath(), c.Request.URL.Path, c.Writer.Status(), time.Since(startedAt))
	}
}
