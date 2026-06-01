package repository

import (
	"errors"
	"time"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// JokeRepository defines the data-access contract for Joke entities.
type JokeRepository interface {
	Create(joke *models.Joke) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	GetByID(id uint) (*models.Joke, error)
	GetByDate(date time.Time) (*models.Joke, error)
	ListAll(limit, offset int) ([]models.Joke, int64, error)
	ToggleLike(jokeID, userID uint) (bool, int, error)
	IncrementCommentCount(id uint, delta int) error
}

type jokeRepository struct {
	db *gorm.DB
}

// NewJokeRepository constructs the GORM-backed JokeRepository.
func NewJokeRepository(db *gorm.DB) JokeRepository {
	return &jokeRepository{db: db}
}

func (r *jokeRepository) Create(joke *models.Joke) error {
	err := r.db.Create(joke).Error
	if isDuplicateKeyError(err) {
		return ErrDuplicateKey
	}
	return err
}

func (r *jokeRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&models.Joke{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		if isDuplicateKeyError(result.Error) {
			return ErrDuplicateKey
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *jokeRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Joke{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *jokeRepository) GetByID(id uint) (*models.Joke, error) {
	var joke models.Joke
	err := r.db.Preload("CreatedBy").First(&joke, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &joke, err
}

func (r *jokeRepository) GetByDate(date time.Time) (*models.Joke, error) {
	var joke models.Joke
	err := r.db.Preload("CreatedBy").Where("active_date = ?", date.Format("2006-01-02")).First(&joke).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &joke, err
}

func (r *jokeRepository) ListAll(limit, offset int) ([]models.Joke, int64, error) {
	var jokes []models.Joke
	var total int64

	base := r.db.Model(&models.Joke{})

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("CreatedBy").
		Order("active_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&jokes).Error

	return jokes, total, err
}

func (r *jokeRepository) ToggleLike(jokeID, userID uint) (bool, int, error) {
	var liked bool
	var count int

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.JokeLike
		err := tx.Where("joke_id = ? AND user_id = ?", jokeID, userID).First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Not liked yet, so create like
			if err := tx.Create(&models.JokeLike{JokeID: jokeID, UserID: userID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Joke{}).Where("id = ?", jokeID).UpdateColumn("likes_count", gorm.Expr("likes_count + ?", 1)).Error; err != nil {
				return err
			}
			liked = true
		} else if err == nil {
			// Already liked, so hard-delete the like (Unscoped removes it
			// permanently so the unique index stays clean for future likes).
			if err := tx.Unscoped().Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.Joke{}).Where("id = ?", jokeID).UpdateColumn("likes_count", gorm.Expr("likes_count - ?", 1)).Error; err != nil {
				return err
			}
			liked = false
		} else {
			return err
		}

		var joke models.Joke
		if err := tx.Select("likes_count").First(&joke, jokeID).Error; err != nil {
			return err
		}
		count = joke.LikesCount
		return nil
	})

	return liked, count, err
}

func (r *jokeRepository) IncrementCommentCount(id uint, delta int) error {
	return r.db.Model(&models.Joke{}).
		Where("id = ?", id).
		UpdateColumn("comments_count", gorm.Expr("comments_count + ?", delta)).
		Error
}
