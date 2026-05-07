package middleware

import (
	"net/http"

	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/gin-gonic/gin"
)

// AdminRequired returns a Gin middleware that verifies the authenticated user
// has the IsAdmin flag set. It MUST be chained after AuthRequired, which injects
// the "userID" value into the context.
//
// On failure it aborts with 403 Forbidden using the standard response envelope.
func AdminRequired(userRepo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("userID")
		if !exists {
			// AuthRequired should have caught this; treat as internal misconfiguration.
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "not authenticated"})
			return
		}

		userID, ok := userIDVal.(uint)
		if !ok || userID == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "invalid token claims"})
			return
		}

		user, err := userRepo.FindByID(userID)
		if err != nil || !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": "admin access required"})
			return
		}

		c.Next()
	}
}
