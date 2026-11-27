package grpc_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usergrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/mocks"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name string
		id   int64
		err  error
		code codes.Code
	}{
		{
			name: "success",
			id:   7,
			code: codes.OK,
		},
		{
			name: "invalid ID",
			code: codes.InvalidArgument,
		},
		{
			name: "missing user",
			id:   7,
			err:  fmt.Errorf("profile: %w", domain.ErrUserNotFound),
			code: codes.NotFound,
		},
		{
			name: "storage failure",
			id:   7,
			err:  errors.New("private storage details"),
			code: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profiles := mocks.NewMockProfileUseCase(gomock.NewController(t))
			profile := &domain.Profile{
				User: domain.User{
					ID:        7,
					Username:  "reader",
					CreatedAt: time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC),
				},
				Reviews: 3,
				Favorites: []domain.ProfileFilm{
					{
						ID:    42,
						Title: "Film",
					},
				},
			}

			if tt.id > 0 {
				profiles.EXPECT().GetProfile(t.Context(), tt.id).Return(profile, tt.err)
			}

			handler := usergrpc.NewUserHandler(nil, profiles)
			response, err := handler.GetProfile(t.Context(), &pb.GetProfileRequest{
				UserId: tt.id,
			})
			require.Equal(t, tt.code, status.Code(err))

			if tt.code != codes.OK {
				require.Nil(t, response)

				if tt.code == codes.Internal {
					require.Equal(t, "could not get profile", status.Convert(err).Message())
				}

				return
			}

			require.NoError(t, err)
			require.Equal(t, profile.User.ID, response.User.Id)
			require.Equal(t, profile.User.Username, response.User.Username)
			require.Equal(t, profile.User.CreatedAt, response.CreatedAt.AsTime())
			require.Equal(t, profile.Reviews, response.Reviews)
			require.Len(t, response.Favorites, 1)
			require.Equal(t, profile.Favorites[0].ID, response.Favorites[0].Id)
			require.Equal(t, profile.Favorites[0].Title, response.Favorites[0].Title)
		})
	}
}
