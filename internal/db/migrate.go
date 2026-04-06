package db

import (
	"fmt"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all application models and then applies
// any additional raw-SQL constraints that AutoMigrate cannot express.
func Migrate(db *gorm.DB) error {
	// AutoMigrate creates/alters tables to match struct definitions.
	// Order matters: referenced tables must exist before their dependents.
	err := db.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Comment{},
		&models.Like{},
		&models.Follow{},
		&models.Notification{},
	)
	if err != nil {
		return fmt.Errorf("AutoMigrate failed: %w", err)
	}

	// ── Composite unique indexes not expressible via struct tags ───────────────

	// Prevent a user from liking the same target twice.
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_unique
		ON likes (user_id, target_id, target_type)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("likes unique index: %w", err)
	}

	// Prevent duplicate follow edges.
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_follows_unique
		ON follows (follower_id, followee_id)
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return fmt.Errorf("follows unique index: %w", err)
	}

	// Prevent a user from following themselves (CHECK constraint).
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_name = 'follows'
				  AND constraint_name = 'chk_follows_no_self_follow'
			) THEN
				ALTER TABLE follows
				ADD CONSTRAINT chk_follows_no_self_follow
				CHECK (follower_id <> followee_id);
			END IF;
		END
		$$
	`).Error; err != nil {
		return fmt.Errorf("follows self-follow check: %w", err)
	}

	return nil
}
