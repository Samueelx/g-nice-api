package router

import (
	"net/http"

	"github.com/Samueelx/g-nice-api/internal/email"
	"github.com/Samueelx/g-nice-api/internal/handlers"
	"github.com/Samueelx/g-nice-api/internal/middleware"
	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/Samueelx/g-nice-api/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New creates and configures the Gin engine with all middleware and routes.
func New(db *gorm.DB, ts *token.Service, mailer email.Sender) *gin.Engine {
	r := gin.New()

	// ── Global middleware ─────────────────────────────────────────────────────
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// ── Dependency wiring ─────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)

	// Auth
	authSvc := services.NewAuthService(userRepo, ts, mailer)
	authHandler := handlers.NewAuthHandler(authSvc)

	// Users
	userSvc := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

	// Posts
	postSvc := services.NewPostService(postRepo, userRepo)
	postHandler := handlers.NewPostHandler(postSvc)

	// ── Health check ──────────────────────────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── API v1 ────────────────────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// ── Auth (public) ─────────────────────────────────────────────────────
		auth := v1.Group("/auth")
		{
			auth.POST("/register",       authHandler.Register)
			auth.POST("/verify-otp",     authHandler.VerifyOTP)
			auth.POST("/resend-otp",     authHandler.ResendOTP)
			auth.POST("/login",          authHandler.Login)
		}

		// ── Public read-only resources ────────────────────────────────────────
		v1.GET("/users/:username", userHandler.GetUserByUsername)
		v1.GET("/users/:username/posts", postHandler.ListUserPosts)
		v1.GET("/posts", postHandler.ListFeed)
		v1.GET("/posts/:id", postHandler.GetPost)

		// ── Protected routes (JWT required) ───────────────────────────────────
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(ts))
		{
			// User profile
			protected.GET("/users/me", userHandler.GetMe)
			protected.PATCH("/users/me", userHandler.UpdateMe)

			// Posts
			protected.POST("/posts", postHandler.CreatePost)
			protected.PATCH("/posts/:id", postHandler.UpdatePost)
			protected.DELETE("/posts/:id", postHandler.DeletePost)
		}
	}


	return r
}
