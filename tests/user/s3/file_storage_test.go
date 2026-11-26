package s3_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	users3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/mocks"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

func TestGetURL(t *testing.T) {
	failure := errors.New("storage unavailable")

	tests := []struct {
		name       string
		storageURL string
		storageErr error
		wantURL    string
		wantErr    error
	}{
		{
			name:       "signed URL",
			storageURL: "https://media.example/avatar?signature=private",
			wantURL:    "https://media.example/avatar?signature=private",
		},
		{
			name:       "missing file",
			storageErr: shareds3.ErrFileNotFound,
			wantErr:    domain.ErrFileNotFound,
		},
		{
			name:       "wrapped missing file",
			storageErr: fmt.Errorf("head object: %w", shareds3.ErrFileNotFound),
			wantErr:    domain.ErrFileNotFound,
		},
		{
			name:       "storage failure",
			storageErr: failure,
			wantErr:    failure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewMockFileStorage(gomock.NewController(t))
			storage.EXPECT().GetURL(t.Context(), "avatar.png").Return(tt.storageURL, tt.storageErr)
			adapter := users3.NewFileStorage(storage)
			url, err := adapter.GetURL(t.Context(), "avatar.png")
			require.Equal(t, tt.wantURL, url)
			require.ErrorIs(t, err, tt.wantErr)
			require.ErrorIs(t, err, tt.storageErr)
		})
	}
}

func TestFileOperations(t *testing.T) {
	failure := errors.New("storage unavailable")

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "success",
		},
		{
			name: "storage failure",
			err:  failure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewMockFileStorage(gomock.NewController(t))
			data := []byte("image data")
			storage.EXPECT().UploadFile(t.Context(), "avatar.png", "image/png", data).Return(tt.err)
			storage.EXPECT().DeleteFile(t.Context(), "avatar.png").Return(tt.err)

			adapter := users3.NewFileStorage(storage)
			require.ErrorIs(t, adapter.UploadFile(t.Context(), "avatar.png", "image/png", data), tt.err)
			require.ErrorIs(t, adapter.DeleteFile(t.Context(), "avatar.png"), tt.err)
		})
	}
}
