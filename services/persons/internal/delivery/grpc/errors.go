package grpc

import (
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toGRPCError converts use case errors at the gRPC delivery boundary.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrPersonNotFound):
		return status.Error(codes.NotFound, domain.ErrPersonNotFound.Error())
	case errors.Is(err, domain.ErrInvalidFilmID),
		errors.Is(err, domain.ErrInvalidPersonID),
		errors.Is(err, domain.ErrInvalidFirstName),
		errors.Is(err, domain.ErrInvalidLastName),
		errors.Is(err, domain.ErrInvalidBirthDate),
		errors.Is(err, domain.ErrInvalidPagination):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
