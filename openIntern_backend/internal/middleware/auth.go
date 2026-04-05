package middleware

import (
	"strconv"
	"strings"

	"openIntern/internal/response"
	accountsvc "openIntern/internal/services/account"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		claims, err := accountsvc.ParseToken(parts[1])
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		refreshedToken, expiresAt, err := accountsvc.GenerateToken(claims.UserID)
		if err == nil {
			c.Header("X-Access-Token", refreshedToken)
			c.Header("X-Token-Expires", strconv.FormatInt(expiresAt, 10))
		}
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
