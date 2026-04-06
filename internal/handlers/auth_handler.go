package handlers

import (
	"errors"
	"log"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	authSvc services.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authSvc services.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register godoc
// POST /api/v1/auth/register
// Body: { username, email, password, display_name? }
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	resp, err := h.authSvc.Register(&req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailTaken):
			Conflict(c, "email is already in use")
		case errors.Is(err, services.ErrUsernameTaken):
			Conflict(c, "username is already taken")
		default:
			log.Printf("register error: %v", err)
			InternalError(c)
		}
		return
	}

	Created(c, resp)
}

// Login godoc
// POST /api/v1/auth/login
// Body: { email, password }
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	resp, err := h.authSvc.Login(&req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCreds) {
			Unauthorized(c, "invalid email or password")
			return
		}
		log.Printf("login error: %v", err)
		InternalError(c)
		return
	}

	OK(c, resp)
}
