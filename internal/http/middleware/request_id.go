package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const RequestIDHeader = "X-Request-ID"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set(RequestIDHeader, requestID)
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte("fallback-request-id"))
	}
	return hex.EncodeToString(buf)
}
