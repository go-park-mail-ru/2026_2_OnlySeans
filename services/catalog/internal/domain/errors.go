package domain

import "errors"

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrFileNotFound = errors.New("file not found")
)
