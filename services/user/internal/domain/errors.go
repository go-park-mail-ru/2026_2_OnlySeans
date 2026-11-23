package domain

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("user with this email already exists")
	ErrUserExists         = ErrEmailAlreadyExists
	ErrUsernameTaken      = errors.New("user with this username already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidImage       = errors.New("invalid image")
	ErrAvatarNotFound     = errors.New("avatar not found")
	ErrInvalidAvatarName  = errors.New("invalid avatar filename")
	ErrAvatarTooLarge     = errors.New("avatar file is too large")
	ErrCommitUnknown      = errors.New("transaction commit outcome is unknown")
)
