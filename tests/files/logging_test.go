package files_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUploadAvatarLogging(t *testing.T) {
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	tests := []struct {
		name       string
		owner      domain.UserID
		uploadErr  error
		saveErr    error
		cleanupErr error
		message    string
		level      string
		errorCount int
	}{
		{
			name:    "successful attachment",
			owner:   7,
			message: "account avatar saved",
			level:   "INFO",
		},
		{
			name:    "unauthorized upload",
			message: "avatar upload rejected: invalid user",
			level:   "WARN",
		},
		{
			name:       "storage failure logged once",
			owner:      7,
			uploadErr:  errors.New("storage unavailable"),
			message:    "upload avatar object failed",
			level:      "ERROR",
			errorCount: 1,
		},
		{
			name:       "metadata failure and successful compensation",
			owner:      7,
			saveErr:    errors.New("database unavailable"),
			message:    "unlinked avatar object removed",
			level:      "DEBUG",
			errorCount: 1,
		},
		{
			name:       "compensation failure is a separate event",
			owner:      7,
			saveErr:    errors.New("database unavailable"),
			cleanupErr: errors.New("cleanup unavailable"),
			message:    "remove unlinked avatar object failed",
			level:      "ERROR",
			errorCount: 2,
		},
		{
			name:       "unknown commit keeps object",
			owner:      7,
			saveErr:    domain.ErrCommitUnknown,
			message:    "avatar object retained: commit outcome unknown",
			level:      "WARN",
			errorCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			appLogger, err := logger.NewLogger(&logs, logger.LogLevelDebug, "json")
			require.NoError(t, err)
			ctrl := gomock.NewController(t)
			storage := mocks.NewMockAvatarStorage(ctrl)
			repo := mocks.NewMockAvatarWriter(ctrl)
			if tt.owner > 0 {
				upload := storage.EXPECT().Upload(gomock.Any(), gomock.Any(), "image/png", gomock.Any()).Return(tt.uploadErr)
				if tt.uploadErr == nil {
					save := repo.EXPECT().SaveAvatar(gomock.Any(), tt.owner, gomock.Any()).Return(tt.saveErr).After(upload)
					if tt.saveErr != nil && !errors.Is(tt.saveErr, domain.ErrCommitUnknown) {
						storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(tt.cleanupErr).After(save)
					}
				}
			}
			uc, err := filesusecase.NewUploadAvatarUseCase(storage, repo, testAvatarOptions(), appLogger)
			require.NoError(t, err)
			_, _ = uc.Run(t.Context(), filesusecase.UploadAvatarInput{
				UserID: tt.owner,
				Body:   bytes.NewReader(imageData.Bytes()),
				Size:   int64(imageData.Len()),
			})
			require.Contains(t, logs.String(), `"msg":"`+tt.message+`"`)
			require.Contains(t, logs.String(), `"level":"`+tt.level+`"`)
			require.Equal(t, tt.errorCount, strings.Count(logs.String(), `"level":"ERROR"`))
		})
	}
}

func TestAvatarURLLogging(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		err      error
		message  string
		level    string
	}{
		{
			name:     "signed URL is not logged",
			filename: strings.Repeat("a", 32) + ".png",
			message:  "avatar URL generated",
			level:    "DEBUG",
		},
		{
			name:     "invalid filename does not expose input",
			filename: "untrusted-input-do-not-log",
			message:  "avatar URL request rejected: invalid filename",
			level:    "WARN",
		},
		{
			name:     "missing image is not an internal error",
			filename: strings.Repeat("a", 32) + ".png",
			err:      domain.ErrFileNotFound,
			message:  "avatar object not found",
			level:    "DEBUG",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const signedURL = "https://example.com/image?X-Amz-Signature=private-signature"
			var logs bytes.Buffer
			appLogger, err := logger.NewLogger(&logs, logger.LogLevelDebug, "json")
			require.NoError(t, err)
			storage := mocks.NewMockAvatarURLStorage(gomock.NewController(t))
			if tt.filename != "untrusted-input-do-not-log" {
				storage.EXPECT().GetURL(gomock.Any(), tt.filename).Return(signedURL, tt.err)
			}
			uc, err := filesusecase.NewGetAvatarURLUseCase(storage, appLogger)
			require.NoError(t, err)
			_, _ = uc.Run(t.Context(), tt.filename)
			require.Contains(t, logs.String(), `"msg":"`+tt.message+`"`)
			require.Contains(t, logs.String(), `"level":"`+tt.level+`"`)
			require.NotContains(t, logs.String(), signedURL)
			require.NotContains(t, logs.String(), "private-signature")
			require.NotContains(t, logs.String(), "untrusted-input-do-not-log")
		})
	}
}
