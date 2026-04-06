package repository

import "errors"

// Sentinel errors returned by all repository implementations.
// Handlers and services should use errors.Is() to inspect these.
var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicateKey  = errors.New("duplicate key violation")
)
