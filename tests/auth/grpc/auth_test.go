package grpc_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
)

func TestLogin(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "success",
			code: codes.OK,
		},
		{
			name: "invalid credentials",
			err:  fmt.Errorf("login: %w", domain.ErrInvalidCredentials),
			code: codes.Unauthenticated,
		},
		{
			name: "invalid email",
			err:  utils.ErrInvalidEmail,
			code: codes.InvalidArgument,
		},
		{
			name: "invalid password",
			err:  utils.ErrInvalidPassword,
			code: codes.InvalidArgument,
		},
		{
			name: "internal failure",
			err:  errors.New("service unavailable"),
			code: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := mocks.NewMockAuthUseCase(gomock.NewController(t))
			session := &domain.Session{
				ID:        uuid.New(),
				UserID:    7,
				Role:      "reader",
				ExpiresAt: time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC),
			}
			uc.EXPECT().Login(t.Context(), "reader@example.com", "Password1").Return(session, tt.err)
			handler := authgrpc.NewAuthHandler(uc)
			response, err := handler.Login(t.Context(), &pb.LoginRequest{
				Email:    "reader@example.com",
				Password: "Password1",
			})
			require.Equal(t, tt.code, status.Code(err))

			if tt.err != nil {
				require.Nil(t, response)
				return
			}

			require.NoError(t, err)
			require.Equal(t, session.ID.String(), response.Session.Id)
			require.Equal(t, int64(session.UserID), response.Session.UserId)
			require.Equal(t, session.Role, response.Session.Role)
			require.Equal(t, session.ExpiresAt, response.Session.ExpiresAt.AsTime())
		})
	}
}
