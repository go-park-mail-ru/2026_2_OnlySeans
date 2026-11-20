package reviewsgrpc

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// разные ошибки с одним gRPC-кодом сохраняют свои сообщения и HTTP-статусы
var reviewErrors = []struct {
	cause error
	code  codes.Code
}{
	{domain.ErrInvalidTitle, codes.InvalidArgument},
	{domain.ErrInvalidContent, codes.InvalidArgument},
	{domain.ErrInvalidVersion, codes.InvalidArgument},
	{domain.ErrFilmNotFound, codes.NotFound},
	{domain.ErrReviewNotFound, codes.NotFound},
	{domain.ErrReviewExists, codes.AlreadyExists},
	{domain.ErrVersionConflict, codes.Aborted},
	{domain.ErrUnauthorized, codes.Unauthenticated},
	{domain.ErrForbidden, codes.PermissionDenied},
}

func rpcError(err error) error {
	for _, mapped := range reviewErrors {
		if errors.Is(err, mapped.cause) {
			return status.Error(mapped.code, mapped.cause.Error())
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}
	return status.Error(codes.Internal, "reviews operation failed")
}

func domainError(err error) error {
	rpcStatus, ok := status.FromError(err)
	if !ok {
		return err
	}
	for _, mapped := range reviewErrors {
		if rpcStatus.Code() == mapped.code && rpcStatus.Message() == mapped.cause.Error() {
			return mapped.cause
		}
	}
	return err
}
