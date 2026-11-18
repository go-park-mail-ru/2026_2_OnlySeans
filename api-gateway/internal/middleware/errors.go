package middleware

import "errors"

var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrInternal        = errors.New("internal server error")
	ErrSessionNotFound = errors.New("session not found")
)
