package files_test

import (
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAccountAvatarUseCase(t *testing.T) {
	failure := errors.New("database unavailable")
	tests := []struct {
		name       string
		userID     domain.UserID
		repoErr    error
		storageErr error
		wantErr    error
	}{
		{
			name:   "current avatar",
			userID: 7,
		},
		{
			name:    "anonymous",
			wantErr: domain.ErrUnauthorized,
		},
		{
			name:    "missing avatar",
			userID:  7,
			repoErr: domain.ErrFileNotFound,
			wantErr: domain.ErrFileNotFound,
		},
		{
			name:    "missing user",
			userID:  7,
			repoErr: domain.ErrUserNotFound,
			wantErr: domain.ErrUserNotFound,
		},
		{
			name:    "repository failure",
			userID:  7,
			repoErr: failure,
			wantErr: failure,
		},
		{
			name:       "storage failure",
			userID:     7,
			storageErr: failure,
			wantErr:    failure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockAvatarRepository(ctrl)
			storage := mocks.NewMockAvatarURLStorage(ctrl)
			if tt.userID > 0 {
				lookup := repo.EXPECT().GetAvatar(gomock.Any(), tt.userID).Return(&domain.AvatarFile{Filename: "test.png"}, tt.repoErr)
				if tt.repoErr == nil {
					storage.EXPECT().GetURL(gomock.Any(), "test.png").Return("https://example.com/avatar", tt.storageErr).After(lookup)
				}
				repo.EXPECT().DeleteAvatar(gomock.Any(), tt.userID).Return(tt.repoErr)
			}
			uc, err := filesusecase.NewAccountAvatarUseCase(repo, storage, testFilesLogger(t))
			require.NoError(t, err)
			url, err := uc.GetURL(t.Context(), tt.userID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, url)
			} else {
				require.NoError(t, err)
				require.Equal(t, "https://example.com/avatar", url)
			}
			err = uc.Delete(t.Context(), tt.userID)
			switch {
			case tt.userID == 0:
				require.ErrorIs(t, err, domain.ErrUnauthorized)
			case tt.repoErr != nil:
				require.ErrorIs(t, err, tt.repoErr)
			default:
				require.NoError(t, err)
			}
		})
	}
}
