package domain

import "errors"

var (
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrSessionNotFound     = errors.New("session not found or expired")
	ErrSessionExpired      = errors.New("session expired")
)
