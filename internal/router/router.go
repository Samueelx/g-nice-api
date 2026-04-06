package router

import (
	"net/http"

	"github.com/Samueelx/g-nice-api/internal/handlers"
	"github.com/Samueelx/g-nice-api/internal/middleware"
	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/Samueelx/g-nice-api/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New creates and configures the Gin engine with all middleware and routes.
func New(db *gorm.DB, ts *token.Service) *gin.Engine {
	r := gin.New()

	// ── Global middleware ─────────────────────────────────────────────────────
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// ── Dependency wiring ─────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)

	// Auth
	authSvc := services.NewAuthService(userRepo, ts)
	authHandler := handlers.NewAuthHandler(authSvc)

	// Users
	userSvc := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

	// ── Health check ──────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── API v1 ────────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Public user profiles (no auth required to browse)
		v1.GET("/users/:username", userHandler.GetUserByUsername)

		// Protected routes (require valid JWT)
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(ts))
		{
			// Current user profile
			protected.GET("/users/me", userHandler.GetMe)
			protected.PATCH("/users/me", userHandler.UpdateMe)

			// Posts / feed — coming next
			// protected.GET("/posts",  postHandler.ListFeed)
			// protected.POST("/posts", postHandler.CreatePost)
		}
	}

	return r
}
