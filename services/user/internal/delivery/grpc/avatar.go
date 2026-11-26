package grpc

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *UserHandler) GetAvatarURL(ctx context.Context, req *pb.GetAvatarURLRequest) (*pb.GetAvatarURLResponse, error) {
	userID := req.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	url, err := h.users.GetAvatarURL(ctx, userID)
	if err != nil {
		return nil, avatarRPCError(err, "could not get avatar URL")
	}

	return &pb.GetAvatarURLResponse{Url: url}, nil
}

func (h *UserHandler) GetAvatarURLByFilename(ctx context.Context, req *pb.GetAvatarURLByFilenameRequest) (*pb.GetAvatarURLResponse, error) {
	url, err := h.users.GetAvatarURLByFilename(ctx, req.GetFilename())
	if err != nil {
		return nil, avatarRPCError(err, "could not get avatar URL")
	}

	return &pb.GetAvatarURLResponse{Url: url}, nil
}

func (h *UserHandler) UpdateAvatar(ctx context.Context, req *pb.UpdateAvatarRequest) (*pb.UpdateAvatarResponse, error) {
	userID := req.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	imageData := req.GetImageData()
	if len(imageData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "image data is required")
	}

	filename, err := h.users.UpdateAvatar(ctx, userID, imageData)
	if err != nil {
		return nil, avatarRPCError(err, "could not update avatar")
	}

	return &pb.UpdateAvatarResponse{AvatarFilename: filename}, nil
}

func (h *UserHandler) DeleteAvatar(ctx context.Context, req *pb.DeleteAvatarRequest) (*pb.DeleteAvatarResponse, error) {
	userID := req.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := h.users.DeleteAvatar(ctx, userID); err != nil {
		return nil, avatarRPCError(err, "could not delete avatar")
	}

	return &pb.DeleteAvatarResponse{}, nil
}

func avatarRPCError(err error, internalMessage string) error {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, domain.ErrAvatarNotFound):
		return status.Error(codes.NotFound, "avatar not found")
	case errors.Is(err, domain.ErrInvalidAvatarName):
		return status.Error(codes.InvalidArgument, "invalid avatar filename")
	case errors.Is(err, domain.ErrAvatarTooLarge):
		return status.Error(codes.ResourceExhausted, "avatar file is too large")
	case errors.Is(err, domain.ErrInvalidImage):
		return status.Error(codes.InvalidArgument, "invalid image")
	default:
		return status.Error(codes.Internal, internalMessage)
	}
}
