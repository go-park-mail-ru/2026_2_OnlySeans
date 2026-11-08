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
	"github.com/stretchr/testify/require"
)

type uploadStorage struct {
	uploads   int
	deletes   int
	key       string
	uploadErr error
	deleteErr error
}

func (s *uploadStorage) Upload(_ context.Context, key, _ string, _ io.ReadSeeker) error {
	s.uploads++
	s.key = key
	return s.uploadErr
}
func (s *uploadStorage) Delete(ctx context.Context, key string) error {
	s.deletes++
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if key != s.key {
		return errors.New("cleanup key differs from uploaded key")
	}
	return s.deleteErr
}

type avatarRepo struct {
	calls int
	owner domain.UserID
	file  domain.AvatarFile
	err   error
}

func (r *avatarRepo) SaveAvatar(_ context.Context, userID domain.UserID, file domain.AvatarFile) error {
	r.calls++
	r.owner, r.file = userID, file
	return r.err
}

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
			storage := &uploadStorage{uploadErr: tt.uploadErr, deleteErr: tt.cleanupErr}
			repo := &avatarRepo{err: tt.saveErr}
			uc, err := filesusecase.NewUploadAvatarUseCase(storage, repo, testAvatarOptions(), testFilesLogger(t))
			require.NoError(t, err)
			result, err := uc.Run(t.Context(), filesusecase.UploadAvatarInput{
				UserID: tt.owner,
				Body:   bytes.NewReader(imageData.Bytes()),
				Size:   int64(imageData.Len()),
			})
			require.Equal(t, tt.wantUploads, storage.uploads)
			require.Equal(t, tt.wantSaves, repo.calls)
			require.Equal(t, tt.wantDeletes, storage.deletes)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, result)
				if tt.cleanupErr != nil {
					require.ErrorIs(t, err, tt.cleanupErr)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, storage.key, result.Filename)
				require.Equal(t, tt.owner, repo.owner)
				require.Equal(t, result.Filename, repo.file.Filename)
				require.Equal(t, "image/png", repo.file.ContentType)
				require.Equal(t, int64(imageData.Len()), repo.file.Size)
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
			storage := &uploadStorage{}
			uc, err := filesusecase.NewUploadAvatarUseCase(storage, &avatarRepo{}, options, testFilesLogger(t))
			require.NoError(t, err)
			_, err = uc.Run(t.Context(), filesusecase.UploadAvatarInput{
				UserID: 7,
				Body:   bytes.NewReader(data.Bytes()),
				Size:   int64(data.Len()),
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Zero(t, storage.uploads)
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, storage.uploads)
			}
		})
	}
}
