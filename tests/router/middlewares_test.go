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

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOrigin = "http://localhost:3000"

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	authHandler := newTestAuthHandler(t)
	filmsRepo, err := filmsservice.NewSeededService()
	require.NoError(t, err)
	listFilms, err := filmsusecase.NewListFilmsUseCase(filmsRepo)
	require.NoError(t, err)
	getFilm, err := filmsusecase.NewGetFilmUseCase(filmsRepo)
	require.NoError(t, err)
	listCollections, err := filmsusecase.NewListCollectionsUseCase(filmsRepo)
	require.NoError(t, err)
	getCollection, err := filmsusecase.NewGetCollectionUseCase(filmsRepo)
	require.NoError(t, err)
	filmsHandler, err := films.NewHandler(films.HandlerUseCases{
		ListFilms:       listFilms,
		GetFilm:         getFilm,
		ListCollections: listCollections,
		GetCollection:   getCollection,
	})
	require.NoError(t, err)

	handler, err := router.NewRouter(authHandler, filmsHandler, newTestReviewsHandler(t, filmsRepo), testOrigin)
	require.NoError(t, err)
	return handler
}

func newTestAuthHandler(t *testing.T) *auth.Handler {
	t.Helper()

	users := authservice.NewInMemoryUserRepo()
	sessions := authservice.NewInMemorySessionStore(time.Hour)
	passwordService := authservice.BcryptPasswordService{}

	register, err := authusecase.NewRegisterUserUseCase(users, sessions, passwordService)
	require.NoError(t, err)
	login, err := authusecase.NewLoginUserUseCase(users, sessions, passwordService)
	require.NoError(t, err)
	logout, err := authusecase.NewLogoutUserUseCase(sessions)
	require.NoError(t, err)
	authenticate, err := authusecase.NewAuthenticateUserUseCase(sessions)
	require.NoError(t, err)
	getUser, err := authusecase.NewGetUserUseCase(users)
	require.NoError(t, err)

	handler, err := auth.NewHandler(auth.HandlerUseCases{
		Register:     register,
		Login:        login,
		Logout:       logout,
		Authenticate: authenticate,
		GetUser:      getUser,
	})
	require.NoError(t, err)
	return handler
}

func newTestReviewsHandler(t *testing.T, films domain.FilmRepository) *reviewsdelivery.Handler {
	t.Helper()

	repo := reviewsservice.NewInMemoryReviewRepo()
	list, err := reviewsusecase.NewListReviewsUseCase(repo, films)
	require.NoError(t, err)
	create, err := reviewsusecase.NewCreateReviewUseCase(repo, films)
	require.NoError(t, err)
	update, err := reviewsusecase.NewUpdateReviewUseCase(repo)
	require.NoError(t, err)
	del, err := reviewsusecase.NewDeleteReviewUseCase(repo)
	require.NoError(t, err)

	handler, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   list,
		Create: create,
		Update: update,
		Delete: del,
	})
	require.NoError(t, err)
	return handler
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
	userID       domain.UserID
	err          error
	gotSessionID string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, sessionID string) (domain.UserID, error) {
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
		{"no cookie", nil, nil, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"invalid session", sidCookie, domain.ErrUnauthorized, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), false},
		{"internal error", sidCookie, domain.ErrInternal, http.StatusInternalServerError, domain.ErrInternal.Error(), false},
		{"unexpected error is hidden", sidCookie, errors.New("db is down"), http.StatusInternalServerError, domain.ErrInternal.Error(), false},
		{"valid session", sidCookie, nil, http.StatusOK, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authn := &fakeAuthenticator{userID: 42, err: tt.authErr}

			nextCalled := false
			var gotID domain.UserID
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
				assert.Equal(t, domain.UserID(42), gotID)
				return
			}

			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
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

// заголовки безопасности есть во всех видах ответов: API, статика, preflight, ошибка
func TestWithSecurityHeaders(t *testing.T) {
	handler := newTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"ручка API", http.MethodGet, "/api/films"},
		{"статика", http.MethodGet, "/static/posters/inceptio.svg"},
		{"preflight", http.MethodOptions, "/api/login"},
		{"ошибка", http.MethodGet, "/api/films/0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
			assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get("Content-Security-Policy"))
			assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		})
	}
}

// запросы, меняющие данные,с чужих сайтов 403
func TestCSRFProtection(t *testing.T) {
	handler := newTestRouter(t)

	tests := []struct {
		name        string
		method      string
		path        string
		origin      string
		fetchSite   string
		wantBlocked bool
	}{
		{"чужой сайт", http.MethodPost, "/api/login", "http://evil.example", "cross-site", true},
		{"соседний порт", http.MethodPost, "/api/login", "http://localhost:9999", "same-site", true},
		{"старый браузер", http.MethodPost, "/api/login", "http://evil.example", "", true},
		{"удаление отзыва с чужого", http.MethodDelete, "/api/reviews/1", "http://evil.example", "cross-site", true},
		{"наш фронт", http.MethodPost, "/api/login", testOrigin, "same-site", false},
		{"не браузер", http.MethodPost, "/api/login", "", "", false},
		{"GET с чужого", http.MethodGet, "/api/films", "http://evil.example", "cross-site", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tt.fetchSite)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if tt.wantBlocked {
				//заблокирован
				assert.Equal(t, http.StatusForbidden, rec.Code)
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, router.ErrCrossOrigin.Error(), body["error"])
			} else {
				//пропущен
				assert.NotEqual(t, http.StatusForbidden, rec.Code)
			}
		})
	}
}
