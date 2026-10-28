package domain

import (
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/pkg/utils"
)

var (
	ErrNilRepo               = errors.New("repository cannot be nil")
	ErrInvalidEmail          = utils.ErrInvalidEmail
	ErrInvalidUsername       = utils.ErrInvalidUsername
	ErrInvalidPassword       = utils.ErrInvalidPassword
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrInternal              = errors.New("internal server error")
	ErrUserExists            = errors.New("user with this email already exists")
	ErrUsernameTaken         = errors.New("user with this username already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrNilSessionStore       = errors.New("session store cannot be nil")
	ErrSessionNotFound       = errors.New("session not found or expired")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrInvalidPasswordHash   = errors.New("invalid password hash")
	ErrInvalidUserRepo       = errors.New("invalid user repository")
	ErrInvalidSessionStoe    = errors.New("invalid session store")
	ErrInvalidPasswordVer    = errors.New("Invalid passworf verifyer")
	ErrInvalidPasswordHasher = errors.New("Invalid passworf hasher")
)
