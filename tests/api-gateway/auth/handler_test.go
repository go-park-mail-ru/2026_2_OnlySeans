package auth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pbauth "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/auth"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

type authClientStub struct {
	authclient.AuthClient
	issue  func(context.Context, int64, string) (*pbauth.Session, error)
	login  func(context.Context, string, string) (*pbauth.Session, error)
	logout func(context.Context, string) error
}

func (s authClientStub) IssueSession(ctx context.Context, id int64, role string) (*pbauth.Session, error) {
	return s.issue(ctx, id, role)
}
func (s authClientStub) Login(ctx context.Context, email, password string) (*pbauth.Session, error) {
	return s.login(ctx, email, password)
}
func (s authClientStub) Logout(ctx context.Context, id string) error { return s.logout(ctx, id) }

type userClientStub struct {
	userclient.UserClient
	create func(context.Context, string, string, string) (int64, error)
	get    func(context.Context, int64) (*pbuser.User, error)
}

func (s userClientStub) CreateUser(ctx context.Context, email, name, password string) (int64, error) {
	return s.create(ctx, email, name, password)
}
func (s userClientStub) GetByID(ctx context.Context, id int64) (*pbuser.User, error) {
	return s.get(ctx, id)
}

func testAuthLogger(t *testing.T) logger.Logger {
	t.Helper()

	l, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
	require.NoError(t, err)

	return l
}

func TestAuthRejectsInvalidRequests(t *testing.T) {
	h := authhttp.NewAuthHandler(authClientStub{}, userClientStub{}, testAuthLogger(t))
	for name, handler := range map[string]http.HandlerFunc{
		"register": h.Register,
		"login":    h.Login,
		"logout":   h.Logout,
	} {
		for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			t.Run(name+"/"+method, func(t *testing.T) {
				response := httptest.NewRecorder()
				handler(response, httptest.NewRequest(method, "/", nil))
				require.Equal(t, http.StatusMethodNotAllowed, response.Code)
				require.Empty(t, response.Result().Cookies())
			})
		}
	}

	for name, handler := range map[string]http.HandlerFunc{
		"register": h.Register,
		"login":    h.Login,
	} {
		for _, body := range []string{"", `null`, `[]`, `{"email":1}`, `{"email":`, `{} {}`} {
			t.Run(name+"/"+body, func(t *testing.T) {
				response := httptest.NewRecorder()
				handler(response, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
				require.Equal(t, http.StatusBadRequest, response.Code)
				require.Empty(t, response.Result().Cookies())
			})
		}
	}
}

func TestRegisterStages(t *testing.T) {
	failure := errors.New("private database failure")

	tests := []struct {
		name       string
		createErr  error
		getErr     error
		issueErr   error
		wantStatus int
		calls      []string
	}{
		{
			name:       "success",
			wantStatus: http.StatusCreated,
			calls:      []string{"create", "get", "session"},
		},
		{
			name:       "email taken",
			createErr:  userclient.ErrUserExists,
			wantStatus: http.StatusConflict,
			calls:      []string{"create"},
		},
		{
			name:       "username taken",
			createErr:  userclient.ErrUsernameTaken,
			wantStatus: http.StatusConflict,
			calls:      []string{"create"},
		},
		{
			name:       "invalid email",
			createErr:  userclient.ErrInvalidEmail,
			wantStatus: http.StatusBadRequest,
			calls:      []string{"create"},
		},
		{
			name:       "invalid username",
			createErr:  userclient.ErrInvalidUsername,
			wantStatus: http.StatusBadRequest,
			calls:      []string{"create"},
		},
		{
			name:       "invalid password",
			createErr:  userclient.ErrInvalidPassword,
			wantStatus: http.StatusBadRequest,
			calls:      []string{"create"},
		},
		{
			name:       "user creation failure",
			createErr:  failure,
			wantStatus: http.StatusInternalServerError,
			calls:      []string{"create"},
		},
		{
			name:       "load user failure",
			getErr:     failure,
			wantStatus: http.StatusInternalServerError,
			calls:      []string{"create", "get"},
		},
		{
			name:       "session failure",
			issueErr:   failure,
			wantStatus: http.StatusInternalServerError,
			calls:      []string{"create", "get", "session"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			expires := time.Now().Add(time.Hour).Truncate(time.Second)
			users := userClientStub{
				create: func(_ context.Context, email, name, password string) (int64, error) {
					calls = append(calls, "create")
					require.Equal(t, "Mixed@example.test", email)
					require.Equal(t, "Anna", name)
					require.Equal(t, "Password1", password)
					return 7, tt.createErr
				},
				get: func(_ context.Context, id int64) (*pbuser.User, error) {
					calls = append(calls, "get")
					require.EqualValues(t, 7, id)
					return &pbuser.User{
						Id:       7,
						Email:    "Mixed@example.test",
						Username: "Anna",
					}, tt.getErr
				},
			}
			auth := authClientStub{
				issue: func(_ context.Context, id int64, role string) (*pbauth.Session, error) {
					calls = append(calls, "session")
					require.EqualValues(t, 7, id)
					require.Equal(t, "reader", role)
					return &pbauth.Session{
						Id:        "session",
						UserId:    id,
						ExpiresAt: timestamppb.New(expires),
					}, tt.issueErr
				},
			}
			h := authhttp.NewAuthHandler(auth, users, testAuthLogger(t))
			h.CookieSecure = true
			response := httptest.NewRecorder()
			h.Register(response, httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{"email":" Mixed@example.test ","username":" Anna ","password":"Password1"}`)))
			require.Equal(t, tt.wantStatus, response.Code)
			require.Equal(t, tt.calls, calls)
			require.NotContains(t, response.Body.String(), "private")

			if tt.wantStatus != http.StatusCreated {
				require.Empty(t, response.Result().Cookies())
				return
			}

			require.NotContains(t, strings.ToLower(response.Body.String()), "password")
			cookies := response.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, authhttp.SessionCookieName, cookies[0].Name)
			require.Equal(t, "session", cookies[0].Value)
			require.True(t, cookies[0].HttpOnly)
			require.True(t, cookies[0].Secure)
			require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
			require.Equal(t, "/", cookies[0].Path)
			require.True(t, expires.Equal(cookies[0].Expires))
		})
	}
}

func TestLoginStages(t *testing.T) {
	tests := []struct {
		name     string
		loginErr error
		getErr   error
		status   int
	}{
		{
			name:   "success",
			status: http.StatusOK,
		},
		{
			name:     "invalid credentials",
			loginErr: authclient.ErrInvalidCredentials,
			status:   http.StatusUnauthorized,
		},
		{
			name:     "auth service failure",
			loginErr: authclient.ErrInternal,
			status:   http.StatusInternalServerError,
		},
		{
			name:   "user lookup failure",
			getErr: userclient.ErrInternal,
			status: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logins, lookups := 0, 0
			auth := authClientStub{
				login: func(_ context.Context, email, password string) (*pbauth.Session, error) {
					logins++
					require.Equal(t, "Mixed@example.test", email)
					require.Equal(t, "Password1", password)
					return &pbauth.Session{
						Id:        "session",
						UserId:    7,
						ExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
					}, tt.loginErr
				},
			}
			users := userClientStub{
				get: func(_ context.Context, id int64) (*pbuser.User, error) {
					lookups++
					require.EqualValues(t, 7, id)
					return &pbuser.User{
						Id:    7,
						Email: "Mixed@example.test",
					}, tt.getErr
				},
			}
			h := authhttp.NewAuthHandler(auth, users, testAuthLogger(t))
			response := httptest.NewRecorder()
			h.Login(response, httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"email":" Mixed@example.test ","password":"Password1"}`)))
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, 1, logins)

			if tt.loginErr == nil {
				require.Equal(t, 1, lookups)
			} else {
				require.Zero(t, lookups)
			}

			if tt.status == http.StatusOK {
				require.Len(t, response.Result().Cookies(), 1)
				require.Equal(t, "session", response.Result().Cookies()[0].Value)
				require.NotContains(t, strings.ToLower(response.Body.String()), "password")
			} else {
				require.Empty(t, response.Result().Cookies())
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name   string
		cookie bool
		err    error
		status int
	}{
		{
			name:   "delete session",
			cookie: true,
			status: http.StatusNoContent,
		},
		{
			name:   "no session cookie",
			status: http.StatusNoContent,
		},
		{
			name:   "auth service unavailable",
			cookie: true,
			err:    authclient.ErrInternal,
			status: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			auth := authClientStub{
				logout: func(_ context.Context, id string) error {
					calls++
					require.Equal(t, "session", id)
					return tt.err
				},
			}
			h := authhttp.NewAuthHandler(auth, userClientStub{}, testAuthLogger(t))

			request := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
			if tt.cookie {
				request.AddCookie(&http.Cookie{
					Name:  authhttp.SessionCookieName,
					Value: "session",
				})
			}

			response := httptest.NewRecorder()
			h.Logout(response, request)
			require.Equal(t, tt.status, response.Code)

			if tt.cookie {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}

			if tt.err == nil {
				cookies := response.Result().Cookies()
				require.Len(t, cookies, 1)
				require.Equal(t, -1, cookies[0].MaxAge)
				require.Empty(t, cookies[0].Value)
			} else {
				require.Empty(t, response.Result().Cookies())
			}
		})
	}
}
