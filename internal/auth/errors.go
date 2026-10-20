package auth

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"

var (
	ErrNilRepo            = domain.ErrNilRepo
	ErrInvalidEmail       = domain.ErrInvalidEmail
	ErrInvalidUsername    = domain.ErrInvalidUsername
	ErrInvalidPassword    = domain.ErrInvalidPassword
	ErrInvalidCredentials = domain.ErrInvalidCredentials
	ErrInternal           = domain.ErrInternal
	ErrUserExists         = domain.ErrUserExists
	ErrUserNotFound       = domain.ErrUserNotFound
	ErrNilSessionStore    = domain.ErrNilSessionStore
	ErrSessionNotFound    = domain.ErrSessionNotFound
	ErrUnauthorized       = domain.ErrUnauthorized
)
