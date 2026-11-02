package auth_test

import "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"

type testLogger struct{}

func (l *testLogger) Info(string, map[string]any)  {}
func (l *testLogger) Debug(string, map[string]any) {}
func (l *testLogger) Warn(string, map[string]any)  {}
func (l *testLogger) Error(string, map[string]any) {}

var _ logger.Logger = (*testLogger)(nil)

func newTestLogger() logger.Logger {
	return &testLogger{}
}
