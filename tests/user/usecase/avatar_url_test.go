package usecase_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/mocks"
)

func TestGetAvatarURLByFilename(t *testing.T) {
	failure := errors.New("storage unavailable")
	filename := "1234567890abcdef1234567890abcdef.png"

	tests := []struct {
		name       string
		filename   string
		key        string
		storageErr error
		wantErr    error
	}{
		{
			name:     "current filename",
			filename: filename,
			key:      filename,
		},
		{
			name:     "legacy filename",
			filename: "12345678-1234-1234-1234-1234567890ab.jpeg",
			key:      "avatars/12345678-1234-1234-1234-1234567890ab.jpeg",
		},
		{
			name:       "missing file",
			filename:   filename,
			key:        filename,
			storageErr: fmt.Errorf("storage: %w", domain.ErrFileNotFound),
			wantErr:    domain.ErrAvatarNotFound,
		},
		{
			name:       "storage failure",
			filename:   filename,
			key:        filename,
			storageErr: failure,
			wantErr:    failure,
		},
		{
			name:     "invalid filename",
			filename: "../avatar.png",
			wantErr:  domain.ErrInvalidAvatarName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			storage := mocks.NewMockFileStorage(ctrl)
			hasher := mocks.NewMockPasswordHasher(ctrl)
			url := "https://media.example/avatar"

			if tt.key != "" {
				storedURL := url
				if tt.storageErr != nil {
					storedURL = ""
				}

				storage.EXPECT().GetURL(t.Context(), tt.key).Return(storedURL, tt.storageErr)
			}

			uc := usecase.NewUserUseCase(nil, nil, storage, hasher, nil)
			got, err := uc.GetAvatarURLByFilename(t.Context(), tt.filename)
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr != nil {
				require.Empty(t, got)
			} else {
				require.Equal(t, url, got)
			}
		})
	}
}
