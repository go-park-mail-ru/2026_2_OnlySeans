package files_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

const (
	testMaxAvatarSize      = 5 << 20
	testMaxAvatarDimension = 4096
)

func testAvatarOptions() filesusecase.UploadAvatarOptions {
	return filesusecase.UploadAvatarOptions{
		MaxSize:        testMaxAvatarSize,
		MaxDimension:   testMaxAvatarDimension,
		CleanupTimeout: 5 * time.Second,
	}
}

type accountAvatarRepo struct{}

func (*accountAvatarRepo) GetAvatar(context.Context, domain.UserID) (*domain.AvatarFile, error) {
	return nil, domain.ErrFileNotFound
}
func (*accountAvatarRepo) DeleteAvatar(context.Context, domain.UserID) error { return nil }
func testAccountAvatar(t *testing.T, storage filesusecase.AvatarURLStorage) *filesusecase.AccountAvatarUseCase {
	t.Helper()
	uc, err := filesusecase.NewAccountAvatarUseCase(&accountAvatarRepo{}, storage, testFilesLogger(t))
	require.NoError(t, err)
	return uc
}

func testFilesLogger(t *testing.T) logger.Logger {
	t.Helper()
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
	require.NoError(t, err)
	return appLogger
}
