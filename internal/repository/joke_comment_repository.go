package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// JokeCommentRepository defines the data-access contract for JokeComment entities.
type JokeCommentRepository interface {
	Create(comment *models.JokeComment) error
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	GetByID(id uint) (*models.JokeComment, error)
	ListComments(jokeID uint, limit, offset int) ([]models.JokeComment, int64, error)
	ListReplies(parentID uint, limit, offset int) ([]models.JokeComment, int64, error)
	ToggleLike(commentID, userID uint) (bool, int, error)
	IncrementReplyCount(id uint, delta int) error
}

type jokeCommentRepository struct {
	db *gorm.DB
}

// NewJokeCommentRepository constructs the GORM-backed JokeCommentRepository.
func NewJokeCommentRepository(db *gorm.DB) JokeCommentRepository {
	return &jokeCommentRepository{db: db}
}

func (r *jokeCommentRepository) Create(comment *models.JokeComment) error {
	return r.db.Create(comment).Error
}

func (r *jokeCommentRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&models.JokeComment{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *jokeCommentRepository) Delete(id uint) error {
	result := r.db.Delete(&models.JokeComment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *jokeCommentRepository) GetByID(id uint) (*models.JokeComment, error) {
	var comment models.JokeComment
	err := r.db.Preload("User").First(&comment, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &comment, err
}

func (r *jokeCommentRepository) ListComments(jokeID uint, limit, offset int) ([]models.JokeComment, int64, error) {
	var comments []models.JokeComment
	var total int64

	base := r.db.Model(&models.JokeComment{}).Where("joke_id = ? AND parent_id IS NULL", jokeID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	return comments, total, err
}

func (r *jokeCommentRepository) ListReplies(parentID uint, limit, offset int) ([]models.JokeComment, int64, error) {
	var comments []models.JokeComment
	var total int64

	base := r.db.Model(&models.JokeComment{}).Where("parent_id = ?", parentID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("User").
		Order("created_at ASC"). // Usually replies are ordered chronological
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	return comments, total, err
}

func (r *jokeCommentRepository) ToggleLike(commentID, userID uint) (bool, int, error) {
	var liked bool
	var count int

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.JokeCommentLike
		err := tx.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&models.JokeCommentLike{CommentID: commentID, UserID: userID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.JokeComment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count + ?", 1)).Error; err != nil {
				return err
			}
			liked = true
		} else if err == nil {
			// Hard-delete so the unique index slot is freed for future likes.
			if err := tx.Unscoped().Delete(&existing).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.JokeComment{}).Where("id = ?", commentID).UpdateColumn("likes_count", gorm.Expr("likes_count - ?", 1)).Error; err != nil {
				return err
			}
			liked = false
		} else {
			return err
		}

		var comment models.JokeComment
		if err := tx.Select("likes_count").First(&comment, commentID).Error; err != nil {
			return err
		}
		count = comment.LikesCount
		return nil
	})

	return liked, count, err
}

func (r *jokeCommentRepository) IncrementReplyCount(id uint, delta int) error {
	return r.db.Model(&models.JokeComment{}).
		Where("id = ?", id).
		UpdateColumn("replies_count", gorm.Expr("replies_count + ?", delta)).
		Error
}
