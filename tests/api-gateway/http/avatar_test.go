package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	userhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type avatarUserStub struct {
	userclient.UserClient
	upload func(context.Context, int64, []byte) (string, error)
}

func (s avatarUserStub) UpdateAvatar(ctx context.Context, id int64, data []byte) (string, error) {
	return s.upload(ctx, id, data)
}

func TestGatewayAvatarUpload(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		path          string
		field         string
		files         int
		data          []byte
		authenticated bool
		uploadErr     error
		status        int
		wantCalls     int
	}{
		{
			name:          "upload",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			data:          []byte("image bytes"),
			authenticated: true,
			status:        http.StatusCreated,
			wantCalls:     1,
		},
		{
			name:          "replace own avatar",
			method:        http.MethodPut,
			path:          "/api/me/avatar",
			field:         "avatar",
			files:         1,
			data:          []byte("image bytes"),
			authenticated: true,
			status:        http.StatusOK,
			wantCalls:     1,
		},
		{
			name:   "anonymous upload",
			method: http.MethodPost,
			path:   "/api/avatars",
			field:  "avatar",
			files:  1,
			data:   []byte("image bytes"),
			status: http.StatusUnauthorized,
		},
		{
			name:          "missing file",
			method:        http.MethodPost,
			path:          "/api/avatars",
			authenticated: true,
			status:        http.StatusBadRequest,
		},
		{
			name:          "wrong field",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "photo",
			files:         1,
			data:          []byte("image bytes"),
			authenticated: true,
			status:        http.StatusBadRequest,
		},
		{
			name:          "duplicate file",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         2,
			data:          []byte("image bytes"),
			authenticated: true,
			status:        http.StatusBadRequest,
		},
		{
			name:          "empty file",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			authenticated: true,
			status:        http.StatusBadRequest,
		},
		{
			name:          "oversized file",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			data:          make([]byte, imageutil.MaxAvatarSize+1),
			authenticated: true,
			status:        http.StatusRequestEntityTooLarge,
		},
		{
			name:          "invalid image from user service",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			data:          []byte("invalid"),
			authenticated: true,
			uploadErr:     userclient.ErrInvalidArgument,
			status:        http.StatusBadRequest,
			wantCalls:     1,
		},
		{
			name:          "deleted account",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			data:          []byte("image bytes"),
			authenticated: true,
			uploadErr:     userclient.ErrUserNotFound,
			status:        http.StatusUnauthorized,
			wantCalls:     1,
		},
		{
			name:          "storage failure",
			method:        http.MethodPost,
			path:          "/api/avatars",
			field:         "avatar",
			files:         1,
			data:          []byte("image bytes"),
			authenticated: true,
			uploadErr:     errors.New("private storage failure"),
			status:        http.StatusInternalServerError,
			wantCalls:     1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			users := avatarUserStub{
				upload: func(_ context.Context, id int64, data []byte) (string, error) {
					calls++
					require.EqualValues(t, 7, id)
					require.Equal(t, tt.data, data)
					return "avatar.png", tt.uploadErr
				},
			}
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			auth := authStub{}
			handler, err := router.NewRouter(router.Handlers{
				Auth:  authhttp.NewAuthHandler(auth, users, appLogger),
				Files: userhttp.NewAvatarHandler(users, appLogger),
			}, middleware.NewAuthMiddleware(auth, users, appLogger), []string{"http://localhost:8000"}, appLogger)
			require.NoError(t, err)
			var body bytes.Buffer

			writer := multipart.NewWriter(&body)
			for range tt.files {
				part, err := writer.CreateFormFile(tt.field, "test.png")
				require.NoError(t, err)
				_, err = part.Write(tt.data)
				require.NoError(t, err)
			}

			require.NoError(t, writer.Close())

			request := httptest.NewRequest(tt.method, tt.path, &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			request.Header.Set("Origin", "http://localhost:8000")

			if tt.authenticated {
				request.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.wantCalls, calls)
			require.NotContains(t, response.Body.String(), "private")

			if tt.status == http.StatusCreated || tt.status == http.StatusOK {
				require.JSONEq(t, `{"filename":"avatar.png"}`, response.Body.String())
			}
		})
	}
}
