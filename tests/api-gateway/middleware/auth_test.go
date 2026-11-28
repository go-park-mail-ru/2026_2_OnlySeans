package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	authpb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

type sessionClient struct {
	authclient.AuthClient
	check   func(context.Context, string) (authclient.SessionInfo, error)
	refresh func(context.Context, string) (*authpb.Session, error)
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
					return authclient.SessionInfo{RefreshSupported: true, LastRefreshedAt: time.Now(),
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

func (s sessionClient) RefreshSession(ctx context.Context, id string) (*authpb.Session, error) {
	return s.refresh(ctx, id)
}

func TestAuthenticateRefreshesOnlyOnNewDay(t *testing.T) {
	for _, last := range []time.Time{{}, time.Now().AddDate(0, 0, -1), time.Now()} {
		calls := 0
		expires := time.Now().Add(72 * time.Hour).Truncate(time.Second)
		client := sessionClient{
			check: func(context.Context, string) (authclient.SessionInfo, error) {
				return authclient.SessionInfo{RefreshSupported: true, UserID: 7, Role: "reader", LastRefreshedAt: last}, nil
			},
			refresh: func(context.Context, string) (*authpb.Session, error) {
				calls++
				return &authpb.Session{Id: "session", UserId: 7, Role: "reader", ExpiresAt: timestamppb.New(expires)}, nil
			},
		}
		guard := middleware.NewAuthMiddleware(client, nil, nil)
		guard.CookieSecure = true
		req := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

		w := httptest.NewRecorder()
		guard.Authenticate(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })(w, req)
		require.Equal(t, http.StatusNoContent, w.Code)

		if last.IsZero() || last.Before(time.Now().Add(-time.Hour)) {
			require.Equal(t, 1, calls)

			cookies := w.Result().Cookies()
			require.Len(t, cookies, 1)
			require.True(t, expires.Equal(cookies[0].Expires))
			require.True(t, cookies[0].HttpOnly)
			require.True(t, cookies[0].Secure)
			require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		} else {
			require.Zero(t, calls)
			require.Empty(t, w.Header().Get("Set-Cookie"))
		}
	}
}

func TestOptionalAuthenticationSharesSessionCheck(t *testing.T) {
	checks := 0
	client := sessionClient{
		check: func(context.Context, string) (authclient.SessionInfo, error) {
			checks++
			return authclient.SessionInfo{RefreshSupported: true, UserID: 7, Role: "reader", LastRefreshedAt: time.Now()}, nil
		},
	}
	guard := middleware.NewAuthMiddleware(client, nil, nil)
	next := http.HandlerFunc(guard.Authenticate(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

	w := httptest.NewRecorder()
	guard.OptionalAuthenticate(next).ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, 1, checks)
}

func TestOptionalAuthenticationAllowsGuestAndRefreshesPublicRequest(t *testing.T) {
	checks := 0
	refreshes := 0
	client := sessionClient{
		check: func(context.Context, string) (authclient.SessionInfo, error) {
			checks++
			return authclient.SessionInfo{RefreshSupported: true, UserID: 7, Role: "reader"}, nil
		},
		refresh: func(context.Context, string) (*authpb.Session, error) {
			refreshes++
			return &authpb.Session{Id: "session", UserId: 7, ExpiresAt: timestamppb.New(time.Now().Add(72 * time.Hour))}, nil
		},
	}
	guard := middleware.NewAuthMiddleware(client, nil, nil)
	handler := guard.OptionalAuthenticate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	guest := httptest.NewRecorder()
	handler.ServeHTTP(guest, httptest.NewRequest(http.MethodGet, "/api/films", nil))
	require.Equal(t, http.StatusNoContent, guest.Code)
	require.Zero(t, checks)

	req := httptest.NewRequest(http.MethodGet, "/api/films", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, 1, checks)
	require.Equal(t, 1, refreshes)
	require.Len(t, w.Result().Cookies(), 1)
}

func TestSessionCookieRecoveryAndLogout(t *testing.T) {
	for _, path := range []string{"/api/authorised", "/api/logout"} {
		t.Run(path, func(t *testing.T) {
			expiry := time.Now().Add(72 * time.Hour).UTC().Truncate(time.Second)
			client := sessionClient{check: func(context.Context, string) (authclient.SessionInfo, error) {
				return authclient.SessionInfo{UserID: 7, ExpiresAt: expiry, LastRefreshedAt: time.Now(), RefreshSupported: true}, nil
			}}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

			w := httptest.NewRecorder()
			middleware.NewAuthMiddleware(client, nil, nil).Authenticate(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })(w, req)
			require.Equal(t, http.StatusNoContent, w.Code)

			if path == "/api/logout" {
				require.Empty(t, w.Result().Cookies())
			} else {
				require.Len(t, w.Result().Cookies(), 1)
				require.True(t, expiry.Equal(w.Result().Cookies()[0].Expires))
				require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestOldAuthResponseDoesNotRefresh(t *testing.T) {
	client := sessionClient{check: func(context.Context, string) (authclient.SessionInfo, error) {
		return authclient.SessionInfo{UserID: 7, Role: "reader"}, nil
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/authorised", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session"})

	w := httptest.NewRecorder()
	middleware.NewAuthMiddleware(client, nil, nil).Authenticate(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Empty(t, w.Result().Cookies())
}
