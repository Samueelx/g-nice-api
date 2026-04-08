package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// FollowResult is returned by Follow and Unfollow operations.
// It carries the updated state so the frontend can update its UI immediately.
type FollowResult struct {
	Following      bool `json:"following"`       // true = now following
	FollowersCount int  `json:"followers_count"` // updated count on the target user
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrSelfFollow    = errors.New("you cannot follow yourself")
	ErrFollowNotFound = errors.New("user not found")
)

// ── Interface ─────────────────────────────────────────────────────────────────

// FollowService handles follow/unfollow operations and follow list retrieval.
type FollowService interface {
	// Follow creates a follow from followerID → followeeUsername.
	// Idempotent: returns the current state without error if already following.
	Follow(followerID uint, followeeUsername string) (*FollowResult, error)

	// Unfollow removes the follow relationship.
	// Idempotent: returns the current state without error if not following.
	Unfollow(followerID uint, followeeUsername string) (*FollowResult, error)

	// ListFollowers returns a paginated list of public profiles for followers of username.
	ListFollowers(username string, page, pageSize int) (*pagination.Page[PublicProfile], error)

	// ListFollowing returns a paginated list of public profiles for users that username follows.
	ListFollowing(username string, page, pageSize int) (*pagination.Page[PublicProfile], error)

	// ListFollowersByID returns followers for a known userID (used by /me routes).
	ListFollowersByID(userID uint, page, pageSize int) (*pagination.Page[PublicProfile], error)

	// ListFollowingByID returns following for a known userID (used by /me routes).
	ListFollowingByID(userID uint, page, pageSize int) (*pagination.Page[PublicProfile], error)

	// IsFollowing reports whether followerID is currently following followeeUsername.
	IsFollowing(followerID uint, followeeUsername string) (bool, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type followService struct {
	followRepo repository.FollowRepository
	userRepo   repository.UserRepository
}

// NewFollowService constructs a FollowService.
func NewFollowService(followRepo repository.FollowRepository, userRepo repository.UserRepository) FollowService {
	return &followService{followRepo: followRepo, userRepo: userRepo}
}

// Follow creates a follower → followee edge.
func (s *followService) Follow(followerID uint, followeeUsername string) (*FollowResult, error) {
	followee, err := s.userRepo.FindByUsername(followeeUsername)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFollowNotFound
	}
	if err != nil {
		return nil, err
	}

	// Self-follow guard.
	if followerID == followee.ID {
		return nil, ErrSelfFollow
	}

	// Already following? Return current state idempotently.
	_, err = s.followRepo.FindByFollowerAndFollowee(followerID, followee.ID)
	if err == nil {
		return &FollowResult{Following: true, FollowersCount: followee.FollowersCount}, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// Create the follow edge.
	if err := s.followRepo.Create(&models.Follow{
		FollowerID: followerID,
		FolloweeID: followee.ID,
	}); err != nil {
		// Race condition: another request created it first — treat as success.
		if errors.Is(err, repository.ErrDuplicateKey) {
			return &FollowResult{Following: true, FollowersCount: followee.FollowersCount}, nil
		}
		return nil, err
	}

	// Update both counter columns — best-effort.
	_ = s.userRepo.IncrementCounter(followee.ID, "followers_count", 1)
	_ = s.userRepo.IncrementCounter(followerID, "following_count", 1)

	return &FollowResult{Following: true, FollowersCount: followee.FollowersCount + 1}, nil
}

// Unfollow removes the follower → followee edge.
func (s *followService) Unfollow(followerID uint, followeeUsername string) (*FollowResult, error) {
	followee, err := s.userRepo.FindByUsername(followeeUsername)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFollowNotFound
	}
	if err != nil {
		return nil, err
	}

	if followerID == followee.ID {
		return nil, ErrSelfFollow
	}

	// Check if actually following — idempotent if not.
	_, err = s.followRepo.FindByFollowerAndFollowee(followerID, followee.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return &FollowResult{Following: false, FollowersCount: followee.FollowersCount}, nil
	}
	if err != nil {
		return nil, err
	}

	// Remove the edge.
	if err := s.followRepo.Delete(followerID, followee.ID); err != nil {
		return nil, err
	}

	// Update counter columns — best-effort, floor at 0.
	_ = s.userRepo.IncrementCounter(followee.ID, "followers_count", -1)
	_ = s.userRepo.IncrementCounter(followerID, "following_count", -1)

	count := followee.FollowersCount - 1
	if count < 0 {
		count = 0
	}
	return &FollowResult{Following: false, FollowersCount: count}, nil
}

// ListFollowers returns a paginated page of public profiles for users following username.
func (s *followService) ListFollowers(username string, page, pageSize int) (*pagination.Page[PublicProfile], error) {
	user, err := s.userRepo.FindByUsername(username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFollowNotFound
	}
	if err != nil {
		return nil, err
	}

	offset := pagination.Offset(page, pageSize)
	users, total, err := s.followRepo.ListFollowers(user.ID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	profiles := usersToPublicProfiles(users)
	return pagination.New(profiles, total, page, pageSize), nil
}

// ListFollowing returns a paginated page of public profiles for users that username follows.
func (s *followService) ListFollowing(username string, page, pageSize int) (*pagination.Page[PublicProfile], error) {
	user, err := s.userRepo.FindByUsername(username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrFollowNotFound
	}
	if err != nil {
		return nil, err
	}

	offset := pagination.Offset(page, pageSize)
	users, total, err := s.followRepo.ListFollowing(user.ID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	profiles := usersToPublicProfiles(users)
	return pagination.New(profiles, total, page, pageSize), nil
}

// IsFollowing checks whether followerID is following the user with the given username.
func (s *followService) IsFollowing(followerID uint, followeeUsername string) (bool, error) {
	followee, err := s.userRepo.FindByUsername(followeeUsername)
	if errors.Is(err, repository.ErrNotFound) {
		return false, ErrFollowNotFound
	}
	if err != nil {
		return false, err
	}

	_, err = s.followRepo.FindByFollowerAndFollowee(followerID, followee.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ListFollowersByID returns a paginated page of followers for a known userID.
func (s *followService) ListFollowersByID(userID uint, page, pageSize int) (*pagination.Page[PublicProfile], error) {
	offset := pagination.Offset(page, pageSize)
	users, total, err := s.followRepo.ListFollowers(userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	profiles := usersToPublicProfiles(users)
	return pagination.New(profiles, total, page, pageSize), nil
}

// ListFollowingByID returns a paginated page of following for a known userID.
func (s *followService) ListFollowingByID(userID uint, page, pageSize int) (*pagination.Page[PublicProfile], error) {
	offset := pagination.Offset(page, pageSize)
	users, total, err := s.followRepo.ListFollowing(userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	profiles := usersToPublicProfiles(users)
	return pagination.New(profiles, total, page, pageSize), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func usersToPublicProfiles(users []models.User) []PublicProfile {
	profiles := make([]PublicProfile, 0, len(users))
	for i := range users {
		profiles = append(profiles, *toPublicProfile(&users[i]))
	}
	return profiles
}
