package router

import (
	"log"
	"net/http"

	"github.com/Samueelx/g-nice-api/internal/config"
	"github.com/Samueelx/g-nice-api/internal/email"
	"github.com/Samueelx/g-nice-api/internal/handlers"
	"github.com/Samueelx/g-nice-api/internal/middleware"
	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/Samueelx/g-nice-api/internal/storage"
	"github.com/Samueelx/g-nice-api/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New creates and configures the Gin engine with all middleware and routes.
func New(db *gorm.DB, ts *token.Service, mailer email.Sender, cfg *config.Config) *gin.Engine {
	r := gin.New()

	// Enforce 10 MB multipart limit — must match the UploadService constant.
	r.MaxMultipartMemory = 10 << 20

	// ── Global middleware ─────────────────────────────────────────────────────
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())

	// ── Dependency wiring ─────────────────────────────────────────────────────
	userRepo    := repository.NewUserRepository(db)
	postRepo    := repository.NewPostRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	likeRepo    := repository.NewLikeRepository(db)
	followRepo  := repository.NewFollowRepository(db)
	notifRepo   := repository.NewNotificationRepository(db)

	// Notifications (constructed first — other services depend on it)
	notifSvc     := services.NewNotificationService(notifRepo)
	notifHandler := handlers.NewNotificationHandler(notifSvc)

	// Auth
	authSvc     := services.NewAuthService(userRepo, ts, mailer)
	authHandler := handlers.NewAuthHandler(authSvc)

	// Users
	userSvc     := services.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

	// Posts
	postSvc     := services.NewPostService(postRepo, userRepo)
	postHandler := handlers.NewPostHandler(postSvc)

	// Comments
	commentSvc     := services.NewCommentService(commentRepo, postRepo, notifSvc)
	commentHandler := handlers.NewCommentHandler(commentSvc)

	// Likes
	likeSvc     := services.NewLikeService(likeRepo, postRepo, commentRepo, notifSvc)
	likeHandler := handlers.NewLikeHandler(likeSvc)

	// Follows
	followSvc     := services.NewFollowService(followRepo, userRepo, notifSvc)
	followHandler := handlers.NewFollowHandler(followSvc)

	// Search
	searchSvc     := services.NewSearchService(userRepo, postRepo)
	searchHandler := handlers.NewSearchHandler(searchSvc)

	// Media uploads (S3)
	s3Store, err := storage.NewS3Storage(storage.S3Config{
		AccessKeyID:     cfg.AWSAccessKeyID,
		SecretAccessKey: cfg.AWSSecretAccessKey,
		Region:          cfg.AWSRegion,
		Bucket:          cfg.AWSS3Bucket,
	})
	if err != nil {
		log.Fatalf("❌ Failed to initialise S3 storage: %v", err)
	}
	uploadSvc     := services.NewUploadService(s3Store)
	uploadHandler := handlers.NewUploadHandler(uploadSvc)


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
		v1.GET("/users/:username",           userHandler.GetUserByUsername)
		v1.GET("/users/:username/posts",     postHandler.ListUserPosts)
		v1.GET("/users/:username/followers", followHandler.ListFollowers)
		v1.GET("/users/:username/following", followHandler.ListFollowing)
		v1.GET("/posts",                     postHandler.ListFeed)
		v1.GET("/posts/:id",                 postHandler.GetPost)
		v1.GET("/posts/:id/comments",        commentHandler.ListComments)
		v1.GET("/comments/:cid/replies",     commentHandler.ListReplies)
		v1.GET("/search",                    searchHandler.Search)

		// ── Protected routes (JWT required) ───────────────────────────────────
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(ts))
		{
			// User profile
			protected.GET("/users/me",             userHandler.GetMe)
			protected.PATCH("/users/me",           userHandler.UpdateMe)
			protected.GET("/users/me/followers",   followHandler.GetMyFollowers)
			protected.GET("/users/me/following",   followHandler.GetMyFollowing)

			// Follows (protected mutations)
			protected.POST("/users/:username/follow",   followHandler.Follow)
			protected.DELETE("/users/:username/follow", followHandler.Unfollow)
			protected.GET("/users/:username/follow",    followHandler.CheckFollowing)

			// Posts
			protected.POST("/posts",          postHandler.CreatePost)
			protected.PATCH("/posts/:id",      postHandler.UpdatePost)
			protected.DELETE("/posts/:id",     postHandler.DeletePost)

			// Comments
			protected.POST("/posts/:id/comments",                  commentHandler.CreateComment)
			protected.POST("/posts/:id/comments/:cid/replies",     commentHandler.CreateReply)
			protected.PATCH("/comments/:cid",                      commentHandler.UpdateComment)
			protected.DELETE("/comments/:cid",                     commentHandler.DeleteComment)

			// Likes (toggle)
			protected.POST("/posts/:id/like",     likeHandler.TogglePostLike)
			protected.POST("/comments/:cid/like", likeHandler.ToggleCommentLike)

			// Notifications
			// NOTE: read-all must be registered before /:nid to avoid Gin routing it as an ID.
			protected.GET("/notifications",                notifHandler.List)
			protected.GET("/notifications/unread-count",   notifHandler.UnreadCount)
			protected.PATCH("/notifications/read-all",     notifHandler.MarkAllRead)
			protected.PATCH("/notifications/:nid/read",    notifHandler.MarkRead)
			protected.DELETE("/notifications/:nid",        notifHandler.Delete)

			// Media uploads
			protected.POST("/uploads",                     uploadHandler.Upload)
		}
	}

	return r
}

