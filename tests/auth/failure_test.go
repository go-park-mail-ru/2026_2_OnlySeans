package auth_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var errBoom = errors.New("boom: connection refused 10.0.0.5:5432")

func newMockAuthHandler(t *testing.T, users domain.UserRepository, sessions domain.SessionStore, hasher domain.PasswordHasher) *authdelivery.Handler {
	t.Helper()
	appLogger := newTestLogger()
	register, err := authusecase.NewRegisterUserUseCase(users, sessions, hasher, appLogger)
	require.NoError(t, err)
	login, err := authusecase.NewLoginUserUseCase(users, sessions, &authservice.BcryptPasswordService{}, appLogger)
	require.NoError(t, err)
	logout, err := authusecase.NewLogoutUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	authenticate, err := authusecase.NewAuthenticateUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	get, err := authusecase.NewGetUserUseCase(users, appLogger)
	require.NoError(t, err)
	handler, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{Register: register, Login: login, Logout: logout, Authenticate: authenticate, GetUser: get}, appLogger)
	require.NoError(t, err)
	return handler
}

func TestHandler_Register_StorageFailures(t *testing.T) {
	for _, stage := range []string{"create", "read", "session"} {
		t.Run(stage, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserRepository(ctrl)
			sessions := mocks.NewMockSessionStore(ctrl)
			hasher := mocks.NewMockPasswordHasher(ctrl)
			calls := []any{hasher.EXPECT().Hash("Password1").Return("hash", nil)}
			if stage == "create" {
				calls = append(calls, users.EXPECT().Create(gomock.Any(), "anna@example.com", "anna", "hash").Return(domain.UserID(0), errBoom))
			} else {
				calls = append(calls, users.EXPECT().Create(gomock.Any(), "anna@example.com", "anna", "hash").Return(domain.UserID(7), nil))
				if stage == "read" {
					calls = append(calls, users.EXPECT().GetUser(gomock.Any(), domain.UserID(7)).Return(nil, errBoom))
				} else {
					calls = append(calls, users.EXPECT().GetUser(gomock.Any(), domain.UserID(7)).Return(&domain.User{ID: 7}, nil), sessions.EXPECT().Create(gomock.Any(), domain.UserID(7)).Return(nil, errBoom))
				}
			}
			gomock.InOrder(calls...)
			handler := newMockAuthHandler(t, users, sessions, hasher)
			rec := rawRequest(handler.Register, http.MethodPost, validRegisterBody)
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
			assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
			assert.Empty(t, rec.Result().Cookies())
		})
	}
}

func TestHandler_Logout_StorageFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	sessions := mocks.NewMockSessionStore(ctrl)
	sessions.EXPECT().Delete(gomock.Any(), "sid").Return(errBoom)
	handler := newMockAuthHandler(t, mocks.NewMockUserRepository(ctrl), sessions, mocks.NewMockPasswordHasher(ctrl))
	rec := rawRequest(handler.Logout, http.MethodPost, "", &http.Cookie{Name: authdelivery.SessionCookieName, Value: "sid"})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, rec.Body.String())
	assert.Empty(t, rec.Result().Cookies())
}

func TestHandler_Authenticate(t *testing.T) {
	for _, tc := range []struct {
		name            string
		getErr, wantErr error
	}{
		{"found", nil, nil},
		{"not found", domain.ErrSessionNotFound, domain.ErrUnauthorized},
		{"storage failure", errBoom, domain.ErrInternal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			sessions := mocks.NewMockSessionStore(ctrl)
			sessions.EXPECT().Get(gomock.Any(), "sid").Return(&domain.Session{ID: "sid", UserID: 7}, tc.getErr)
			handler := newMockAuthHandler(t, mocks.NewMockUserRepository(ctrl), sessions, mocks.NewMockPasswordHasher(ctrl))
			id, err := handler.Authenticate(context.Background(), "sid")
			require.ErrorIs(t, err, tc.wantErr)
			if err == nil {
				require.Equal(t, domain.UserID(7), id)
			} else {
				require.Zero(t, id)
			}
		})
	}
}
