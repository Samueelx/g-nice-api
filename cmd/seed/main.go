//go:build seed

// Run with: go run -tags seed ./cmd/seed
package main

import (
	"log"
	"os"
	"time"

	"github.com/Samueelx/g-nice-api/internal/config"
	"github.com/Samueelx/g-nice-api/internal/db"
	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	Username    string
	Email       string
	DisplayName string
	Password    string // plaintext — hashed before insert
	Bio         string
	IsAdmin     bool
	IsVerified  bool
}

var testUsers = []seedUser{
	{
		Username:    "alice",
		Email:       "alice@test.com",
		DisplayName: "Alice Wanjiru",
		Password:    "Password123!",
		Bio:         "Coffee enthusiast & weekend hiker. 🏔️",
		IsVerified:  true,
	},
	{
		Username:    "bob",
		Email:       "bob@test.com",
		DisplayName: "Bob Otieno",
		Password:    "Password123!",
		Bio:         "Software dev by day, guitarist by night. 🎸",
		IsVerified:  false,
	},
	{
		Username:    "carol",
		Email:       "carol@test.com",
		DisplayName: "Carol Muthoni",
		Password:    "Password123!",
		Bio:         "Photographer & storyteller. 📷",
		IsVerified:  true,
	},
	{
		Username:    "admin",
		Email:       "admin@test.com",
		DisplayName: "Platform Admin",
		Password:    "Admin1234!",
		Bio:         "G-Nice platform administrator.",
		IsAdmin:     true,
		IsVerified:  true,
	},
}

type seedEvent struct {
	Title       string
	Description string
	Location    string
	Date        time.Time
	Price       float64
	Category    string
	ImageURL    string
	IsFeatured  bool
}

// Dates are set relative to now so events always feel upcoming when you seed.
var testEvents = []seedEvent{
	{
		Title:       "Nairobi Wine & Dine Mixer",
		Description: "An exclusive evening of fine wines, craft cocktails, and gourmet bites. Connect with foodies and wine enthusiasts from across Nairobi in an intimate rooftop setting.",
		Location:    "Sky Lounge, Westlands, Nairobi",
		Date:        time.Now().AddDate(0, 0, 14),
		Price:       2500,
		Category:    "Food & Drink",
		ImageURL:    "https://images.unsplash.com/photo-1510812431401-41d2bd2722f3?w=1200",
		IsFeatured:  true,
	},
	{
		Title:       "Garden Gala: Flowers & Candlelight Dinner",
		Description: "A magical al-fresco dining experience set among lush flower arrangements and warm candlelight. Perfect for date nights, anniversaries, and intimate celebrations.",
		Location:    "The Botanical Gardens, Karen, Nairobi",
		Date:        time.Now().AddDate(0, 0, 21),
		Price:       3500,
		Category:    "Lifestyle",
		ImageURL:    "https://images.unsplash.com/photo-1464366400600-7168b8af9bc3?w=1200",
		IsFeatured:  false,
	},
	{
		Title:       "Love & Community Wellness Fair",
		Description: "A free community wellness day celebrating mental health, self-love, and connection. Featuring yoga sessions, motivational talks, group meditations, and pop-up markets.",
		Location:    "Uhuru Park, Nairobi CBD",
		Date:        time.Now().AddDate(0, 0, 30),
		Price:       0,
		Category:    "Health & Wellness",
		ImageURL:    "https://images.unsplash.com/photo-1489710437720-ebb67ec84dd2?w=1200",
		IsFeatured:  true,
	},
	{
		Title:       "Afrobeats Live: Nairobi Nights",
		Description: "Experience Nairobi's most electrifying live music night. Top Afrobeats and Afropop artists take the stage for a night of non-stop dancing, culture, and vibes.",
		Location:    "KICC Open Air Grounds, Nairobi",
		Date:        time.Now().AddDate(0, 1, 7),
		Price:       1500,
		Category:    "Music & Entertainment",
		ImageURL:    "https://images.unsplash.com/photo-1540039155733-5bb30b53aa14?w=1200",
		IsFeatured:  true,
	},
	{
		Title:       "G-Nice Annual Summit 2026",
		Description: "The flagship G-Nice community summit bringing together creators, entrepreneurs, and innovators. Full-day programme of keynotes, panel discussions, workshops, and networking.",
		Location:    "Sarit Expo Centre, Westlands, Nairobi",
		Date:        time.Now().AddDate(0, 2, 0),
		Price:       5000,
		Category:    "Conference",
		ImageURL:    "https://images.unsplash.com/photo-1540575467063-178a50c2df87?w=1200",
		IsFeatured:  false,
	},
}

func main() {
	// ── Load .env first so APP_ENV is available for the guard ───────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, reading from system environment")
	}

	// ── Safety guard — refuse to run outside development ──────────────────────
	if os.Getenv("APP_ENV") != "development" {
		log.Fatal("🚫 seed: refusing to run outside APP_ENV=development")
	}

	// ── Load config & connect DB ──────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Config error: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("❌ Database error: %v", err)
	}

	if err := db.Migrate(database); err != nil {
		log.Fatalf("❌ Migration error: %v", err)
	}

	// ── Seed users ────────────────────────────────────────────────────────────
	log.Println("🌱 Seeding test users...")

	for _, u := range testUsers {
		// Skip if the username already exists so the script is safely re-runnable.
		var count int64
		database.Model(&models.User{}).Where("username = ?", u.Username).Count(&count)
		if count > 0 {
			log.Printf("   ⏭️  skipping %q — already exists", u.Username)
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), 12)
		if err != nil {
			log.Fatalf("❌ bcrypt error for %q: %v", u.Username, err)
		}

		user := &models.User{
			Username:        u.Username,
			Email:           u.Email,
			DisplayName:     u.DisplayName,
			PasswordHash:    string(hash),
			Bio:             u.Bio,
			IsAdmin:         u.IsAdmin,
			IsVerified:      u.IsVerified,
			IsEmailVerified: true, // always bypass OTP for seed users
		}

		if err := database.Create(user).Error; err != nil {
			log.Fatalf("❌ Failed to create user %q: %v", u.Username, err)
		}

		log.Printf("   ✅ created %-12s  email: %-22s  password: %s  admin: %v",
			u.Username, u.Email, u.Password, u.IsAdmin)
	}

	// ── Seed events ───────────────────────────────────────────────────────────
	log.Println("🌱 Seeding test events...")

	// Events are authored by the admin user — look them up first.
	var adminUser models.User
	if err := database.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Fatalf("❌ Could not find admin user to author events. Seed users first: %v", err)
	}

	for _, e := range testEvents {
		var count int64
		database.Model(&models.Event{}).Where("title = ?", e.Title).Count(&count)
		if count > 0 {
			log.Printf("   ⏭️  skipping event %q — already exists", e.Title)
			continue
		}

		event := &models.Event{
			Title:       e.Title,
			Description: e.Description,
			Location:    e.Location,
			Date:        e.Date,
			Price:       e.Price,
			Category:    e.Category,
			ImageURL:    e.ImageURL,
			IsFeatured:  e.IsFeatured,
			UserID:      adminUser.ID,
		}

		if err := database.Create(event).Error; err != nil {
			log.Fatalf("❌ Failed to create event %q: %v", e.Title, err)
		}

		log.Printf("   ✅ created event: %s  [%s]  KES %.0f  featured: %v",
			e.Title, e.Category, e.Price, e.IsFeatured)
	}

	log.Println("✅ Seeding complete.")
}
