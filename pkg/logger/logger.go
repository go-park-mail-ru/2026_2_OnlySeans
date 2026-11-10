package logger

import "context"

type LogLevelType string

const (
	LogLevelDebug LogLevelType = "debug"
	LogLevelInfo  LogLevelType = "info"
	LogLevelWarn  LogLevelType = "warn"
	LogLevelError LogLevelType = "error"
)

type Logger interface {
	Info(msg string, fields map[string]any)
	Debug(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
	InfoContext(ctx context.Context, msg string, fields map[string]any)
	DebugContext(ctx context.Context, msg string, fields map[string]any)
	WarnContext(ctx context.Context, msg string, fields map[string]any)
	ErrorContext(ctx context.Context, msg string, fields map[string]any)
}
