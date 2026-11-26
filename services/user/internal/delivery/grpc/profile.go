package grpc

import (
	"context"
	"errors"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *UserHandler) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	profile, err := h.profiles.GetProfile(ctx, req.GetUserId())
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "could not get profile")
	}
	return &pb.GetProfileResponse{
		User:      toProtoUser(profile.User),
		CreatedAt: timestamppb.New(profile.User.CreatedAt),
		Favorites: toProtoProfileFilms(profile.Favorites),
		Ratings:   toProtoProfileFilms(profile.Ratings),
		Reviews:   profile.Reviews,
	}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	if req.Username == nil && req.Email == nil {
		return nil, status.Error(codes.InvalidArgument, "email or username is required")
	}

	user, err := h.users.UpdateProfile(ctx, req.GetUserId(), req.Username, req.Email)
	switch {
	case errors.Is(err, domain.ErrInvalidEmail):
		return nil, status.Error(codes.InvalidArgument, "invalid email")
	case errors.Is(err, domain.ErrInvalidUsername):
		return nil, status.Error(codes.InvalidArgument, "invalid username")
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		return nil, status.Error(codes.AlreadyExists, "email taken")
	case errors.Is(err, domain.ErrUsernameTaken):
		return nil, status.Error(codes.AlreadyExists, "username taken")
	case errors.Is(err, domain.ErrUserNotFound):
		return nil, status.Error(codes.NotFound, "user not found")
	case err != nil:
		return nil, status.Error(codes.Internal, "could not update profile")
	}
	return &pb.UpdateProfileResponse{User: toProtoUser(user)}, nil
}

func toProtoUser(user domain.User) *pb.User {
	avatarFilename := user.AvatarFilename
	if avatarFilename != nil {
		filename := strings.TrimPrefix(*avatarFilename, "avatars/")
		avatarFilename = &filename
	}

	return &pb.User{
		Id:             user.ID,
		Email:          user.Email,
		Username:       user.Username,
		Role:           user.Role,
		AvatarFilename: avatarFilename,
	}
}

func toProtoProfileFilms(films []domain.ProfileFilm) []*pb.ProfileFilm {
	result := make([]*pb.ProfileFilm, len(films))
	for i, film := range films {
		result[i] = &pb.ProfileFilm{
			Id:       film.ID,
			Title:    film.Title,
			Year:     film.Year,
			Type:     film.Type,
			AgeLimit: film.AgeLimit,
			Genres:   film.Genres,
			Poster:   film.PosterURL,
			Average:  film.Average,
			Score:    film.Score,
		}
	}
	return result
}
