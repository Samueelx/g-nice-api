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
// On success returns 201 with a message; no JWT issued yet.
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

// VerifyOTP godoc
// POST /api/v1/auth/verify-otp
// Body: { email, otp }
// Validates the 6-digit code and returns a JWT on success.
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req services.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	resp, err := h.authSvc.VerifyOTP(&req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAlreadyVerified):
			Conflict(c, "email address is already verified")
		case errors.Is(err, services.ErrOTPExpired):
			BadRequest(c, "verification code has expired — please request a new one")
		case errors.Is(err, services.ErrOTPMaxAttempts):
			c.JSON(429, APIResponse{Success: false, Error: "too many incorrect attempts — please request a new code"})
		case errors.Is(err, services.ErrInvalidOTP):
			BadRequest(c, "invalid verification code")
		default:
			log.Printf("verify-otp error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, resp)
}

// ResendOTP godoc
// POST /api/v1/auth/resend-otp
// Body: { email }
// Generates a fresh OTP and resends the verification email.
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req services.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	err := h.authSvc.ResendOTP(&req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAlreadyVerified):
			Conflict(c, "email address is already verified")
		default:
			// Cooldown errors are plain strings from the service.
			// Distinguish them from true internal errors by checking the message.
			log.Printf("resend-otp error: %v", err)
			c.JSON(429, APIResponse{Success: false, Error: err.Error()})
		}
		return
	}

	OK(c, gin.H{"message": "A new verification code has been sent to your email address."})
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
		switch {
		case errors.Is(err, services.ErrInvalidCreds):
			Unauthorized(c, "invalid email or password")
		case errors.Is(err, services.ErrNotVerified):
			c.JSON(403, APIResponse{
				Success: false,
				Error:   "please verify your email address before logging in",
				Message: "check your inbox for the verification code",
			})
		default:
			log.Printf("login error: %v", err)
			InternalError(c)
		}
		return
	}

	OK(c, resp)
}
