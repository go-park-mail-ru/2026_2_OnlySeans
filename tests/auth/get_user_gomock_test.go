package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authuc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetUser_Gomock(t *testing.T) {
	avatar := "avatar.png"
	user := &domain.User{ID: 7, Email: "anna@example.com", Username: "Anna", Role: domain.RoleEditor, AvatarFilename: &avatar}
	for _, tc := range []struct {
		name             string
		user             *domain.User
		repoErr, wantErr error
	}{
		{name: "success", user: user},
		{name: "not found", repoErr: domain.ErrUserNotFound, wantErr: domain.ErrUserNotFound},
		{name: "database unavailable", repoErr: errors.New("database unavailable")},
		{name: "nil user", wantErr: domain.ErrUserNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			users := mocks.NewMockUserRepository(ctrl)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			users.EXPECT().GetUser(ctx, domain.UserID(7)).Return(tc.user, tc.repoErr).Times(1)
			uc, err := authuc.NewGetUserUseCase(users, newTestLogger())
			require.NoError(t, err)

			result, err := uc.Run(ctx, authuc.GetUserInput{UserID: 7})
			wantErr := tc.wantErr
			if wantErr == nil {
				wantErr = tc.repoErr
			}
			require.ErrorIs(t, err, wantErr)
			if wantErr != nil {
				require.Nil(t, result)
				return
			}
			require.Equal(t, &authuc.GetUserOutput{ID: 7, Email: user.Email, Username: user.Username, Role: user.Role, AvatarFilename: &avatar}, result)
		})
	}
}
