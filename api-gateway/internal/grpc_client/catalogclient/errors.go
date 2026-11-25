package catalogclient

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrFilmNotFound             = errors.New("film not found")
	ErrCollectionNotFound       = errors.New("collection not found")
	ErrRatingNotFound           = errors.New("rating not found")
	ErrFileNotFound             = errors.New("file not found")
	ErrUnauthorized             = errors.New("unauthorized")
	ErrForbidden                = errors.New("forbidden")
	ErrInvalidFilmInput         = errors.New("invalid film input")
	ErrInvalidFilmRating        = errors.New("invalid film rating")
	ErrInvalidLibraryData       = errors.New("invalid library data")
	ErrInvalidPreferences       = errors.New("invalid preferences")
	ErrInvalidArgument          = errors.New("invalid argument")
	ErrInternal                 = errors.New("internal server error")
	ErrInvalidPosterFilmID      = errors.New("film id must be a positive integer")
	ErrPosterMissing            = errors.New("poster file is required")
	ErrInvalidPosterSize        = errors.New("poster size must be positive")
	ErrPosterTooLarge           = errors.New("poster file is too large")
	ErrInvalidPosterImage       = errors.New("poster must be a valid JPEG or PNG image")
	ErrPosterDimensionsTooLarge = errors.New("poster dimensions exceed the allowed limit")
)

type rpcError struct {
	target error
	detail *status.Status
}

func (e *rpcError) Error() string {
	return e.detail.Message()
}

func (e *rpcError) Unwrap() error {
	return e.target
}

func (e *rpcError) GRPCStatus() *status.Status {
	return e.detail
}

func fromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	detail := status.Convert(err)
	var target error

	switch detail.Code() {
	case codes.NotFound:
		for _, candidate := range []error{ErrFilmNotFound, ErrCollectionNotFound, ErrRatingNotFound, ErrFileNotFound} {
			if detail.Message() == candidate.Error() {
				target = candidate
				break
			}
		}
	case codes.InvalidArgument:
		target = ErrInvalidArgument

		for _, candidate := range []error{ErrInvalidFilmInput, ErrInvalidFilmRating, ErrInvalidLibraryData, ErrInvalidPreferences, ErrInvalidPosterFilmID, ErrPosterMissing, ErrInvalidPosterSize, ErrInvalidPosterImage, ErrPosterDimensionsTooLarge} {
			if detail.Message() == candidate.Error() || strings.HasPrefix(detail.Message(), candidate.Error()+": ") {
				target = candidate
				break
			}
		}
	case codes.ResourceExhausted:
		return ErrPosterTooLarge
	case codes.Unauthenticated:
		target = ErrUnauthorized
	case codes.PermissionDenied:
		target = ErrForbidden
	}

	if target == nil {
		return ErrInternal
	}

	return &rpcError{
		target: target,
		detail: detail,
	}
}
