package auth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

var errBoom = errors.New("boom: connection refused 10.0.0.5:5432")

type flakyRepo struct {
	*auth.InMemoryUserRepo
	createErr  error
	getUserErr error
}

func (r flakyRepo) Create(ctx context.Context, email, username, passwordHash string) (auth.UserID, error) {
	if r.createErr != nil {
		return 0, r.createErr
	}
	return r.InMemoryUserRepo.Create(ctx, email, username, passwordHash)
}

func (r flakyRepo) GetUser(ctx context.Context, id auth.UserID) (*auth.User, error) {
	if r.getUserErr != nil {
		return nil, r.getUserErr
	}
	return r.InMemoryUserRepo.GetUser(ctx, id)
}

type flakyStore struct {
	*auth.InMemorySessionStore
	createErr error
	getErr    error
	deleteErr error
}

func (s flakyStore) Create(userID auth.UserID) (*auth.Session, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.InMemorySessionStore.Create(userID)
}

func (s flakyStore) Get(sessionID string) (*auth.Session, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.InMemorySessionStore.Get(sessionID)
}

func (s flakyStore) Delete(sessionID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return s.InMemorySessionStore.Delete(sessionID)
}

func newFlakyUseCase(t *testing.T, repo flakyRepo, store flakyStore) *auth.UseCase {
	t.Helper()

	repo.InMemoryUserRepo = auth.NewInMemoryUserRepo()
	store.InMemorySessionStore = auth.NewInMemorySessionStore(time.Hour)

	uc, err := auth.NewUseCase(repo, store)
	require.NoError(t, err)
	return uc
}

func TestUseCase_Register_StorageFailures(t *testing.T) {
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
			uc := newFlakyUseCase(t, tt.repo, tt.store)

			result, err := uc.Register(context.Background(), "anna@example.com", "anna", "Password1")
			assert.Nil(t, result)
			assert.Equal(t, auth.ErrInternal, err)
		})
	}
}

func TestUseCase_Authenticate_StorageFailure(t *testing.T) {
	uc := newFlakyUseCase(t, flakyRepo{}, flakyStore{getErr: errBoom})

	userID, err := uc.Authenticate(context.Background(), "sid")
	assert.Zero(t, userID)
	assert.Equal(t, auth.ErrInternal, err)
}

func TestUseCase_Logout_StorageFailure(t *testing.T) {
	uc := newFlakyUseCase(t, flakyRepo{}, flakyStore{deleteErr: errBoom})

	assert.Equal(t, auth.ErrInternal, uc.Logout("sid"))
}

func TestUseCase_Login_EmptyPassword(t *testing.T) {
	uc := newTestUseCase(t)
	ctx := context.Background()

	_, err := uc.Register(ctx, "anna@example.com", "anna", "Password1")
	require.NoError(t, err)

	_, err = uc.Login(ctx, "anna@example.com", "")
	assert.ErrorIs(t, err, auth.ErrInvalidPassword)
}

func TestHandler_InternalErrorIsHidden(t *testing.T) {
	tests := []struct {
		name    string
		repo    flakyRepo
		store   flakyStore
		request func(h *auth.Handler) (http.HandlerFunc, string, []*http.Cookie)
	}{
		{
			name: "register",
			repo: flakyRepo{createErr: errBoom},
			request: func(h *auth.Handler) (http.HandlerFunc, string, []*http.Cookie) {
				return h.Register, validRegisterBody, nil
			},
		},
		{
			name:  "logout",
			store: flakyStore{deleteErr: errBoom},
			request: func(h *auth.Handler) (http.HandlerFunc, string, []*http.Cookie) {
				return h.Logout, "", []*http.Cookie{{Name: auth.SessionCookieName, Value: "sid"}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := auth.NewHandler(newFlakyUseCase(t, tt.repo, tt.store))
			require.NoError(t, err)

			handlerFunc, body, cookies := tt.request(h)
			rec := rawRequest(handlerFunc, http.MethodPost, body, cookies...)

			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
			assert.Empty(t, rec.Result().Cookies())
		})
	}
}
