package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultExpiry = 15 * time.Minute // 15 minutes for access token

// Claims is the JWT payload shared between token generation and middleware validation.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service handles JWT generation and parsing.
// It is constructed once in main and injected wherever needed.
type Service struct {
	secret []byte
	expiry time.Duration
}

// New constructs a token.Service from a plain secret string.
func New(secret string) *Service {
	return &Service{
		secret: []byte(secret),
		expiry: defaultExpiry,
	}
}

// Generate signs a new JWT containing the user's ID and email.
func (s *Service) Generate(userID uint, email string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

// Parse validates a raw token string and returns the embedded Claims.
func (s *Service) Parse(raw string) (*Claims, error) {
	raw = strings.TrimPrefix(raw, "Bearer ")

	claims := &Claims{}
	t, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("invalid or expired token")
	}
	return claims, nil
}
