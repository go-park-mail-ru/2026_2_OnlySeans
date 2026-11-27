package logger_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		level   logger.LogLevelType
		format  string
		wantErr string
	}{
		{
			name:    "debug/text",
			level:   logger.LogLevelDebug,
			format:  "text",
			wantErr: "",
		},
		{
			name:    "info/json",
			level:   logger.LogLevelInfo,
			format:  "json",
			wantErr: "",
		},
		{
			name:    "warn",
			level:   logger.LogLevelWarn,
			format:  "text",
			wantErr: "",
		},
		{
			name:    "error",
			level:   logger.LogLevelError,
			format:  "json",
			wantErr: "",
		},
		{
			name:    "регистр и пробелы игнорируются",
			level:   "  INFO ",
			format:  " JSON ",
			wantErr: "",
		},
		{
			name:    "неизвестный уровень",
			level:   "verbose",
			format:  "text",
			wantErr: "invalid log level",
		},
		{
			name:    "пустой уровень",
			level:   "",
			format:  "text",
			wantErr: "invalid log level",
		},
		{
			name:    "неизвестный формат",
			level:   "info",
			format:  "xml",
			wantErr: "invalid log format",
		},
		{
			name:    "пустой формат",
			level:   "info",
			format:  "",
			wantErr: "invalid log format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := logger.NewLogger(&bytes.Buffer{}, tt.level, tt.format)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, l)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, l)
		})
	}
}

func TestSlogLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		level logger.LogLevelType
		want  map[string]bool
	}{
		{
			level: logger.LogLevelDebug,
			want: map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			level: logger.LogLevelInfo,
			want: map[string]bool{
				"debug": false,
				"info":  true,
				"warn":  true,
				"error": true,
			},
		},
		{
			level: logger.LogLevelWarn,
			want: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  true,
				"error": true,
			},
		},
		{
			level: logger.LogLevelError,
			want: map[string]bool{
				"debug": false,
				"info":  false,
				"warn":  false,
				"error": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			for method, wantLogged := range tt.want {
				var buf bytes.Buffer
				l, err := logger.NewLogger(&buf, tt.level, "text")
				require.NoError(t, err)

				switch method {
				case "debug":
					l.Debug("msg", nil)
				case "info":
					l.Info("msg", nil)
				case "warn":
					l.Warn("msg", nil)
				case "error":
					l.Error("msg", nil)
				}

				assert.Equal(t, wantLogged, buf.Len() > 0, "метод %s при уровне %s", method, tt.level)
			}
		})
	}
}

func TestSlogLogger_TextFormat(t *testing.T) {
	tests := []struct {
		name         string
		msg          string
		fields       map[string]any
		wantContains []string
		wantOrder    []string
	}{
		{
			name: "поля идут по алфавиту",
			msg:  "hello",
			fields: map[string]any{
				"b": 2,
				"a": 1,
				"c": 3,
			},
			wantContains: []string{"level=INFO", "msg=hello"},
			wantOrder:    []string{"a=1", "b=2", "c=3"},
		},
		{
			name:         "nil вместо полей",
			msg:          "hello",
			fields:       nil,
			wantContains: []string{"level=INFO", "msg=hello"},
		},
		{
			name:         "пустые поля",
			msg:          "hello",
			fields:       map[string]any{},
			wantContains: []string{"level=INFO", "msg=hello"},
		},
		{
			name:         "сообщение с пробелом берётся в кавычки",
			msg:          "only message",
			fields:       nil,
			wantContains: []string{`msg="only message"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l, err := logger.NewLogger(&buf, "debug", "text")
			require.NoError(t, err)

			require.NotPanics(t, func() { l.Info(tt.msg, tt.fields) })

			out := buf.String()
			for _, want := range tt.wantContains {
				assert.Contains(t, out, want)
			}

			prev := -1

			for _, want := range tt.wantOrder {
				idx := strings.Index(out, want)
				require.GreaterOrEqual(t, idx, 0, "нет %q в выводе", want)
				assert.Greater(t, idx, prev, "%q стоит не на своём месте", want)
				prev = idx
			}
		})
	}
}

func TestSlogLogger_FieldTypesInJSON(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]any
		want   map[string]any
	}{
		{
			name: "строка",
			fields: map[string]any{
				"path": "/api",
			},
			want: map[string]any{
				"path": "/api",
			},
		},
		{
			name: "число",
			fields: map[string]any{
				"status": 200,
			},
			want: map[string]any{
				"status": float64(200),
			},
		},
		{
			name: "bool",
			fields: map[string]any{
				"ok": true,
			},
			want: map[string]any{
				"ok": true,
			},
		},
		{
			name: "несколько полей разных типов",
			fields: map[string]any{
				"status": 404,
				"ok":     false,
				"path":   "/x",
			},
			want: map[string]any{
				"status": float64(404),
				"ok":     false,
				"path":   "/x",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l, err := logger.NewLogger(&buf, "info", "json")
			require.NoError(t, err)

			l.Info("typed", tt.fields)

			var rec map[string]any
			require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))

			for k, want := range tt.want {
				assert.Equal(t, want, rec[k], "поле %q", k)
			}
		})
	}
}
