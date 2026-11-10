package auth_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	uc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func profileTestHandler(t *testing.T, repo uc.ProfileReader, update domain.UserRepository, posters uc.ProfilePosterStorage, auth router.Authenticator) http.Handler {
	t.Helper()
	appLogger := testutil.DiscardLogger()
	get, err := uc.NewGetProfileUseCase(repo, posters, appLogger)
	require.NoError(t, err)
	updateProfile, err := uc.NewUpdateProfileUseCase(update)
	require.NoError(t, err)
	handler := delivery.NewProfileHandler(get, updateProfile, appLogger)
	routes := mux.NewRouter()
	routes.HandleFunc("/api/me/profile", router.RequireAuth(auth)(handler.Get)).Methods(http.MethodGet)
	routes.HandleFunc("/api/me/profile", router.RequireAuth(auth)(handler.Update)).Methods(http.MethodPut)
	routes.HandleFunc("/api/me", router.RequireAuth(auth)(handler.Update)).Methods(http.MethodPut)
	return router.WithRequestID(router.WithLogging(routes, appLogger))
}

func TestProfileGetHTTP(t *testing.T) {
	created := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	key, url := "private-poster-key", "https://media.example/poster.png"
	score := 9
	source := &domain.Profile{
		User:      domain.ProfileUser{ID: 7, Email: "anna@example.com", Username: "anna", CreatedAt: created},
		Favorites: []domain.ProfileFilm{{ID: 5, Title: "Film", Year: 2020, Type: domain.FilmTypeMovie, AgeLimit: 16, Genres: "Drama", PosterStorageKey: &key, Average: 8.5}},
		Ratings:   []domain.ProfileFilm{{ID: 5, Title: "Film", Year: 2020, Type: domain.FilmTypeMovie, Genres: "Drama", Score: &score}},
		Reviews:   3,
	}
	for _, tc := range []struct {
		name             string
		session          bool
		authErr, repoErr error
		profile          *domain.Profile
		status           int
	}{
		{name: "profile", session: true, profile: source, status: http.StatusOK},
		{name: "empty lists", session: true, profile: &domain.Profile{User: source.User}, status: http.StatusOK},
		{name: "anonymous", status: http.StatusUnauthorized},
		{name: "invalid session", session: true, authErr: domain.ErrUnauthorized, status: http.StatusUnauthorized},
		{name: "deleted account", session: true, repoErr: domain.ErrUserNotFound, status: http.StatusUnauthorized},
		{name: "private database failure", session: true, repoErr: errors.New("private database failure"), status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockProfileReader(ctrl)
			update := mocks.NewMockUserRepository(ctrl)
			posters := mocks.NewMockProfilePosterStorage(ctrl)
			auth := mocks.NewMockAuthenticator(ctrl)
			if tc.session {
				authentication := auth.EXPECT().Authenticate(gomock.Any(), "sid").Return(domain.UserID(7), tc.authErr)
				if tc.authErr == nil {
					lookup := repo.EXPECT().GetProfile(gomock.Any(), domain.UserID(7)).Return(tc.profile, tc.repoErr).After(authentication)
					if tc.profile == source {
						posters.EXPECT().GetURL(gomock.Any(), key).Return(url, nil).After(lookup)
					}
				}
			}
			handler := profileTestHandler(t, repo, update, posters, auth)
			req := httptest.NewRequest(http.MethodGet, "/api/me/profile?user_id=99", nil)
			if tc.session {
				req.AddCookie(&http.Cookie{Name: delivery.SessionCookieName, Value: "sid"})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
			require.NotContains(t, rec.Body.String(), key)
			require.NotContains(t, rec.Body.String(), "password_hash")
			require.NotContains(t, rec.Body.String(), "private database failure")
			if tc.status != http.StatusOK {
				return
			}
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			var body struct {
				User struct {
					ID              int
					Email, Username string
					CreatedAt       time.Time `json:"created_at"`
				}
				Favorites []map[string]any
				Ratings   []map[string]any
				Reviews   int
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, 7, body.User.ID)
			require.Equal(t, created, body.User.CreatedAt)
			require.NotNil(t, body.Favorites)
			require.NotNil(t, body.Ratings)
			if tc.profile == source {
				require.Equal(t, 3, body.Reviews)
				require.Len(t, body.Favorites, 1)
				require.Equal(t, url, body.Favorites[0]["poster"])
				require.Equal(t, float64(2020), body.Favorites[0]["year"])
				require.Equal(t, "Drama", body.Favorites[0]["genres"])
				require.Equal(t, 8.5, body.Favorites[0]["average"])
				require.Equal(t, float64(9), body.Ratings[0]["score"])
			} else {
				require.Empty(t, body.Favorites)
				require.Empty(t, body.Ratings)
				require.Zero(t, body.Reviews)
			}
		})
	}
}

func TestProfileUpdateHTTP(t *testing.T) {
	for _, path := range []string{"/api/me/profile", "/api/me"} {
		for _, tc := range []struct {
			name, body string
			session    bool
			repoErr    error
			status     int
		}{
			{"success", `{"username":" anna ","email":" anna@example.com "}`, true, nil, http.StatusOK},
			{"username conflict", `{"username":"anna","email":"anna@example.com"}`, true, domain.ErrUsernameTaken, http.StatusConflict},
			{"email conflict", `{"username":"anna","email":"anna@example.com"}`, true, domain.ErrUserExists, http.StatusConflict},
			{"deleted user", `{"username":"anna","email":"anna@example.com"}`, true, domain.ErrUserNotFound, http.StatusUnauthorized},
			{"private database error", `{"username":"anna","email":"anna@example.com"}`, true, errors.New("private database error"), http.StatusInternalServerError},
			{"invalid email", `{"username":"anna","email":"invalid"}`, true, nil, http.StatusBadRequest},
			{"invalid name", `{"username":"","email":"anna@example.com"}`, true, nil, http.StatusBadRequest},
			{"cannot select other user", `{"username":"anna","email":"anna@example.com","user_id":99}`, true, nil, http.StatusBadRequest},
			{"invalid JSON", `{`, true, nil, http.StatusBadRequest},
			{"anonymous", `{}`, false, nil, http.StatusUnauthorized},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				read := mocks.NewMockProfileReader(ctrl)
				update := mocks.NewMockUserRepository(ctrl)
				posters := mocks.NewMockProfilePosterStorage(ctrl)
				auth := mocks.NewMockAuthenticator(ctrl)
				if tc.session {
					authentication := auth.EXPECT().Authenticate(gomock.Any(), "sid").Return(domain.UserID(7), nil)
					if tc.status == http.StatusOK || tc.repoErr != nil {
						update.EXPECT().UpdateProfile(gomock.Any(), domain.UserID(7), "anna@example.com", "anna").Return(&domain.User{ID: 7, Email: "anna@example.com", Username: "anna", PasswordHash: "private-password-hash"}, tc.repoErr).After(authentication)
					}
				}
				handler := profileTestHandler(t, read, update, posters, auth)
				req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				if tc.session {
					req.AddCookie(&http.Cookie{Name: delivery.SessionCookieName, Value: "sid"})
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				require.Equal(t, tc.status, rec.Code, rec.Body.String())
				require.NotContains(t, rec.Body.String(), "private")
				if tc.status == http.StatusOK {
					require.JSONEq(t, `{"user":{"id":7,"email":"anna@example.com","username":"anna"}}`, rec.Body.String())
					require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
				}
				if tc.status == http.StatusConflict && path == "/api/me/profile" {
					expected := "email taken"
					if errors.Is(tc.repoErr, domain.ErrUsernameTaken) {
						expected = "username taken"
					}
					require.JSONEq(t, `{"error":"`+expected+`"}`, rec.Body.String())
				}
			})
		}
	}
}
