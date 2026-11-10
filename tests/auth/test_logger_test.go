package auth_test

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func newTestLogger() logger.Logger {
	return testutil.DiscardLogger()
}
