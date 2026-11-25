package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
)

func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, repository.ErrCommitUnknown) {
		return status.Error(codes.Internal, "internal server error")
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	}

	for _, mapping := range []struct {
		target error
		code   codes.Code
	}{
		{
			target: domain.ErrFilmNotFound,
			code:   codes.NotFound,
		},
		{
			target: domain.ErrCollectionNotFound,
			code:   codes.NotFound,
		},
		{
			target: domain.ErrRatingNotFound,
			code:   codes.NotFound,
		},
		{
			target: domain.ErrFileNotFound,
			code:   codes.NotFound,
		},
		{
			target: domain.ErrUnauthorized,
			code:   codes.Unauthenticated,
		},
		{
			target: domain.ErrFilmExists,
			code:   codes.AlreadyExists,
		},
		{
			target: domain.ErrCollectionExists,
			code:   codes.AlreadyExists,
		},
		{
			target: domain.ErrDuplicateCollectionFilm,
			code:   codes.AlreadyExists,
		},
		{
			target: domain.ErrInvalidFilmInput,
			code:   codes.InvalidArgument,
		},
		{
			target: domain.ErrInvalidLibraryData,
			code:   codes.InvalidArgument,
		},
		{
			target: domain.ErrInvalidPreferences,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidFilmID,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidCollectionID,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidLimit,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidOffset,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrPosterMissing,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidPosterSize,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrInvalidPosterImage,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrPosterDimensionsTooLarge,
			code:   codes.InvalidArgument,
		},
		{
			target: usecase.ErrPosterTooLarge,
			code:   codes.ResourceExhausted,
		},
	} {
		if errors.Is(err, mapping.target) {
			return status.Error(mapping.code, mapping.target.Error())
		}
	}

	if errors.Is(err, domain.ErrInvalidFilmRating) {
		return status.Error(codes.InvalidArgument, domain.InvalidFilmRatingLimitsError().Error())
	}

	return status.Error(codes.Internal, "internal server error")
}

func missingRequest() error {
	return status.Error(codes.InvalidArgument, "request is required")
}
