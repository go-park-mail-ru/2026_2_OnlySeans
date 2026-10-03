package auth

import "errors"

var (
	ErrNilUseCase         = errors.New("usecase cannot be nil")
	ErrNilRepo            = errors.New("repository cannot be nil")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInternal           = errors.New("internal server error")
	ErrUserExists         = errors.New("user with this email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrNilSessionStore    = errors.New("session store cannot be nil")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrUnauthorized       = errors.New("unauthorized")
)
