package domain

import "errors"

var (
	ErrInvalidSearchQuery = errors.New("q must be from 2 to 200 characters")
	ErrInvalidSearchLimit = errors.New("limit must be an integer from 1 to 5")
)
