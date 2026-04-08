package repository

import (
	"errors"
	"strings"
)

// Sentinel errors returned by all repository implementations.
// Handlers and services should use errors.Is() to inspect these.
var (
	ErrNotFound     = errors.New("record not found")
	ErrDuplicateKey = errors.New("duplicate key violation")
)

// isDuplicateKeyError detects PostgreSQL unique constraint violations.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "UNIQUE constraint failed") // SQLite (for tests)
}

