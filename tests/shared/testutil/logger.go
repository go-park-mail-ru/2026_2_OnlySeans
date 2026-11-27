package testutil

import (
	"io"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

func DiscardLogger() logger.Logger {
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
	if err != nil {
		panic(err)
	}

	return appLogger
}
