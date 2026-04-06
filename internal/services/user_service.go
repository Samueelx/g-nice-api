package services

import (
	"errors"
	"time"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// UpdateProfileRequest uses pointer fields so the handler can distinguish
// between a field not sent at all vs. intentionally set to an empty value.
type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
	Bio         *string `json:"bio"          binding:"omitempty,max=500"`
	AvatarURL   *string `json:"avatar_url"   binding:"omitempty,max=500"`
	IsPrivate   *bool   `json:"is_private"`
}

// PublicProfile is the safe representation returned for other users' profiles.
// Email is intentionally omitted — only the owner sees their own email via /users/me.
type PublicProfile struct {
	ID             uint      `json:"id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	Bio            string    `json:"bio"`
	AvatarURL      string    `json:"avatar_url"`
	IsVerified     bool      `json:"is_verified"`
	IsPrivate      bool      `json:"is_private"`
	PostsCount     int       `json:"posts_count"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// toPublicProfile converts a User model into its safe, email-free public form.
func toPublicProfile(u *models.User) *PublicProfile {
	return &PublicProfile{
		ID:             u.ID,
		Username:       u.Username,
		DisplayName:    u.DisplayName,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		IsVerified:     u.IsVerified,
		IsPrivate:      u.IsPrivate,
		PostsCount:     u.PostsCount,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		CreatedAt:      u.CreatedAt,
	}
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrProfileNotFound = errors.New("user not found")

// ── Interface + implementation ────────────────────────────────────────────────

// UserService handles user profile operations.
type UserService interface {
	// GetMe returns the full User record for the authenticated user (includes email).
	GetMe(userID uint) (*models.User, error)
	// UpdateProfile applies a partial update and returns the refreshed User.
	UpdateProfile(userID uint, req *UpdateProfileRequest) (*models.User, error)
	// GetUserByUsername returns the public profile for any user.
	GetUserByUsername(username string) (*PublicProfile, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService constructs a UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// GetMe fetches the authenticated user's own record.
func (s *userService) GetMe(userID uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProfileNotFound
	}
	return user, err
}

// UpdateProfile applies only the non-nil fields from the request.
func (s *userService) UpdateProfile(userID uint, req *UpdateProfileRequest) (*models.User, error) {
	fields := make(map[string]interface{})

	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Bio != nil {
		fields["bio"] = *req.Bio
	}
	if req.AvatarURL != nil {
		fields["avatar_url"] = *req.AvatarURL
	}
	if req.IsPrivate != nil {
		fields["is_private"] = *req.IsPrivate
	}

	// Nothing to update — just return the current record.
	if len(fields) == 0 {
		return s.userRepo.FindByID(userID)
	}

	if err := s.userRepo.UpdateFields(userID, fields); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Re-fetch to return the fresh state from the database.
	return s.userRepo.FindByID(userID)
}

// GetUserByUsername looks up any user by their unique username and returns their public profile.
func (s *userService) GetUserByUsername(username string) (*PublicProfile, error) {
	user, err := s.userRepo.FindByUsername(username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return toPublicProfile(user), nil
}
