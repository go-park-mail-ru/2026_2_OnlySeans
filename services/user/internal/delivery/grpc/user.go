package grpc

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	users    usecase.UserUseCase
	profiles *usecase.ProfileUseCase
}

func NewUserHandler(users usecase.UserUseCase, profiles *usecase.ProfileUseCase) *UserHandler {
	return &UserHandler{
		users:    users,
		profiles: profiles,
	}
}

func (h *UserHandler) GetByEmail(ctx context.Context, req *pb.GetByEmailRequest) (*pb.GetByEmailResponse, error) {
	if req.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	user, err := h.users.GetByEmail(ctx, req.GetEmail())
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "could not get user")
	}

	return &pb.GetByEmailResponse{
		UserId:       user.ID,
		Role:         user.Role,
		PasswordHash: user.PasswordHash,
	}, nil
}
