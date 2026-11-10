package files_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUploadAvatarAttachesToAccount(t *testing.T) {
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	databaseErr := errors.New("database unavailable")
	uploadErr := errors.New("object storage unavailable")
	cleanupErr := errors.New("cleanup unavailable")
	tests := []struct {
		name        string
		owner       domain.UserID
		saveErr     error
		uploadErr   error
		cleanupErr  error
		wantErr     error
		wantUploads int
		wantSaves   int
		wantDeletes int
	}{
		{
			name: "avatar metadata saved for session user", owner: 7,
			wantUploads: 1, wantSaves: 1,
		},
		{
			name: "anonymous request rejected", wantErr: domain.ErrUnauthorized,
		},
		{
			name: "storage fails before metadata", owner: 7, uploadErr: uploadErr, wantErr: uploadErr,
			wantUploads: 1,
		},
		{
			name: "metadata failure removes new object", owner: 7, saveErr: databaseErr, wantErr: databaseErr,
			wantUploads: 1, wantSaves: 1, wantDeletes: 1,
		},
		{
			name: "deleted account removes new object", owner: 7, saveErr: domain.ErrUserNotFound, wantErr: domain.ErrUserNotFound,
			wantUploads: 1, wantSaves: 1, wantDeletes: 1,
		},
		{
			name: "cleanup error retains original error", owner: 7, saveErr: databaseErr, cleanupErr: cleanupErr, wantErr: databaseErr,
			wantUploads: 1, wantSaves: 1, wantDeletes: 1,
		},
		{
			name: "ambiguous commit preserves possibly attached object", owner: 7, saveErr: domain.ErrAvatarCommitUnknown, wantErr: domain.ErrAvatarCommitUnknown,
			wantUploads: 1, wantSaves: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			storage := mocks.NewMockAvatarStorage(ctrl)
			repo := mocks.NewMockAvatarWriter(ctrl)
			var key string
			var metadata domain.AvatarFile
			var calls []any
			if tt.wantUploads > 0 {
				calls = append(calls, storage.EXPECT().Upload(gomock.Any(), gomock.Any(), "image/png", gomock.Any()).DoAndReturn(func(_ context.Context, name, mime string, body io.ReadSeeker) error {
					key = name
					actual, err := io.ReadAll(body)
					require.NoError(t, err)
					require.Equal(t, imageData.Bytes(), actual)
					return tt.uploadErr
				}).Times(tt.wantUploads))
			}
			if tt.wantSaves > 0 {
				calls = append(calls, repo.EXPECT().SaveAvatar(gomock.Any(), tt.owner, gomock.Any()).DoAndReturn(func(_ context.Context, _ domain.UserID, file domain.AvatarFile) error {
					metadata = file
					require.Equal(t, key, file.Filename)
					return tt.saveErr
				}).Times(tt.wantSaves))
			}
			if tt.wantDeletes > 0 {
				calls = append(calls, storage.EXPECT().Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, name string) error {
					require.NoError(t, ctx.Err())
					require.Equal(t, key, name)
					return tt.cleanupErr
				}).Times(tt.wantDeletes))
			}
			gomock.InOrder(calls...)
			uc, err := filesusecase.NewUploadAvatarUseCase(storage, repo, testAvatarOptions(), testFilesLogger(t))
			require.NoError(t, err)
			result, err := uc.Run(t.Context(), filesusecase.UploadAvatarInput{
				UserID: tt.owner,
				Body:   bytes.NewReader(imageData.Bytes()),
				Size:   int64(imageData.Len()),
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, result)
				if tt.cleanupErr != nil {
					require.ErrorIs(t, err, tt.cleanupErr)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, key, result.Filename)
				require.Equal(t, result.Filename, metadata.Filename)
				require.Equal(t, "image/png", metadata.ContentType)
				require.Equal(t, int64(imageData.Len()), metadata.Size)
			}
		})
	}
}

func TestUploadAvatarConfiguredLimits(t *testing.T) {
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	tests := []struct {
		name         string
		maxSize      int64
		maxDimension int
		wantErr      error
	}{
		{
			name:         "exact limits accept image",
			maxSize:      int64(data.Len()),
			maxDimension: 2,
		},
		{
			name:         "configured byte limit rejects image",
			maxSize:      int64(data.Len() - 1),
			maxDimension: 2,
			wantErr:      filesusecase.ErrAvatarTooLarge,
		},
		{
			name:         "configured dimension limit rejects image",
			maxSize:      int64(data.Len()),
			maxDimension: 1,
			wantErr:      filesusecase.ErrAvatarDimensionsTooLarge,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := testAvatarOptions()
			options.MaxSize = tt.maxSize
			options.MaxDimension = tt.maxDimension
			ctrl := gomock.NewController(t)
			storage := mocks.NewMockAvatarStorage(ctrl)
			repo := mocks.NewMockAvatarWriter(ctrl)
			if tt.wantErr == nil {
				upload := storage.EXPECT().Upload(gomock.Any(), gomock.Any(), "image/png", gomock.Any()).Return(nil)
				repo.EXPECT().SaveAvatar(gomock.Any(), domain.UserID(7), gomock.Any()).Return(nil).After(upload)
			}
			uc, err := filesusecase.NewUploadAvatarUseCase(storage, repo, options, testFilesLogger(t))
			require.NoError(t, err)
			_, err = uc.Run(t.Context(), filesusecase.UploadAvatarInput{
				UserID: 7,
				Body:   bytes.NewReader(data.Bytes()),
				Size:   int64(data.Len()),
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
