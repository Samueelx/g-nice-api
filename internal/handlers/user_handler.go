package handlers

import (
	"errors"
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests for user profile operations.
type UserHandler struct {
	userSvc services.UserService
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(userSvc services.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// GetMe godoc
// GET /api/v1/users/me
// Returns the full profile of the authenticated user (including email).
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	user, err := h.userSvc.GetMe(userID)
	if err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("GetMe error: %v", err)
		InternalError(c)
		return
	}

	OK(c, user)
}

// UpdateMe godoc
// PATCH /api/v1/users/me
// Body: { display_name?, bio?, avatar_url?, is_private? }
// Only provided (non-null) fields are written to the database.
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req services.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	user, err := h.userSvc.UpdateProfile(userID, &req)
	if err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("UpdateMe error: %v", err)
		InternalError(c)
		return
	}

	OK(c, user)
}

// GetUserByUsername godoc
// GET /api/v1/users/:username
// Returns the public profile of any user (no email exposed).
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		BadRequest(c, "username is required")
		return
	}

	profile, err := h.userSvc.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, services.ErrProfileNotFound) {
			NotFound(c, "user not found")
			return
		}
		log.Printf("GetUserByUsername error: %v", err)
		InternalError(c)
		return
	}

	OK(c, profile)
}


