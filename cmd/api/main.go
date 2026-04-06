package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Samueelx/g-nice-api/internal/config"
	"github.com/Samueelx/g-nice-api/internal/db"
	"github.com/Samueelx/g-nice-api/internal/router"
	"github.com/Samueelx/g-nice-api/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// ── Load .env ─────────────────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, reading environment variables from system")
	}

	// ── Load config ───────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config error: %v", err)
	}

	// ── Set Gin mode ──────────────────────────────────────────────────────────
	gin.SetMode(cfg.GinMode)

	// ── Build token service ───────────────────────────────────────────────────
	ts := token.New(cfg.JWTSecret)

	// ── Connect to database ───────────────────────────────────────────────────
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ Database error: %v", err)
	}

	// ── Run migrations ────────────────────────────────────────────────────────
	if err := db.Migrate(database); err != nil {
		log.Fatalf("❌ Migration error: %v", err)
	}
	log.Println("✅ Database migrations applied")

	// ── Build router ──────────────────────────────────────────────────────────
	r := router.New(database, ts)

	// ── Start server with graceful shutdown ───────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 g-nice-api listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Forced shutdown: %v", err)
	}
	log.Println("✅ Server exited cleanly")
}
