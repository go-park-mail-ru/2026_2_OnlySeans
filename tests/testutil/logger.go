package testutil

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"io"
)

func DiscardLogger() logger.Logger {
	log, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
	if err != nil {
		panic(err)
	}
	return log
}
