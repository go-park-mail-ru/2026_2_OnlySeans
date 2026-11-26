package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/tests/mocks"
	passwordutil "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/utils"
)

func TestLoginErrors(t *testing.T) {
	backendErr := errors.New("database unavailable")
	tests := []struct {
		name        string
		email       string
		password    string
		userErr     error
		verifyErr   error
		wantErr     error
		userCalls   int
		verifyCalls int
	}{
		{
			name:        "wrong password",
			email:       "reader@example.com",
			password:    "wrong-password",
			verifyErr:   passwordutil.ErrPasswordMismatch,
			wantErr:     domain.ErrInvalidCredentials,
			userCalls:   1,
			verifyCalls: 1,
		},
		{
			name:      "unknown email",
			email:     "unknown@example.com",
			password:  "Password1",
			userErr:   domain.ErrUserNotFound,
			wantErr:   domain.ErrInvalidCredentials,
			userCalls: 1,
		},
		{
			name:    "empty password",
			email:   "reader@example.com",
			wantErr: utils.ErrInvalidPassword,
		},
		{
			name:     "invalid email",
			email:    "invalid-email",
			password: "Password1",
			wantErr:  utils.ErrInvalidEmail,
		},
		{
			name:      "user service failure",
			email:     "reader@example.com",
			password:  "Password1",
			userErr:   backendErr,
			wantErr:   backendErr,
			userCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserClient(ctrl)
			verifier := mocks.NewMockPasswordVerifier(ctrl)
			sessions := mocks.NewMockSessionUseCase(ctrl)

			if tt.userCalls != 0 {
				users.EXPECT().GetUserByEmail(t.Context(), tt.email).Return(domain.User{
					PasswordHash: "stored hash",
				}, tt.userErr).Times(tt.userCalls)
			}

			if tt.verifyCalls != 0 {
				verifier.EXPECT().Verify(tt.password, "stored hash").Return(tt.verifyErr).Times(tt.verifyCalls)
			}

			uc := usecase.NewAuthUseCase(users, sessions, verifier)
			result, err := uc.Login(t.Context(), tt.email, tt.password)
			require.Nil(t, result)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
