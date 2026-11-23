package reviewsclient

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
