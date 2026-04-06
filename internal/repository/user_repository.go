package repository

import (
	"errors"
	"strings"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// UserRepository defines the data-access contract for User entities.
// All methods return repository-level sentinel errors (ErrNotFound, ErrDuplicateKey)
// so callers never need to import gorm or database driver packages.
type UserRepository interface {
	Create(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	ExistsByEmail(email string) (bool, error)
	ExistsByUsername(username string) (bool, error)
	Update(user *models.User) error
	// UpdateFields performs a partial update: only the keys present in fields are written.
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository constructs the GORM-backed implementation.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &user, err
}

func (r *userRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// UpdateFields writes only the supplied columns for the user with the given id.
// Uses GORM's Updates(map) which skips zero-value fields intentionally.
func (r *userRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&models.User{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// isDuplicateKeyError checks whether an error is a PostgreSQL unique violation.
func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "23505") // PostgreSQL SQLSTATE
}
