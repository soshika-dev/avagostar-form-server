package middleware

import (
	"net/http"
	"strings"

	"avagostar-form-server/internal/services"
	"avagostar-form-server/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthConfig struct {
	Secret string
}

func JWTAuth(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.RespondError(c, utils.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "missing token", nil))
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &services.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.Secret), nil
		})
		if err != nil || !token.Valid {
			utils.RespondError(c, utils.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid token", nil))
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*services.Claims)
		if !ok {
			utils.RespondError(c, utils.NewAppError(http.StatusUnauthorized, "UNAUTHORIZED", "invalid token", nil))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}
