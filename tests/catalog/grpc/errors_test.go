package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cataloggrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
)

func TestToGRPCError(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		code    codes.Code
		message string
	}{
		{
			name: "success",
			code: codes.OK,
		},
		{
			name:    "missing film",
			err:     domain.ErrFilmNotFound,
			code:    codes.NotFound,
			message: domain.ErrFilmNotFound.Error(),
		},
		{
			name:    "wrapped missing collection",
			err:     fmt.Errorf("read: %w", domain.ErrCollectionNotFound),
			code:    codes.NotFound,
			message: domain.ErrCollectionNotFound.Error(),
		},
		{
			name:    "missing rating",
			err:     domain.ErrRatingNotFound,
			code:    codes.NotFound,
			message: domain.ErrRatingNotFound.Error(),
		},
		{
			name:    "missing file",
			err:     domain.ErrFileNotFound,
			code:    codes.NotFound,
			message: domain.ErrFileNotFound.Error(),
		},
		{
			name:    "unauthorized",
			err:     domain.ErrUnauthorized,
			code:    codes.Unauthenticated,
			message: domain.ErrUnauthorized.Error(),
		},
		{
			name:    "film conflict",
			err:     domain.ErrFilmExists,
			code:    codes.AlreadyExists,
			message: domain.ErrFilmExists.Error(),
		},
		{
			name:    "collection conflict",
			err:     domain.ErrCollectionExists,
			code:    codes.AlreadyExists,
			message: domain.ErrCollectionExists.Error(),
		},
		{
			name:    "duplicate collection film",
			err:     domain.ErrDuplicateCollectionFilm,
			code:    codes.AlreadyExists,
			message: domain.ErrDuplicateCollectionFilm.Error(),
		},
		{
			name:    "invalid film",
			err:     domain.ErrInvalidFilmInput,
			code:    codes.InvalidArgument,
			message: domain.ErrInvalidFilmInput.Error(),
		},
		{
			name:    "invalid library",
			err:     domain.ErrInvalidLibraryData,
			code:    codes.InvalidArgument,
			message: domain.ErrInvalidLibraryData.Error(),
		},
		{
			name:    "invalid preferences",
			err:     domain.ErrInvalidPreferences,
			code:    codes.InvalidArgument,
			message: domain.ErrInvalidPreferences.Error(),
		},
		{
			name:    "invalid film id",
			err:     usecase.ErrInvalidFilmID,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidFilmID.Error(),
		},
		{
			name:    "invalid collection id",
			err:     usecase.ErrInvalidCollectionID,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidCollectionID.Error(),
		},
		{
			name:    "invalid limit",
			err:     usecase.ErrInvalidLimit,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidLimit.Error(),
		},
		{
			name:    "invalid offset",
			err:     usecase.ErrInvalidOffset,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidOffset.Error(),
		},
		{
			name:    "missing poster",
			err:     usecase.ErrPosterMissing,
			code:    codes.InvalidArgument,
			message: usecase.ErrPosterMissing.Error(),
		},
		{
			name:    "invalid size",
			err:     usecase.ErrInvalidPosterSize,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidPosterSize.Error(),
		},
		{
			name:    "invalid image",
			err:     usecase.ErrInvalidPosterImage,
			code:    codes.InvalidArgument,
			message: usecase.ErrInvalidPosterImage.Error(),
		},
		{
			name:    "dimensions",
			err:     usecase.ErrPosterDimensionsTooLarge,
			code:    codes.InvalidArgument,
			message: usecase.ErrPosterDimensionsTooLarge.Error(),
		},
		{
			name:    "file size",
			err:     usecase.ErrPosterTooLarge,
			code:    codes.ResourceExhausted,
			message: usecase.ErrPosterTooLarge.Error(),
		},
		{
			name:    "rating limits",
			err:     domain.InvalidFilmRatingLimitsError(),
			code:    codes.InvalidArgument,
			message: domain.InvalidFilmRatingLimitsError().Error(),
		},
		{
			name:    "canceled",
			err:     fmt.Errorf("query: %w", context.Canceled),
			code:    codes.Canceled,
			message: "request canceled",
		},
		{
			name:    "deadline",
			err:     context.DeadlineExceeded,
			code:    codes.DeadlineExceeded,
			message: "request deadline exceeded",
		},
		{
			name:    "unknown commit overrides cancellation",
			err:     errors.Join(repository.ErrCommitUnknown, context.Canceled),
			code:    codes.Internal,
			message: "internal server error",
		},
		{
			name:    "database details redacted",
			err:     errors.New("postgres://private-host SQL INSERT password"),
			code:    codes.Internal,
			message: "internal server error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &catalogStub{
				err: tc.err,
			}
			handler := cataloggrpc.NewCatalogHandler(stub, nil, nil, nil, nil, nil)
			_, err := handler.GetFilm(t.Context(), &pb.GetFilmRequest{
				FilmId: 42,
			})
			require.Equal(t, tc.code, status.Code(err))
			require.Equal(t, tc.message, status.Convert(err).Message())
		})
	}
}
