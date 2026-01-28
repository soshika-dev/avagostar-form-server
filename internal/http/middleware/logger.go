package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		requestID, _ := c.Get(RequestIDHeader)
		requestIDStr, _ := requestID.(string)

		log.Info("request completed",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", latency,
			"request_id", requestIDStr,
			"ip", c.ClientIP(),
		)
	}
}
