package router

import (
	"github.com/Samueelx/g-nice-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New creates and configures the Gin engine with all middleware and routes.
func New(db *gorm.DB) *gin.Engine {
	r := gin.New()

	// Global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// ── Health check ──────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── API v1 ────────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		// auth := v1.Group("/auth")
		// {
		// 	auth.POST("/register", handlers.Register(db))
		// 	auth.POST("/login",    handlers.Login(db))
		// }

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			// Users
			// protected.GET("/users/me", handlers.GetMe(db))

			// Posts / feed — to be added next
		}
	}

	return r
}
