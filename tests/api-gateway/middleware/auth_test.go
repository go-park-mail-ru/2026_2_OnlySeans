package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

type sessionClient struct {
	authclient.AuthClient
	check func(context.Context, string) (authclient.SessionInfo, error)
}

func (s sessionClient) CheckSession(ctx context.Context, id string) (authclient.SessionInfo, error) {
	return s.check(ctx, id)
}

type roleClient struct {
	userclient.UserClient
	role  string
	err   error
	calls int
}

func (s *roleClient) GetByID(_ context.Context, _ int64) (*pb.User, error) {
	s.calls++

	return &pb.User{
		Role: s.role,
	}, s.err
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name   string
		cookie *http.Cookie
		err    error
		status int
		calls  int
	}{
		{
			name:   "missing cookie",
			status: http.StatusUnauthorized,
		},
		{
			name: "empty cookie",
			cookie: &http.Cookie{
				Name: "session_id",
			},
			status: http.StatusUnauthorized,
		},
		{
			name: "active session",
			cookie: &http.Cookie{
				Name:  "session_id",
				Value: "session",
			},
			status: http.StatusNoContent,
			calls:  1,
		},
		{
			name: "expired or unknown session",
			cookie: &http.Cookie{
				Name:  "session_id",
				Value: "session",
			},
			err:    authclient.ErrSessionNotFound,
			status: http.StatusUnauthorized,
			calls:  1,
		},
		{
			name: "auth service unavailable",
			cookie: &http.Cookie{
				Name:  "session_id",
				Value: "session",
			},
			err:    authclient.ErrInternal,
			status: http.StatusInternalServerError,
			calls:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			appLogger, err := logger.NewLogger(&logs, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			calls := 0
			auth := sessionClient{
				check: func(_ context.Context, id string) (authclient.SessionInfo, error) {
					calls++
					require.Equal(t, "session", id)
					return authclient.SessionInfo{
						UserID: 7,
						Role:   "reader",
					}, tt.err
				},
			}
			guard := middleware.NewAuthMiddleware(auth, nil, appLogger)
			called := false
			handler := guard.Authenticate(func(w http.ResponseWriter, r *http.Request) {
				called = true
				id, ok := middleware.UserIDFromContext(r.Context())
				require.True(t, ok)
				require.EqualValues(t, 7, id)

				role, ok := middleware.RoleFromContext(r.Context())
				require.True(t, ok)
				require.Equal(t, "reader", role)
				w.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
			if tt.cookie != nil {
				request.AddCookie(tt.cookie)
			}

			response := httptest.NewRecorder()
			handler(response, request)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, tt.calls, calls)
			require.Equal(t, tt.status == http.StatusNoContent, called)
		})
	}
}

func TestPermissions(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		role          string
		err           error
		adminStatus   int
		editorStatus  int
	}{
		{
			name:         "anonymous",
			adminStatus:  http.StatusUnauthorized,
			editorStatus: http.StatusUnauthorized,
		},
		{
			name:          "reader",
			authenticated: true,
			role:          "reader",
			adminStatus:   http.StatusForbidden,
			editorStatus:  http.StatusForbidden,
		},
		{
			name:          "editor",
			authenticated: true,
			role:          "editor",
			adminStatus:   http.StatusForbidden,
			editorStatus:  http.StatusNoContent,
		},
		{
			name:          "admin",
			authenticated: true,
			role:          "admin",
			adminStatus:   http.StatusNoContent,
			editorStatus:  http.StatusNoContent,
		},
		{
			name:          "unknown role",
			authenticated: true,
			role:          "other",
			adminStatus:   http.StatusForbidden,
			editorStatus:  http.StatusForbidden,
		},
		{
			name:          "deleted account",
			authenticated: true,
			err:           userclient.ErrUserNotFound,
			adminStatus:   http.StatusUnauthorized,
			editorStatus:  http.StatusUnauthorized,
		},
		{
			name:          "user service unavailable",
			authenticated: true,
			err:           userclient.ErrInternal,
			adminStatus:   http.StatusInternalServerError,
			editorStatus:  http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		for _, kind := range []string{"admin", "editor"} {
			t.Run(tt.name+"/"+kind, func(t *testing.T) {
				var logs bytes.Buffer
				appLogger, err := logger.NewLogger(&logs, logger.LogLevelInfo, "json")
				require.NoError(t, err)
				users := &roleClient{
					role: tt.role,
					err:  tt.err,
				}
				guard := middleware.NewAuthMiddleware(nil, users, appLogger)
				next := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }
				handler := guard.RequireAdmin(next)
				want := tt.adminStatus

				if kind == "editor" {
					handler = guard.RequireCatalogEditor(next)
					want = tt.editorStatus
				}

				request := httptest.NewRequest(http.MethodPost, "/api/persons", nil)
				if tt.authenticated {
					request = request.WithContext(middleware.ContextWithUserID(request.Context(), 7))
				}

				response := httptest.NewRecorder()
				handler(response, request)
				require.Equal(t, want, response.Code)

				if tt.authenticated {
					require.Equal(t, 1, users.calls)
				} else {
					require.Zero(t, users.calls)
				}
			})
		}
	}
}

func TestRequestLogging(t *testing.T) {
	tests := []struct {
		name   string
		status int
		level  string
	}{
		{
			name:   "success",
			status: http.StatusNoContent,
			level:  "INFO",
		},
		{
			name:   "client error",
			status: http.StatusBadRequest,
			level:  "WARN",
		},
		{
			name:   "server error",
			status: http.StatusInternalServerError,
			level:  "ERROR",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			appLogger, err := logger.NewLogger(&logs, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			handler := middleware.WithRequestID(middleware.WithLogging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NotEmpty(t, logger.RequestID(r.Context()))
				w.WriteHeader(tt.status)
			}), appLogger))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/films?token=private", nil))
			var entry map[string]any
			require.NoError(t, json.Unmarshal(logs.Bytes(), &entry))
			require.Equal(t, tt.level, entry["level"])
			require.Equal(t, "/api/films", entry["path"])
			require.Equal(t, "GET", entry["method"])
			require.EqualValues(t, tt.status, entry["status"])
			require.Equal(t, response.Header().Get("X-Request-ID"), entry["request_id"])
			require.Contains(t, entry, "duration_ms")
			require.NotContains(t, logs.String(), "private")
		})
	}
}

func TestCORSAndCSRF(t *testing.T) {
	const origin = "http://localhost:8000"

	tests := []struct {
		name    string
		method  string
		origin  string
		status  int
		allowed bool
	}{
		{
			name:    "trusted preflight",
			method:  http.MethodOptions,
			origin:  origin,
			status:  http.StatusNoContent,
			allowed: true,
		},
		{
			name:   "untrusted preflight",
			method: http.MethodOptions,
			origin: "http://evil.example",
			status: http.StatusNoContent,
		},
		{
			name:    "trusted write",
			method:  http.MethodPost,
			origin:  origin,
			status:  http.StatusOK,
			allowed: true,
		},
		{
			name:   "untrusted write rejected",
			method: http.MethodPost,
			origin: "http://evil.example",
			status: http.StatusForbidden,
		},
		{
			name:   "safe request from untrusted origin",
			method: http.MethodGet,
			origin: "http://evil.example",
			status: http.StatusOK,
		},
		{
			name:   "no origin",
			method: http.MethodGet,
			status: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := middleware.WithCORSAndCSRF(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }), []string{origin})
			require.NoError(t, err)

			request := httptest.NewRequest(tt.method, "/api/films", nil)
			request.Header.Set("Origin", tt.origin)

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", response.Header().Get("Access-Control-Allow-Methods"))

			if tt.allowed {
				require.Equal(t, origin, response.Header().Get("Access-Control-Allow-Origin"))
			} else {
				require.Empty(t, response.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}
