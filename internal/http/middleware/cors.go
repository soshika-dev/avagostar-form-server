package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0
	allowedMap := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		allowedMap[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll || originAllowed(origin, allowedMap) {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
			}
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func originAllowed(origin string, allowedMap map[string]struct{}) bool {
	if origin == "" {
		return false
	}
	if _, ok := allowedMap[origin]; ok {
		return true
	}
	for allowed := range allowedMap {
		if strings.HasPrefix(origin, allowed) {
			return true
		}
	}
	return false
}
