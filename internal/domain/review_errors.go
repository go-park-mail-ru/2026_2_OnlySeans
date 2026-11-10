package domain

import "errors"

var (
	ErrReviewNotFound  = errors.New("review not found")
	ErrReviewExists    = errors.New("review already exists")
	ErrForbidden       = errors.New("access forbidden")
	ErrInvalidTitle    = errors.New("title is invalid")
	ErrInvalidContent  = errors.New("content is invalid")
	ErrNilReviewsRepo  = errors.New("reviews repository is nil")
	ErrVersionConflict = errors.New("review was changed by someone else")
	ErrInvalidVersion  = errors.New("version is invalid")
)
