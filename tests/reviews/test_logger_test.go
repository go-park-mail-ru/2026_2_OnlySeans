package reviews_test

import "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"

type reviewsTestLogger struct{}

func (l *reviewsTestLogger) Info(string, map[string]any)  {}
func (l *reviewsTestLogger) Debug(string, map[string]any) {}
func (l *reviewsTestLogger) Warn(string, map[string]any)  {}
func (l *reviewsTestLogger) Error(string, map[string]any) {}

var _ logger.Logger = (*reviewsTestLogger)(nil)

func newReviewsTestLogger() logger.Logger {
	return &reviewsTestLogger{}
}
