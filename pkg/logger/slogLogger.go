package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strings"
)

type SlogLogger struct {
	logger *slog.Logger
}

func NewLogger(w io.Writer, level LogLevelType, format string) (*SlogLogger, error) {
	var lvl slog.Level

	switch LogLevelType(strings.ToLower(strings.TrimSpace(string(level)))) {
	case LogLevelDebug:
		lvl = slog.LevelDebug
	case LogLevelInfo:
		lvl = slog.LevelInfo
	case LogLevelWarn:
		lvl = slog.LevelWarn
	case LogLevelError:
		lvl = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level %q: want debug, info, warn or error", level)
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		return nil, fmt.Errorf("invalid log format %q: want text or json", format)
	}

	logger := slog.New(handler)

	return &SlogLogger{logger: logger}, nil
}

func (l *SlogLogger) Info(msg string, fields map[string]any) {
	l.log(slog.LevelInfo, msg, fields)
}

func (l *SlogLogger) Debug(msg string, fields map[string]any) {
	l.log(slog.LevelDebug, msg, fields)
}

func (l *SlogLogger) Warn(msg string, fields map[string]any) {
	l.log(slog.LevelWarn, msg, fields)
}

func (l *SlogLogger) Error(msg string, fields map[string]any) {
	l.log(slog.LevelError, msg, fields)
}

func toAttrs(fields map[string]any) []slog.Attr {
	slogFields := make([]slog.Attr, 0, len(fields))

	keys := slices.Sorted(maps.Keys(fields))
	for _, key := range keys {
		slogFields = append(slogFields, slog.Any(key, fields[key]))
	}

	return slogFields
}

func (l *SlogLogger) log(level slog.Level, msg string, fields map[string]any) {
	l.logger.LogAttrs(context.Background(), level, msg, toAttrs(fields)...)
}
