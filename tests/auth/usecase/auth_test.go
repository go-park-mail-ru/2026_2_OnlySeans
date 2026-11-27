package usecase_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/tests/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/auth/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/metrics"
)

func TestLoginPreservesEmailCase(t *testing.T) {
	tests := []struct {
		name        string
		storedEmail string
		email       string
		lookupEmail string
		wantErr     error
	}{
		{
			name:        "mixed case",
			storedEmail: "Mixed@example.test",
			email:       "Mixed@example.test",
			lookupEmail: "Mixed@example.test",
		},
		{
			name:        "lower case",
			storedEmail: "reader@example.test",
			email:       "reader@example.test",
			lookupEmail: "reader@example.test",
		},
		{
			name:        "upper case",
			storedEmail: "READER@EXAMPLE.TEST",
			email:       "READER@EXAMPLE.TEST",
			lookupEmail: "READER@EXAMPLE.TEST",
		},
		{
			name:        "surrounding whitespace",
			storedEmail: "Mixed@example.test",
			email:       "  Mixed@example.test\t",
			lookupEmail: "Mixed@example.test",
		},
		{
			name:        "different case remains a different email",
			storedEmail: "Mixed@example.test",
			email:       "mixed@example.test",
			lookupEmail: "mixed@example.test",
			wantErr:     domain.ErrInvalidCredentials,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserClient(ctrl)
			verifier := mocks.NewMockPasswordVerifier(ctrl)
			sessions := mocks.NewMockSessionUseCase(ctrl)
			user := domain.User{
				ID:           7,
				Role:         "reader",
				PasswordHash: "stored hash",
			}
			session := domain.Session{
				ID:     uuid.New(),
				UserID: user.ID,
				Role:   string(user.Role),
			}

			var lookupErr error
			if tt.wantErr != nil {
				lookupErr = domain.ErrUserNotFound
			}

			users.EXPECT().GetUserByEmail(t.Context(), tt.lookupEmail).Return(user, lookupErr)

			if tt.wantErr == nil {
				gomock.InOrder(
					verifier.EXPECT().Verify("Password1", "stored hash").Return(nil),
					sessions.EXPECT().Create(t.Context(), user.ID, string(user.Role)).Return(session, nil),
				)
			}

			uc := usecase.NewAuthUseCase(users, sessions, verifier, metrics.New())

			got, err := uc.Login(t.Context(), tt.email, "Password1")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, &session, got)
		})
	}
}
