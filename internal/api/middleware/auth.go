package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth returns a Gin middleware that validates the Authorization header.
// The expected token is passed at construction time (read from env by the caller).
func APIKeyAuth(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header must be in the format: Bearer <token>",
			})
			return
		}

		if parts[1] != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			return
		}

		c.Next()
	}
}
