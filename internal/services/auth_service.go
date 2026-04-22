package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Samueelx/g-nice-api/internal/email"
	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/Samueelx/g-nice-api/internal/token"
	"golang.org/x/crypto/bcrypt"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Username    string `json:"username"     binding:"required,min=3,max=50,alphanum"`
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required,min=8,max=72"`
	DisplayName string `json:"display_name" binding:"omitempty,max=100"`
}

// RegisterResponse is returned after a successful registration.
// No JWT yet — the user must verify their email first.
type RegisterResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}

// VerifyOTPRequest is the payload for POST /auth/verify-otp.
type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp"   binding:"required,len=6"`
}

// ResendOTPRequest is the payload for POST /auth/resend-otp.
type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest is the payload for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse is returned after a successful OTP verification, login, or refresh.
type AuthResponse struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	User         *models.User `json:"user"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrEmailTaken      = errors.New("email is already in use")
	ErrUsernameTaken   = errors.New("username is already taken")
	ErrInvalidCreds    = errors.New("invalid email or password")
	ErrNotVerified         = errors.New("email address is not verified")
	ErrAlreadyVerified     = errors.New("email address is already verified")
	ErrInvalidOTP          = errors.New("invalid or incorrect verification code")
	ErrOTPExpired          = errors.New("verification code has expired")
	ErrOTPMaxAttempts      = errors.New("too many incorrect attempts — please request a new code")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

const (
	otpExpiry      = 10 * time.Minute
	otpMaxAttempts = 5
	// Cooldown: if an OTP was sent within the last 60 s, block resend.
	otpResendCooldown = 60 * time.Second
)

// ── Interface ─────────────────────────────────────────────────────────────────

// AuthService handles user registration, OTP verification, and login.
type AuthService interface {
	Register(req *RegisterRequest) (*RegisterResponse, error)
	VerifyOTP(req *VerifyOTPRequest) (*AuthResponse, error)
	ResendOTP(req *ResendOTPRequest) error
	Login(req *LoginRequest) (*AuthResponse, error)
	Refresh(req *RefreshRequest) (*AuthResponse, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type authService struct {
	userRepo repository.UserRepository
	tokens   *token.Service
	mailer   email.Sender
}

// NewAuthService constructs an AuthService with all its dependencies.
func NewAuthService(userRepo repository.UserRepository, tokens *token.Service, mailer email.Sender) AuthService {
	return &authService{
		userRepo: userRepo,
		tokens:   tokens,
		mailer:   mailer,
	}
}

// Register creates an unverified user account and sends a 6-digit OTP.
// A JWT is NOT issued here — the client must call VerifyOTP first.
func (s *authService) Register(req *RegisterRequest) (*RegisterResponse, error) {
	emailTaken, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if emailTaken {
		return nil, ErrEmailTaken
	}

	usernameTaken, err := s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if usernameTaken {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	otp, otpHash, err := generateOTP()
	if err != nil {
		return nil, err
	}
	expiry := time.Now().Add(otpExpiry)

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
		OTPHash:      otpHash,
		OTPExpiry:    &expiry,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// Send email — if it fails, the user can resend via /auth/resend-otp.
	if err := s.mailer.SendOTP(user.Email, user.DisplayName, otp); err != nil {
		return nil, fmt.Errorf("send verification email: %w", err)
	}

	return &RegisterResponse{
		Message: "A 6-digit verification code has been sent to your email address.",
		Email:   user.Email,
	}, nil
}

// VerifyOTP validates the 6-digit code, marks the email as verified, and issues a JWT.
func (s *authService) VerifyOTP(req *VerifyOTPRequest) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidOTP // don't leak whether the email exists
	}
	if err != nil {
		return nil, err
	}

	if user.IsEmailVerified {
		return nil, ErrAlreadyVerified
	}

	// Check attempt count before anything else.
	if user.OTPAttempts >= otpMaxAttempts {
		return nil, ErrOTPMaxAttempts
	}

	// Check expiry.
	if user.OTPExpiry == nil || time.Now().After(*user.OTPExpiry) {
		return nil, ErrOTPExpired
	}

	// Validate the code.
	if hashOTP(req.OTP) != user.OTPHash {
		// Increment attempts counter atomically.
		_ = s.userRepo.IncrementCounter(user.ID, "otp_attempts", 1)
		return nil, ErrInvalidOTP
	}

	// Mark verified and clear OTP fields.
	if err := s.userRepo.UpdateFields(user.ID, map[string]interface{}{
		"is_email_verified": true,
		"otp_hash":          "",
		"otp_expiry":        nil,
		"otp_attempts":      0,
	}); err != nil {
		return nil, err
	}

	// Reload to get final state.
	user, err = s.userRepo.FindByID(user.ID)
	if err != nil {
		return nil, err
	}

	t, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	rtPlain, rtHash, err := generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}
	rtExpiry := time.Now().Add(7 * 24 * time.Hour)

	if err := s.userRepo.UpdateFields(user.ID, map[string]interface{}{
		"refresh_token_hash":   rtHash,
		"refresh_token_expiry": rtExpiry,
	}); err != nil {
		return nil, err
	}

	return &AuthResponse{Token: t, RefreshToken: rtPlain, User: user}, nil
}

// ResendOTP generates a fresh OTP and resends the verification email.
// Returns ErrAlreadyVerified if the email is already confirmed.
// Enforces a 60-second cooldown to prevent abuse.
func (s *authService) ResendOTP(req *ResendOTPRequest) error {
	user, err := s.userRepo.FindByEmail(req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		// Return nil to prevent email enumeration.
		return nil
	}
	if err != nil {
		return err
	}

	if user.IsEmailVerified {
		return ErrAlreadyVerified
	}

	// Cooldown: block if the current OTP still has more than (expiry - cooldown) left.
	if user.OTPExpiry != nil {
		remaining := time.Until(*user.OTPExpiry)
		if remaining > otpExpiry-otpResendCooldown {
			// OTP was issued less than 60 seconds ago.
			waitSeconds := int((remaining - (otpExpiry - otpResendCooldown)).Seconds())
			return fmt.Errorf("please wait %d seconds before requesting a new code", waitSeconds)
		}
	}

	otp, otpHash, err := generateOTP()
	if err != nil {
		return err
	}
	expiry := time.Now().Add(otpExpiry)

	if err := s.userRepo.UpdateFields(user.ID, map[string]interface{}{
		"otp_hash":     otpHash,
		"otp_expiry":   expiry,
		"otp_attempts": 0,
	}); err != nil {
		return err
	}

	return s.mailer.SendOTP(user.Email, user.DisplayName, otp)
}

// Login verifies credentials and issues a JWT.
// Returns ErrNotVerified if the account's email has not been confirmed.
func (s *authService) Login(req *LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCreds
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	if !user.IsEmailVerified {
		return nil, ErrNotVerified
	}

	t, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	rtPlain, rtHash, err := generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}
	rtExpiry := time.Now().Add(7 * 24 * time.Hour)

	if err := s.userRepo.UpdateFields(user.ID, map[string]interface{}{
		"refresh_token_hash":   rtHash,
		"refresh_token_expiry": rtExpiry,
	}); err != nil {
		return nil, err
	}

	return &AuthResponse{Token: t, RefreshToken: rtPlain, User: user}, nil
}

// Refresh validates a refresh token and issues a new access token and refresh token.
func (s *authService) Refresh(req *RefreshRequest) (*AuthResponse, error) {
	// Parse the userID from the token: "<userID>:<randomHex>"
	parts := strings.Split(req.RefreshToken, ":")
	if len(parts) != 2 {
		return nil, ErrInvalidRefreshToken
	}

	userIDInt, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	userID := uint(userIDInt)

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	// Validate expiry
	if user.RefreshTokenExpiry == nil || time.Now().After(*user.RefreshTokenExpiry) {
		return nil, ErrInvalidRefreshToken
	}

	// Validate hash
	hash := sha256.Sum256([]byte(req.RefreshToken))
	hashStr := hex.EncodeToString(hash[:])
	if hashStr != user.RefreshTokenHash {
		return nil, ErrInvalidRefreshToken
	}

	// Token is valid, rotate it
	t, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	rtPlain, rtHash, err := generateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}
	rtExpiry := time.Now().Add(7 * 24 * time.Hour)

	if err := s.userRepo.UpdateFields(user.ID, map[string]interface{}{
		"refresh_token_hash":   rtHash,
		"refresh_token_expiry": rtExpiry,
	}); err != nil {
		return nil, err
	}

	return &AuthResponse{Token: t, RefreshToken: rtPlain, User: user}, nil
}

// ── OTP helpers ───────────────────────────────────────────────────────────────

// generateOTP returns a cryptographically random 6-digit code and its SHA-256 hash.
func generateOTP() (plaintext, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", "", fmt.Errorf("generate otp: %w", err)
	}
	plaintext = fmt.Sprintf("%06d", n.Int64())
	hash = hashOTP(plaintext)
	return plaintext, hash, nil
}

// hashOTP returns the hex-encoded SHA-256 hash of a plaintext OTP.
func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// generateRefreshToken returns a secure opaque token (prefixed with userID) and its hash.
func generateRefreshToken(userID uint) (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	randomHex := hex.EncodeToString(b)
	plaintext := fmt.Sprintf("%d:%s", userID, randomHex)
	
	// Hash the entire string
	hash := sha256.Sum256([]byte(plaintext))
	return plaintext, hex.EncodeToString(hash[:]), nil
}
