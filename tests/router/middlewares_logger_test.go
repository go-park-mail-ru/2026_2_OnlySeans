package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type logEntry struct {
	level  string
	msg    string
	fields map[string]any
}

// spyLogger запоминает всё, что в него писали (реализует logger.Logger).
type spyLogger struct {
	*mocks.MockLogger
	mu      sync.Mutex
	entries []logEntry
}

func (s *spyLogger) add(level, msg string, fields map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, logEntry{level: level, msg: msg, fields: fields})
}

func TestWithCORS(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		origin          string
		wantStatus      int // 204 на OPTIONS значит, что ответил мидлвар: mux без него вернул бы 405
		wantAllowOrigin string
	}{
		{"preflight отвечает middleware", http.MethodOptions, testOrigin, http.StatusNoContent, testOrigin},
		{"preflight с чужого origin", http.MethodOptions, "http://evil.example", http.StatusNoContent, ""},
		{"обычный запрос доходит до хендлера", http.MethodGet, testOrigin, http.StatusOK, testOrigin},
		{"чужой origin не получает разрешения", http.MethodGet, "http://evil.example", http.StatusOK, ""},
		{"запрос без origin", http.MethodGet, "", http.StatusOK, ""},
	}

	handler := newTestRouter(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/films", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantAllowOrigin, rec.Header().Get("Access-Control-Allow-Origin"))

			if tt.wantAllowOrigin != "" {
				assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
				assert.Equal(t, "Origin", rec.Header().Get("Vary"))
			} else {
				assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"))
			}

			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
			assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		})
	}
}

func TestWithLogging(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		wantPath   string
		wantStatus int
		wantLevel  string
	}{
		{"успешный запрос", http.MethodGet, "/api/films?limit=5", "/api/films", http.StatusOK, "info"},
		{"невалидный id", http.MethodGet, "/api/films/abc", "/api/films/abc", http.StatusBadRequest, "warn"},
		{"неизвестный путь", http.MethodGet, "/nope", "/nope", http.StatusNotFound, "warn"},
		{"метод не разрешён", http.MethodGet, "/api/register", "/api/register", http.StatusMethodNotAllowed, "warn"},
		{"нет сессии", http.MethodPost, "/api/logout", "/api/logout", http.StatusUnauthorized, "warn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := newSpyLogger(t)
			handler := newTestRouterWithLogger(t, spy)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var requestEntries []logEntry

			for _, entry := range spy.entries {
				if entry.msg == "request handled" {
					requestEntries = append(requestEntries, entry)
				}
			}

			require.Len(t, requestEntries, 1)
			entry := requestEntries[0]
			assert.Equal(t, tt.wantLevel, entry.level)
			assert.Equal(t, "request handled", entry.msg)
			assert.Equal(t, tt.method, entry.fields["method"])
			assert.Equal(t, tt.wantPath, entry.fields["path"], "query не должен попадать в path")
			assert.Equal(t, tt.wantStatus, entry.fields["status"])
			assert.Contains(t, entry.fields, "duration_ms")
		})
	}
}

func newSpyLogger(t *testing.T) *spyLogger {
	t.Helper()
	result := &spyLogger{MockLogger: mocks.NewMockLogger(gomock.NewController(t))}
	result.EXPECT().Info(gomock.Any(), gomock.Any()).Do(func(msg string, fields map[string]any) { result.add("info", msg, fields) }).AnyTimes()
	result.EXPECT().InfoContext(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string, fields map[string]any) { result.add("info", msg, fields) }).AnyTimes()
	result.EXPECT().Debug(gomock.Any(), gomock.Any()).Do(func(msg string, fields map[string]any) { result.add("debug", msg, fields) }).AnyTimes()
	result.EXPECT().DebugContext(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string, fields map[string]any) { result.add("debug", msg, fields) }).AnyTimes()
	result.EXPECT().Warn(gomock.Any(), gomock.Any()).Do(func(msg string, fields map[string]any) { result.add("warn", msg, fields) }).AnyTimes()
	result.EXPECT().WarnContext(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string, fields map[string]any) { result.add("warn", msg, fields) }).AnyTimes()
	result.EXPECT().Error(gomock.Any(), gomock.Any()).Do(func(msg string, fields map[string]any) { result.add("error", msg, fields) }).AnyTimes()
	result.EXPECT().ErrorContext(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, msg string, fields map[string]any) { result.add("error", msg, fields) }).AnyTimes()
	return result
}
