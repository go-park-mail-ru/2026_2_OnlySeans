package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOrigin = "http://localhost:3000"

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	authHandler, err := auth.NewHandler(auth.NewInMemoryUserRepo(), auth.NewInMemorySessionStore(time.Hour))
	require.NoError(t, err)
	filmsService, err := films.NewSeededService()
	require.NoError(t, err)

	return router.NewRouter(authHandler, films.NewHandler(filmsService), testOrigin)
}

func TestWithCORS(t *testing.T) {
	handler := newTestRouter(t)

	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int
		wantNextCall    bool
		wantAllowOrigin string
	}{
		{"preflight отвечает middleware", http.MethodOptions, testOrigin, http.StatusNoContent, false, testOrigin},
		{"обычный запрос доходит до хендлера", http.MethodGet, testOrigin, http.StatusOK, true, testOrigin},
		{"чужой origin не получает разрешения", http.MethodGet, "http://evil.example", http.StatusOK, true, ""},
		{"запрос без origin", http.MethodGet, "", http.StatusOK, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/films", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидали %d", rec.Code, tt.wantStatus)
			}
			if nextCalled := rec.Body.Len() > 0; nextCalled != tt.wantNextCall {
				t.Errorf("next вызван = %v, ожидали %v", nextCalled, tt.wantNextCall)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllowOrigin {
				t.Errorf("Allow-Origin = %q, ожидали %q", got, tt.wantAllowOrigin)
			}
			if tt.wantAllowOrigin != "" && rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
				t.Error("нет Access-Control-Allow-Credentials: true")
			}
		})
	}
}

type fakeAuthenticator struct {
	userID       auth.UserID
	err          error
	gotSessionID string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, sessionID string) (auth.UserID, error) {
	f.gotSessionID = sessionID
	return f.userID, f.err
}

func TestRequireAuth(t *testing.T) {
	sidCookie := &http.Cookie{Name: auth.SessionCookieName, Value: "sid"}

	tests := []struct {
		name       string
		cookie     *http.Cookie
		authErr    error
		wantStatus int
		wantError  string
		wantNext   bool
	}{
		{"no cookie", nil, nil, http.StatusUnauthorized, auth.ErrUnauthorized.Error(), false},
		{"invalid session", sidCookie, auth.ErrUnauthorized, http.StatusUnauthorized, auth.ErrUnauthorized.Error(), false},
		{"internal error", sidCookie, auth.ErrInternal, http.StatusInternalServerError, auth.ErrInternal.Error(), false},
		{"unexpected error is hidden", sidCookie, errors.New("db is down"), http.StatusInternalServerError, auth.ErrInternal.Error(), false},
		{"valid session", sidCookie, nil, http.StatusOK, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authn := &fakeAuthenticator{userID: 42, err: tt.authErr}

			nextCalled := false
			var gotID auth.UserID
			var gotOK bool
			handler := router.RequireAuth(authn, func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotID, gotOK = auth.UserIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled, "next called")

			if tt.cookie != nil {
				assert.Equal(t, tt.cookie.Value, authn.gotSessionID, "session ID from cookie must reach Authenticate")
			}

			if tt.wantNext {
				assert.True(t, gotOK, "user ID must be in context")
				assert.Equal(t, auth.UserID(42), gotID)
				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantError, body["error"])
		})
	}
}

func TestRouter_LogoutRequiresAuth(t *testing.T) {
	handler := newTestRouter(t)

	register := httptest.NewRecorder()
	handler.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/api/register",
		strings.NewReader(`{"email":"out@example.com","username":"outuser","password":"Password1"}`)))
	require.Equal(t, http.StatusCreated, register.Code)

	cookies := register.Result().Cookies()
	require.Len(t, cookies, 1)
	valid := cookies[0]

	tests := []struct {
		name       string
		cookie     *http.Cookie
		wantStatus int
	}{
		{"без куки", nil, http.StatusUnauthorized},
		{"неизвестная сессия", &http.Cookie{Name: auth.SessionCookieName, Value: "unknown"}, http.StatusUnauthorized},
		{"валидная сессия", valid, http.StatusNoContent},
		{"повторный выход той же сессией", valid, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
