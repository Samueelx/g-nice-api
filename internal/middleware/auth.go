package middleware

import (
	"net/http"
	"strings"

	"github.com/Samueelx/g-nice-api/internal/token"
	"github.com/gin-gonic/gin"
)

// AuthRequired returns a Gin middleware that validates a JWT Bearer token
// using the provided token.Service and injects the claims into the context.
func AuthRequired(ts *token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "missing or malformed authorization header",
			})
			return
		}

		claims, err := ts.Parse(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid or expired token",
			})
			return
		}

		// Make claims available downstream
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("claims", claims)
		c.Next()
	}
}
