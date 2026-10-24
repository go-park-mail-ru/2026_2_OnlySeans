package auth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
)

var errBoom = errors.New("boom: connection refused 10.0.0.5:5432")

type flakyRepo struct {
	*authservice.InMemoryUserRepo
	createErr  error
	getUserErr error
}

func (r flakyRepo) Create(ctx context.Context, email, username, passwordHash string) (domain.UserID, error) {
	if r.createErr != nil {
		return 0, r.createErr
	}
	return r.InMemoryUserRepo.Create(ctx, email, username, passwordHash)
}

func (r flakyRepo) GetUser(ctx context.Context, id domain.UserID) (*domain.User, error) {
	if r.getUserErr != nil {
		return nil, r.getUserErr
	}
	return r.InMemoryUserRepo.GetUser(ctx, id)
}

type flakyStore struct {
	*authservice.InMemorySessionStore
	createErr error
	getErr    error
	deleteErr error
}

func (s flakyStore) Create(ctx context.Context, userID domain.UserID) (*domain.Session, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.InMemorySessionStore.Create(ctx, userID)
}

func (s flakyStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.InMemorySessionStore.Get(ctx, sessionID)
}

func (s flakyStore) Delete(ctx context.Context, sessionID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.InMemorySessionStore.Delete(ctx, sessionID)
}

func newFlakyHandler(t *testing.T, repo flakyRepo, store flakyStore) (*authdelivery.Handler, flakyStore) {
	t.Helper()

	repo.InMemoryUserRepo = authservice.NewInMemoryUserRepo()
	store.InMemorySessionStore = authservice.NewInMemorySessionStore(time.Hour)

	passwordService := authservice.BcryptPasswordService{}
	register, err := authusecase.NewRegisterUserUseCase(repo, store, passwordService)
	require.NoError(t, err)
	login, err := authusecase.NewLoginUserUseCase(repo, store, passwordService)
	require.NoError(t, err)
	logout, err := authusecase.NewLogoutUserUseCase(store)
	require.NoError(t, err)
	authenticate, err := authusecase.NewAuthenticateUserUseCase(store)
	require.NoError(t, err)
	getUser, err := authusecase.NewGetUserUseCase(repo)
	require.NoError(t, err)

	h, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{
		Register:     register,
		Login:        login,
		Logout:       logout,
		Authenticate: authenticate,
		GetUser:      getUser,
	})
	require.NoError(t, err)
	return h, store
}

func TestHandler_Register_StorageFailures(t *testing.T) {
	tests := []struct {
		name  string
		repo  flakyRepo
		store flakyStore
	}{
		{"репозиторий не создал пользователя", flakyRepo{createErr: errBoom}, flakyStore{}},
		{"репозиторий не вернул пользователя", flakyRepo{getUserErr: errBoom}, flakyStore{}},
		{"хранилище не создало сессию", flakyRepo{}, flakyStore{createErr: errBoom}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newFlakyHandler(t, tt.repo, tt.store)

			rec := rawRequest(h.Register, http.MethodPost, validRegisterBody)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
			assert.Empty(t, rec.Result().Cookies())
		})
	}
}

func TestHandler_Logout_StorageFailure(t *testing.T) {
	h, _ := newFlakyHandler(t, flakyRepo{}, flakyStore{deleteErr: errBoom})
	cookie := &http.Cookie{Name: authdelivery.SessionCookieName, Value: "sid"}

	rec := rawRequest(h.Logout, http.MethodPost, "", cookie)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
	assert.Empty(t, rec.Result().Cookies())
}

func TestHandler_Authenticate(t *testing.T) {
	tests := []struct {
		name    string
		getErr  error
		wantErr error
	}{
		{"сессия найдена", nil, nil},
		{"сессия не найдена", authdelivery.ErrSessionNotFound, authdelivery.ErrUnauthorized},
		{"сбой хранилища скрыт", errBoom, authdelivery.ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, store := newFlakyHandler(t, flakyRepo{}, flakyStore{getErr: tt.getErr})

			sid := "sid"
			if tt.getErr == nil {
				session, err := store.InMemorySessionStore.Create(context.Background(), 7)
				require.NoError(t, err)
				sid = session.ID
			}

			userID, err := h.Authenticate(context.Background(), sid)

			if tt.wantErr != nil {
				assert.Zero(t, userID)
				assert.Equal(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, domain.UserID(7), userID)
		})
	}
}
