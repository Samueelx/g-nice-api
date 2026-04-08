package handlers

import (
	"errors"
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// FollowHandler handles HTTP requests for follow/unfollow and follow list operations.
type FollowHandler struct {
	followSvc services.FollowService
}

// NewFollowHandler constructs a FollowHandler.
func NewFollowHandler(followSvc services.FollowService) *FollowHandler {
	return &FollowHandler{followSvc: followSvc}
}

// Follow godoc
// POST /api/v1/users/:username/follow
// Follows the specified user. Idempotent if already following.
// Response: { following: true, followers_count: N }
func (h *FollowHandler) Follow(c *gin.Context) {
	followerID, ok := extractUserID(c)
	if !ok {
		return
	}
	username := c.Param("username")

	result, err := h.followSvc.Follow(followerID, username)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSelfFollow):
			BadRequest(c, "you cannot follow yourself")
		case errors.Is(err, services.ErrFollowNotFound):
			NotFound(c, "user not found")
		default:
			log.Printf("Follow error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, result)
}

// Unfollow godoc
// DELETE /api/v1/users/:username/follow
// Unfollows the specified user. Idempotent if not currently following.
// Response: { following: false, followers_count: N }
func (h *FollowHandler) Unfollow(c *gin.Context) {
	followerID, ok := extractUserID(c)
	if !ok {
		return
	}
	username := c.Param("username")

	result, err := h.followSvc.Unfollow(followerID, username)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSelfFollow):
			BadRequest(c, "you cannot unfollow yourself")
		case errors.Is(err, services.ErrFollowNotFound):
			NotFound(c, "user not found")
		default:
			log.Printf("Unfollow error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, result)
}

// ListFollowers godoc
// GET /api/v1/users/:username/followers?page=1&page_size=20
// Returns a paginated list of public profiles for users who follow :username.
func (h *FollowHandler) ListFollowers(c *gin.Context) {
	username := c.Param("username")
	page, pageSize := parsePagination(c)

	result, err := h.followSvc.ListFollowers(username, page, pageSize)
	if err != nil {
		if errors.Is(err, services.ErrFollowNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("ListFollowers error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// ListFollowing godoc
// GET /api/v1/users/:username/following?page=1&page_size=20
// Returns a paginated list of public profiles for users that :username follows.
func (h *FollowHandler) ListFollowing(c *gin.Context) {
	username := c.Param("username")
	page, pageSize := parsePagination(c)

	result, err := h.followSvc.ListFollowing(username, page, pageSize)
	if err != nil {
		if errors.Is(err, services.ErrFollowNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("ListFollowing error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// CheckFollowing godoc
// GET /api/v1/users/:username/follow
// Returns whether the authenticated user is currently following :username.
// Used by the frontend to sync follow button state on page load.
func (h *FollowHandler) CheckFollowing(c *gin.Context) {
	followerID, ok := extractUserID(c)
	if !ok {
		return
	}
	username := c.Param("username")

	following, err := h.followSvc.IsFollowing(followerID, username)
	if err != nil {
		if errors.Is(err, services.ErrFollowNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("CheckFollowing error: %v", err)
		InternalError(c)
		return
	}

	OK(c, gin.H{"following": following})
}

// GetMyFollowers godoc
// GET /api/v1/users/me/followers?page=1&page_size=20
// Returns the authenticated user's followers using their userID (no username lookup).
func (h *FollowHandler) GetMyFollowers(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)

	result, err := h.followSvc.ListFollowersByID(userID, page, pageSize)
	if err != nil {
		log.Printf("GetMyFollowers error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// GetMyFollowing godoc
// GET /api/v1/users/me/following?page=1&page_size=20
// Returns the list of users the authenticated user follows.
func (h *FollowHandler) GetMyFollowing(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)

	result, err := h.followSvc.ListFollowingByID(userID, page, pageSize)
	if err != nil {
		log.Printf("GetMyFollowing error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

