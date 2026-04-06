package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/repository"
	"github.com/Samueelx/g-nice-api/internal/token"
	"golang.org/x/crypto/bcrypt"
)

// ── Request / Response DTOs ───────────────────────────────────────────────────

// RegisterRequest is the payload accepted by POST /auth/register.
type RegisterRequest struct {
	Username    string `json:"username"     binding:"required,min=3,max=50,alphanum"`
	Email       string `json:"email"        binding:"required,email"`
	Password    string `json:"password"     binding:"required,min=8,max=72"`
	DisplayName string `json:"display_name" binding:"omitempty,max=100"`
}

// LoginRequest is the payload accepted by POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is returned after a successful register or login.
type AuthResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrEmailTaken    = errors.New("email is already in use")
	ErrUsernameTaken = errors.New("username is already taken")
	ErrInvalidCreds  = errors.New("invalid email or password")
)

// ── Interface + implementation ────────────────────────────────────────────────

// AuthService handles user registration and login.
type AuthService interface {
	Register(req *RegisterRequest) (*AuthResponse, error)
	Login(req *LoginRequest) (*AuthResponse, error)
}

type authService struct {
	userRepo repository.UserRepository
	tokens   *token.Service
}

// NewAuthService constructs an AuthService with its dependencies.
func NewAuthService(userRepo repository.UserRepository, tokens *token.Service) AuthService {
	return &authService{
		userRepo: userRepo,
		tokens:   tokens,
	}
}

// Register creates a new user account and returns a signed JWT.
func (s *authService) Register(req *RegisterRequest) (*AuthResponse, error) {
	// Uniqueness checks
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

	// Hash password (bcrypt cost 12 — good balance of security vs. latency)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, err
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  displayName,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	t, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: t, User: user}, nil
}

// Login verifies credentials and returns a signed JWT on success.
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

	t, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: t, User: user}, nil
}
