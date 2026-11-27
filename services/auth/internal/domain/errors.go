package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found or expired")
	ErrSessionExpired     = errors.New("session expired")
	ErrUserNotFound       = errors.New("user not found")
)
