package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// PostRepository defines the data-access contract for Post entities.
type PostRepository interface {
	Create(post *models.Post) error
	FindByID(id uint) (*models.Post, error)
	FindByIDWithAuthor(id uint) (*models.Post, error)
	// ListFeed returns all public posts ordered by newest first, with Author preloaded.
	ListFeed(limit, offset int) ([]models.Post, int64, error)
	// ListByUserID returns public posts for a specific user ordered by newest first.
	ListByUserID(userID uint, limit, offset int) ([]models.Post, int64, error)
	UpdateFields(id uint, fields map[string]interface{}) error
	IncrementCounter(id uint, column string, delta int) error
	Delete(id uint) error
	// Search performs a case-insensitive search on post content (public posts only).
	// Results are ordered by created_at DESC. Author is preloaded.
	Search(query string, limit, offset int) ([]models.Post, int64, error)
}

type postRepository struct {
	db *gorm.DB
}

// NewPostRepository constructs the GORM-backed PostRepository.
func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *postRepository) FindByID(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.First(&post, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &post, err
}

func (r *postRepository) FindByIDWithAuthor(id uint) (*models.Post, error) {
	var post models.Post
	err := r.db.Preload("Author").First(&post, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &post, err
}

func (r *postRepository) ListFeed(limit, offset int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	base := r.db.Model(&models.Post{}).Where("is_public = ?", true)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Author").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) ListByUserID(userID uint, limit, offset int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	base := r.db.Model(&models.Post{}).Where("user_id = ? AND is_public = ?", userID, true)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Author").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, total, err
}

func (r *postRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&models.Post{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postRepository) IncrementCounter(id uint, column string, delta int) error {
	return r.db.Model(&models.Post{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).
		Error
}

func (r *postRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Post{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Search performs a case-insensitive infix search on post content.
// Only public posts are searched; Author is preloaded for each result.
func (r *postRepository) Search(query string, limit, offset int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	pattern := "%" + query + "%"
	base := r.db.Model(&models.Post{}).
		Where("content ILIKE ? AND is_public = ?", pattern, true)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Author").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, total, err
}
