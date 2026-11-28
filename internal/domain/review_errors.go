package domain

import "errors"

var (
	ErrReviewNotFound  = errors.New("review not found")
	ErrReviewExists    = errors.New("review already exists")
	ErrForbidden       = errors.New("access forbidden")
	ErrInvalidTitle    = errors.New("title is invalid")
	ErrInvalidContent  = errors.New("content is invalid")
	ErrNilReviewsRepo  = errors.New("reviews repository is nil")
	ErrNilReviewPolicy = errors.New("review policy provider is nil")
	ErrVersionConflict = errors.New("review was changed by someone else")
	ErrInvalidVersion  = errors.New("version is invalid")
	ErrFeatureDisabled = errors.New("feature disabled")
	ErrReviewTooLong   = errors.New("review content is too long")
)

type ReviewTooLongError struct {
	MaxLength int
	Revision  int64
}

func (e *ReviewTooLongError) Error() string {
	return ErrReviewTooLong.Error()
}

func (e *ReviewTooLongError) Unwrap() error {
	return ErrReviewTooLong
}
