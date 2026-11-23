package domain

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("user with this email already exists")
	ErrUserExists         = ErrEmailAlreadyExists
	ErrUsernameTaken      = errors.New("user with this username already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidImage       = errors.New("invalid image")
	ErrAvatarNotFound     = errors.New("avatar not found")
	ErrCommitUnknown      = errors.New("transaction commit outcome is unknown")
)
