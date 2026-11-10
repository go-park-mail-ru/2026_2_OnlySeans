package files_test

import (
	"io"
	"testing"
	"time"

	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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

func testAccountAvatar(t *testing.T, storage filesusecase.AvatarURLStorage) *filesusecase.AccountAvatarUseCase {
	t.Helper()
	uc, err := filesusecase.NewAccountAvatarUseCase(mocks.NewMockAvatarRepository(gomock.NewController(t)), storage, testFilesLogger(t))
	require.NoError(t, err)
	return uc
}

func testFilesLogger(t *testing.T) logger.Logger {
	t.Helper()
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
	require.NoError(t, err)
	return appLogger
}
