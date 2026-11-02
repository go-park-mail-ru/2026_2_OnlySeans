package logger

type Logger interface {
	Info(msg string, fields map[string]any)
	Debug(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}
