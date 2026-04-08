package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// FollowRepository defines the data-access contract for Follow entities.
type FollowRepository interface {
	// Create inserts a follow edge. Returns ErrDuplicateKey if it already exists.
	Create(follow *models.Follow) error
	// Delete removes a follow edge by follower+followee IDs.
	// Returns nil (not ErrNotFound) when the edge doesn't exist — callers treat it as idempotent.
	Delete(followerID, followeeID uint) error
	// FindByFollowerAndFollowee returns the follow record, or ErrNotFound.
	FindByFollowerAndFollowee(followerID, followeeID uint) (*models.Follow, error)
	// ListFollowers returns User records for users who follow the given userID.
	ListFollowers(userID uint, limit, offset int) ([]models.User, int64, error)
	// ListFollowing returns User records for users that the given userID follows.
	ListFollowing(userID uint, limit, offset int) ([]models.User, int64, error)
}

type followRepository struct {
	db *gorm.DB
}

// NewFollowRepository constructs the GORM-backed implementation.
func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Create(follow *models.Follow) error {
	if err := r.db.Create(follow).Error; err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (r *followRepository) Delete(followerID, followeeID uint) error {
	// Hard-delete — follow records have no meaningful "deleted" state.
	// We use Unscoped so a future re-follow doesn't violate the unique index on a soft-deleted row.
	return r.db.Unscoped().
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&models.Follow{}).
		Error
}

func (r *followRepository) FindByFollowerAndFollowee(followerID, followeeID uint) (*models.Follow, error) {
	var f models.Follow
	err := r.db.
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &f, err
}

// ListFollowers returns the User records of everyone who follows userID,
// using a subquery so we only hit the users table for the actual user data.
func (r *followRepository) ListFollowers(userID uint, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	followerIDs := r.db.Model(&models.Follow{}).Select("follower_id").Where("followee_id = ?", userID)

	base := r.db.Model(&models.User{}).Where("id IN (?)", followerIDs)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Order("username ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}

// ListFollowing returns the User records of everyone that userID follows.
func (r *followRepository) ListFollowing(userID uint, limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	followeeIDs := r.db.Model(&models.Follow{}).Select("followee_id").Where("follower_id = ?", userID)

	base := r.db.Model(&models.User{}).Where("id IN (?)", followeeIDs)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Order("username ASC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error

	return users, total, err
}
