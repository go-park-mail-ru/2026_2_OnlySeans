package reviews

import "errors"

var (
	ErrNilRepo  = errors.New("review repository is nil")
	ErrNilFilms = errors.New("film getter is nil")

	ErrInvalidTitle   = errors.New("title is invalid")
	ErrInvalidContent = errors.New("content is invalid")

	ErrReviewNotFound = errors.New("review not found")
	ErrReviewExists   = errors.New("review already exists")
	ErrForbidden      = errors.New("access forbidden")

	ErrNilUsecase = errors.New("usecase is nil")

	ErrInvalidID     = errors.New("id must be a positive integer")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("offset must be a non-negative integer")
	ErrInvalidBody   = errors.New("invalid request body")
)
