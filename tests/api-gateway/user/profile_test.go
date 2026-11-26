package user_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	userhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

type profileClient struct {
	userclient.UserClient
	get    func(context.Context, int64) (*pb.GetProfileResponse, error)
	update func(context.Context, int64, string, string) (*pb.User, error)
}

func (s profileClient) GetProfile(ctx context.Context, id int64) (*pb.GetProfileResponse, error) {
	return s.get(ctx, id)
}
func (s profileClient) UpdateProfile(ctx context.Context, id int64, email, name string) (*pb.User, error) {
	return s.update(ctx, id, email, name)
}

func TestProfileGet(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		err           error
		status        int
	}{
		{
			name:          "profile",
			authenticated: true,
			status:        http.StatusOK,
		},
		{
			name:   "anonymous",
			status: http.StatusUnauthorized,
		},
		{
			name:          "deleted account",
			authenticated: true,
			err:           userclient.ErrUserNotFound,
			status:        http.StatusUnauthorized,
		},
		{
			name:          "private database failure",
			authenticated: true,
			err:           errors.New("private database details"),
			status:        http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := profileClient{
				get: func(_ context.Context, id int64) (*pb.GetProfileResponse, error) {
					calls++
					require.EqualValues(t, 7, id)
					return &pb.GetProfileResponse{
						User: &pb.User{
							Id:       7,
							Username: "Anna",
						},
					}, tt.err
				},
			}
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			handler := userhttp.NewProfileHandler(client, appLogger)

			request := httptest.NewRequest(http.MethodGet, "/api/me/profile?user_id=99", nil)
			if tt.authenticated {
				request = request.WithContext(middleware.ContextWithUserID(request.Context(), 7))
			}

			response := httptest.NewRecorder()
			handler.Get(response, request)
			require.Equal(t, tt.status, response.Code)

			if tt.authenticated {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}

			require.NotContains(t, response.Body.String(), "private")

			if tt.status == http.StatusOK {
				require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
				require.Contains(t, response.Body.String(), "Anna")
			}
		})
	}
}

func TestProfileUpdate(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		body          string
		err           error
		status        int
		calls         int
	}{
		{
			name:          "updated",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"Anna"}`,
			status:        http.StatusOK,
			calls:         1,
		},
		{
			name:   "anonymous",
			body:   `{"email":"anna@example.com","username":"Anna"}`,
			status: http.StatusUnauthorized,
		},
		{
			name:          "invalid JSON",
			authenticated: true,
			body:          `{"email":`,
			status:        http.StatusBadRequest,
		},
		{
			name:          "invalid email",
			authenticated: true,
			body:          `{"email":"invalid","username":"Anna"}`,
			err:           userclient.ErrInvalidEmail,
			status:        http.StatusBadRequest,
			calls:         1,
		},
		{
			name:          "invalid name",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"!"}`,
			err:           userclient.ErrInvalidUsername,
			status:        http.StatusBadRequest,
			calls:         1,
		},
		{
			name:          "email conflict",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"Anna"}`,
			err:           userclient.ErrEmailAlreadyExists,
			status:        http.StatusConflict,
			calls:         1,
		},
		{
			name:          "username conflict",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"Anna"}`,
			err:           userclient.ErrUsernameTaken,
			status:        http.StatusConflict,
			calls:         1,
		},
		{
			name:          "deleted account",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"Anna"}`,
			err:           userclient.ErrUserNotFound,
			status:        http.StatusUnauthorized,
			calls:         1,
		},
		{
			name:          "database failure",
			authenticated: true,
			body:          `{"email":"anna@example.com","username":"Anna"}`,
			err:           errors.New("private database details"),
			status:        http.StatusInternalServerError,
			calls:         1,
		},
	}
	for _, tt := range tests {
		for _, path := range []string{"/api/me/profile", "/api/me"} {
			t.Run(tt.name+path, func(t *testing.T) {
				calls := 0
				client := profileClient{
					update: func(_ context.Context, id int64, email, name string) (*pb.User, error) {
						calls++
						require.EqualValues(t, 7, id)
						require.Contains(t, tt.body, email)
						require.Contains(t, tt.body, name)
						return &pb.User{
							Id:       7,
							Email:    email,
							Username: name,
						}, tt.err
					},
				}
				appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
				require.NoError(t, err)

				handler := userhttp.NewProfileHandler(client, appLogger)

				request := httptest.NewRequest(http.MethodPut, path, strings.NewReader(tt.body))
				if tt.authenticated {
					request = request.WithContext(middleware.ContextWithUserID(request.Context(), 7))
				}

				response := httptest.NewRecorder()
				handler.Update(response, request)
				require.Equal(t, tt.status, response.Code)
				require.Equal(t, tt.calls, calls)
				require.NotContains(t, response.Body.String(), "private")
			})
		}
	}
}
