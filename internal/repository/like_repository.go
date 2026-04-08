package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// LikeRepository defines the data-access contract for Like entities.
type LikeRepository interface {
	// FindByUserAndTarget looks up an existing like — returns ErrNotFound if absent.
	FindByUserAndTarget(userID, targetID uint, targetType models.LikeTargetType) (*models.Like, error)
	Create(like *models.Like) error
	Delete(id uint) error
}

type likeRepository struct {
	db *gorm.DB
}

// NewLikeRepository constructs the GORM-backed implementation.
func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) FindByUserAndTarget(userID, targetID uint, targetType models.LikeTargetType) (*models.Like, error) {
	var like models.Like
	err := r.db.Where(
		"user_id = ? AND target_id = ? AND target_type = ?",
		userID, targetID, targetType,
	).First(&like).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &like, err
}

func (r *likeRepository) Create(like *models.Like) error {
	if err := r.db.Create(like).Error; err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (r *likeRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Like{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
