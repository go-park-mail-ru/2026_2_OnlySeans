package files_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/stretchr/testify/require"
)

type accountAvatarStub struct {
	file   *domain.AvatarFile
	err    error
	userID domain.UserID
}

func (r *accountAvatarStub) GetAvatar(_ context.Context, id domain.UserID) (*domain.AvatarFile, error) {
	r.userID = id
	return r.file, r.err
}
func (r *accountAvatarStub) DeleteAvatar(_ context.Context, id domain.UserID) error {
	r.userID = id
	return r.err
}

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
			repo := &accountAvatarStub{file: &domain.AvatarFile{Filename: "test.png"}, err: tt.repoErr}
			storage := &avatarURLStorage{url: "https://example.com/avatar", err: tt.storageErr}
			uc, err := filesusecase.NewAccountAvatarUseCase(repo, storage, testFilesLogger(t))
			require.NoError(t, err)
			url, err := uc.GetURL(t.Context(), tt.userID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, url)
			} else {
				require.NoError(t, err)
				require.Equal(t, storage.url, url)
			}
			if tt.userID == 0 || tt.repoErr != nil {
				require.Zero(t, storage.calls)
			}
			err = uc.Delete(t.Context(), tt.userID)
			switch {
			case tt.userID == 0:
				require.ErrorIs(t, err, domain.ErrUnauthorized)
			case tt.repoErr != nil:
				require.ErrorIs(t, err, tt.repoErr)
			default:
				require.NoError(t, err)
				require.Equal(t, tt.userID, repo.userID)
			}
		})
	}
}
