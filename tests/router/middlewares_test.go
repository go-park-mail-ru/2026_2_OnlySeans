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
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	personsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOrigin = "http://localhost:3000"

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	return newTestRouterWithLogger(t, &spyLogger{})
}

func newTestRouterWithLogger(t *testing.T, appLogger logger.Logger) http.Handler {
	t.Helper()
	return newTestRouterWithAvatarStorage(t, appLogger, &routerAvatarStorage{})
}

func newTestRouterWithAvatarStorage(t *testing.T, appLogger logger.Logger, storage filesusecase.AvatarStorage) http.Handler {
	t.Helper()

	authHandler, users := newTestAuthHandlerWithUsers(t, appLogger)

	filmsRepo, err := filmsservice.NewSeededService()
	require.NoError(t, err)

	listFilms, err := filmsusecase.NewListFilmsUseCase(filmsRepo, appLogger)
	require.NoError(t, err)

	getFilm, err := filmsusecase.NewGetFilmUseCase(filmsRepo, appLogger)
	require.NoError(t, err)

	listCollections, err := filmsusecase.NewListCollectionsUseCase(filmsRepo, appLogger)
	require.NoError(t, err)

	getCollection, err := filmsusecase.NewGetCollectionUseCase(filmsRepo, appLogger)
	require.NoError(t, err)

	filmsHandler, err := films.NewHandler(films.HandlerUseCases{
		ListFilms:       listFilms,
		GetFilm:         getFilm,
		ListCollections: listCollections,
		GetCollection:   getCollection,
	}, appLogger)
	require.NoError(t, err)

	uploadAvatar, err := filesusecase.NewUploadAvatarUseCase(storage, users)
	require.NoError(t, err)
	getAvatarURL, err := filesusecase.NewGetAvatarURLUseCase(&routerAvatarURLStorage{})
	require.NoError(t, err)
	filesHandler, err := filesdelivery.NewHandler(uploadAvatar, getAvatarURL, appLogger)
	require.NoError(t, err)

	return router.NewRouter(
		authHandler,
		filmsHandler,
		newTestReviewsHandler(t, filmsRepo, appLogger),
		newTestPersonsHandler(t, appLogger),
		filesHandler,
		testOrigin,
		appLogger,
	)
}

func newTestAuthHandler(t *testing.T, appLogger logger.Logger) *auth.Handler {
	t.Helper()
	handler, _ := newTestAuthHandlerWithUsers(t, appLogger)
	return handler
}

func newTestAuthHandlerWithUsers(t *testing.T, appLogger logger.Logger) (*auth.Handler, *authservice.InMemoryUserRepo) {
	t.Helper()

	users := authservice.NewInMemoryUserRepo()
	sessions := authservice.NewInMemorySessionStore(time.Hour)
	passwordService := authservice.BcryptPasswordService{}

	register, err := authusecase.NewRegisterUserUseCase(users, sessions, passwordService, appLogger)
	require.NoError(t, err)

	login, err := authusecase.NewLoginUserUseCase(users, sessions, passwordService, appLogger)
	require.NoError(t, err)

	logout, err := authusecase.NewLogoutUserUseCase(sessions, appLogger)
	require.NoError(t, err)

	authenticate, err := authusecase.NewAuthenticateUserUseCase(sessions, appLogger)
	require.NoError(t, err)

	getUser, err := authusecase.NewGetUserUseCase(users, appLogger)
	require.NoError(t, err)

	handler, err := auth.NewHandler(auth.HandlerUseCases{
		Register:     register,
		Login:        login,
		Logout:       logout,
		Authenticate: authenticate,
		GetUser:      getUser,
	}, appLogger)
	require.NoError(t, err)

	return handler, users
}

func newTestReviewsHandler(t *testing.T, films domain.FilmRepository, appLogger logger.Logger) *reviewsdelivery.Handler {
	t.Helper()

	repo := reviewsservice.NewInMemoryReviewRepo()

	list, err := reviewsusecase.NewListReviewsUseCase(repo, films, appLogger)
	require.NoError(t, err)

	create, err := reviewsusecase.NewCreateReviewUseCase(repo, films, appLogger)
	require.NoError(t, err)

	update, err := reviewsusecase.NewUpdateReviewUseCase(repo, appLogger)
	require.NoError(t, err)

	del, err := reviewsusecase.NewDeleteReviewUseCase(repo, appLogger)
	require.NoError(t, err)

	handler, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   list,
		Create: create,
		Update: update,
		Delete: del,
	}, appLogger)
	require.NoError(t, err)

	return handler
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

type fakePersonsRepo struct{}

func (f *fakePersonsRepo) Create(
	ctx context.Context,
	input domain.PersonInput,
) (domain.PersonID, error) {
	return 1, nil
}

func (f *fakePersonsRepo) GetByID(
	ctx context.Context,
	id domain.PersonID,
) (*domain.Person, error) {
	return nil, nil
}

func (f *fakePersonsRepo) List(
	ctx context.Context,
	limit, offset int,
) ([]domain.Person, int, error) {
	return []domain.Person{}, 0, nil
}

func (f *fakePersonsRepo) Update(
	ctx context.Context,
	id domain.PersonID,
	input domain.PersonInput,
) (*domain.Person, error) {
	return nil, nil
}

func (f *fakePersonsRepo) Delete(
	ctx context.Context,
	id domain.PersonID,
) error {
	return nil
}

func newTestPersonsHandler(t *testing.T, appLogger logger.Logger) *personsdelivery.Handler {
	t.Helper()

	repo := &fakePersonsRepo{}

	create, err := personusecase.NewCreatePersonUseCase(repo, appLogger)
	require.NoError(t, err)

	get, err := personusecase.NewGetPersonUseCase(repo, appLogger)
	require.NoError(t, err)

	list, err := personusecase.NewListPersonsUseCase(repo, appLogger)
	require.NoError(t, err)

	update, err := personusecase.NewUpdatePersonUseCase(repo, appLogger)
	require.NoError(t, err)

	del, err := personusecase.NewDeletePersonUseCase(repo, appLogger)
	require.NoError(t, err)

	handler, err := personsdelivery.NewHandler(personsdelivery.HandlerUseCases{
		CreatePerson: create,
		GetPerson:    get,
		ListPersons:  list,
		UpdatePerson: update,
		DeletePerson: del,
	}, appLogger)
	require.NoError(t, err)

	return handler
}
