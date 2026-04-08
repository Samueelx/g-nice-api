package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// CommentRepository defines the data-access contract for Comment entities.
type CommentRepository interface {
	Create(comment *models.Comment) error
	FindByID(id uint) (*models.Comment, error)
	FindByIDWithAuthor(id uint) (*models.Comment, error)
	// ListByPostID returns top-level comments (ParentID IS NULL) for a post, newest first.
	ListByPostID(postID uint, limit, offset int) ([]models.Comment, int64, error)
	// ListReplies returns direct replies to a comment.
	ListReplies(parentID uint, limit, offset int) ([]models.Comment, int64, error)
	UpdateContent(id uint, content string) error
	IncrementCounter(id uint, column string, delta int) error
	Delete(id uint) error
}

type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository constructs the GORM-backed implementation.
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

func (r *commentRepository) FindByID(id uint) (*models.Comment, error) {
	var c models.Comment
	err := r.db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *commentRepository) FindByIDWithAuthor(id uint) (*models.Comment, error) {
	var c models.Comment
	err := r.db.Preload("Author").First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (r *commentRepository) ListByPostID(postID uint, limit, offset int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var total int64

	base := r.db.Model(&models.Comment{}).
		Where("post_id = ? AND parent_id IS NULL", postID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Author").
		Order("created_at ASC"). // show oldest first in comment threads
		Limit(limit).
		Offset(offset).
		Find(&comments).Error

	return comments, total, err
}

func (r *commentRepository) ListReplies(parentID uint, limit, offset int) ([]models.Comment, int64, error) {
	var replies []models.Comment
	var total int64

	base := r.db.Model(&models.Comment{}).Where("parent_id = ?", parentID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Author").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&replies).Error

	return replies, total, err
}

func (r *commentRepository) UpdateContent(id uint, content string) error {
	result := r.db.Model(&models.Comment{}).
		Where("id = ?", id).
		Update("content", content)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *commentRepository) IncrementCounter(id uint, column string, delta int) error {
	return r.db.Model(&models.Comment{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).
		Error
}

func (r *commentRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Comment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
