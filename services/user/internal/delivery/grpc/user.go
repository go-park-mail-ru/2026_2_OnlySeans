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
	profiles usecase.ProfileUseCase
}

func NewUserHandler(users usecase.UserUseCase, profiles usecase.ProfileUseCase) *UserHandler {
	return &UserHandler{
		users:    users,
		profiles: profiles,
	}
}

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	user := domain.User{
		Email:    req.GetEmail(),
		Username: req.GetUsername(),
	}

	userID, err := h.users.Create(ctx, user, req.GetPassword())
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidPassword):
		return nil, status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return nil, status.Error(codes.AlreadyExists, "email taken")
	case errors.Is(err, domain.ErrUsernameTaken):
		return nil, status.Error(codes.AlreadyExists, "username taken")
	case err != nil:
		return nil, status.Error(codes.Internal, "could not create user")
	}

	return &pb.CreateUserResponse{UserId: userID}, nil
}

func (h *UserHandler) GetByID(ctx context.Context, req *pb.GetByIDRequest) (*pb.GetByIDResponse, error) {
	userID := req.GetUserId()
	if userID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	user, err := h.users.GetByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "could not get user")
	}

	return &pb.GetByIDResponse{User: toProtoUser(user)}, nil
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
