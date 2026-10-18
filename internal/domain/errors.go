package domain

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrCapacity          = errors.New("insufficient ticket capacity")
	ErrInvalidInput      = errors.New("invalid input")
	ErrDuplicateEmail    = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrForbidden         = errors.New("forbidden: insufficient role")
)
