package domain

import "errors"

var (
	ErrUserExists    = errors.New("user with this email already exists")
	ErrUsernameTaken = errors.New("user with this username already exists")
	ErrUserNotFound  = errors.New("user not found")
)
