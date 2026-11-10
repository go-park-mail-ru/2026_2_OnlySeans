package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authuc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRegister_Gomock(t *testing.T) {
	input := authuc.RegisterUserInput{Email: "anna@example.com", Username: "Anna", Password: "Password1"}
	avatar := "avatar.png"
	user := &domain.User{ID: 7, Email: input.Email, Username: input.Username, AvatarFilename: &avatar}
	session := &domain.Session{ID: "test-session", UserID: user.ID, ExpiresAt: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)}
	failure := errors.New("dependency unavailable")

	for _, stage := range []string{"success", "hash", "create user", "load user", "nil user", "create session"} {
		t.Run(stage, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserRepository(ctrl)
			sessions := mocks.NewMockSessionStore(ctrl)
			hasher := mocks.NewMockPasswordHasher(ctrl)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var calls []any
			if stage == "hash" {
				calls = append(calls, hasher.EXPECT().Hash(input.Password).Return("", failure))
			} else {
				calls = append(calls, hasher.EXPECT().Hash(input.Password).Return("hashed-password", nil))
				if stage == "create user" {
					calls = append(calls, users.EXPECT().Create(ctx, input.Email, input.Username, "hashed-password").Return(domain.UserID(0), failure))
				} else {
					calls = append(calls, users.EXPECT().Create(ctx, input.Email, input.Username, "hashed-password").Return(user.ID, nil))
					switch stage {
					case "load user":
						calls = append(calls, users.EXPECT().GetUser(ctx, user.ID).Return(nil, failure))
					case "nil user":
						calls = append(calls, users.EXPECT().GetUser(ctx, user.ID).Return(nil, nil))
					default:
						calls = append(calls, users.EXPECT().GetUser(ctx, user.ID).Return(user, nil))
						if stage == "create session" {
							calls = append(calls, sessions.EXPECT().Create(ctx, user.ID).Return(nil, failure))
						} else {
							calls = append(calls, sessions.EXPECT().Create(ctx, user.ID).Return(session, nil))
						}
					}
				}
			}
			gomock.InOrder(calls...)
			uc, err := authuc.NewRegisterUserUseCase(users, sessions, hasher, newTestLogger())
			require.NoError(t, err)
			result, err := uc.Run(ctx, input)
			switch stage {
			case "success":
				require.NoError(t, err)
				require.Equal(t, &authuc.RegisterOutput{
					User:    authuc.UserDTO{ID: user.ID, Email: user.Email, Username: user.Username, AvatarFilename: &avatar},
					Session: authuc.SessionDTO{ID: session.ID, ExpiresAt: session.ExpiresAt},
				}, result)
			case "nil user":
				require.ErrorContains(t, err, "repository returned nil user")
				require.Nil(t, result)
			default:
				require.ErrorIs(t, err, failure)
				require.Nil(t, result)
			}
		})
	}
}

func TestRegister_GomockInvalidInputDoesNotCallDependencies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   authuc.RegisterUserInput
		wantErr error
	}{
		{"password", authuc.RegisterUserInput{Email: "anna@example.com", Username: "Anna", Password: "x"}, domain.ErrInvalidPassword},
		{"email", authuc.RegisterUserInput{Email: "invalid", Username: "Anna", Password: "Password1"}, domain.ErrInvalidEmail},
		{"username", authuc.RegisterUserInput{Email: "anna@example.com", Username: "", Password: "Password1"}, domain.ErrInvalidUsername},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc, err := authuc.NewRegisterUserUseCase(mocks.NewMockUserRepository(ctrl), mocks.NewMockSessionStore(ctrl), mocks.NewMockPasswordHasher(ctrl), newTestLogger())
			require.NoError(t, err)
			result, err := uc.Run(context.Background(), tc.input)
			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, result)
		})
	}
}
