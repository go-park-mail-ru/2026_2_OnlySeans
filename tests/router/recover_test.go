package router_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recoverLogEntry struct {
	level  string
	msg    string
	fields map[string]any
}

type recoverSpyLogger struct {
	mu      sync.Mutex
	entries []recoverLogEntry
}

func (s *recoverSpyLogger) add(level, msg string, f map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, recoverLogEntry{level, msg, f})
}

func (s *recoverSpyLogger) Info(msg string, f map[string]any)  { s.add("info", msg, f) }
func (s *recoverSpyLogger) Debug(msg string, f map[string]any) { s.add("debug", msg, f) }
func (s *recoverSpyLogger) Warn(msg string, f map[string]any)  { s.add("warn", msg, f) }
func (s *recoverSpyLogger) Error(msg string, f map[string]any) { s.add("error", msg, f) }

func TestWithRecover(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantLogged bool
		wantPanic  string // что должно попасть в поле "panic" лога
	}{
		{
			"паника строкой",
			func(http.ResponseWriter, *http.Request) { panic("boom") },
			http.StatusInternalServerError, true, "boom",
		},
		{
			"паника ошибкой",
			func(http.ResponseWriter, *http.Request) { panic(errors.New("db down")) },
			http.StatusInternalServerError, true, "db down",
		},
		{
			"паника числом",
			func(http.ResponseWriter, *http.Request) { panic(42) },
			http.StatusInternalServerError, true, "42",
		},
		{
			"без паники запрос проходит как есть",
			func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) },
			http.StatusTeapot, false, "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &recoverSpyLogger{}
			handler := router.WithRecover(tt.handler, spy)

			req := httptest.NewRequest(http.MethodGet, "/api/films", nil)
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() { handler.ServeHTTP(rec, req) })

			assert.Equal(t, tt.wantStatus, rec.Code)

			if !tt.wantLogged {
				assert.Empty(t, spy.entries)
				return
			}

			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, domain.ErrInternal.Error(), body["error"], "детали паники не должны утекать клиенту")
			assert.NotContains(t, rec.Body.String(), tt.wantPanic)

			require.Len(t, spy.entries, 1)
			entry := spy.entries[0]
			assert.Equal(t, "error", entry.level)
			assert.Equal(t, "panic recovered", entry.msg)
			assert.Equal(t, tt.wantPanic, entry.fields["panic"])
			assert.Equal(t, http.MethodGet, entry.fields["method"])
			assert.Equal(t, "/api/films", entry.fields["path"])
		})
	}
}

// Порядок как в NewRouter: logging снаружи, recover внутри.
func TestWithLogging_WithRecover(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantMsgs   []string
		wantLevel  string // уровень последней записи ("request handled")
	}{
		{
			"после паники логируется 500",
			func(http.ResponseWriter, *http.Request) { panic("boom") },
			http.StatusInternalServerError,
			[]string{"request started", "panic recovered", "request handled"},
			"error",
		},
		{
			"без паники логируется начало и завершение запроса",
			func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) },
			http.StatusCreated,
			[]string{"request started", "request handled"},
			"info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &recoverSpyLogger{}
			handler := router.WithLogging(router.WithRecover(tt.handler, spy), spy)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

			assert.Equal(t, tt.wantStatus, rec.Code)

			var gotMsgs []string
			for _, e := range spy.entries {
				gotMsgs = append(gotMsgs, e.msg)
			}
			assert.Equal(t, tt.wantMsgs, gotMsgs)

			last := spy.entries[len(spy.entries)-1]
			assert.Equal(t, tt.wantStatus, last.fields["status"])
			assert.Equal(t, tt.wantLevel, last.level)
		})
	}
}

// Границы выбора уровня в router.WithLogging.
func TestWithLogging_Level(t *testing.T) {
	tests := []struct {
		status    int
		wantLevel string
	}{
		{http.StatusOK, "info"},
		{http.StatusCreated, "info"},
		{http.StatusNoContent, "info"},
		{http.StatusFound, "info"},
		{399, "info"},
		{400, "warn"},
		{http.StatusNotFound, "warn"},
		{499, "warn"},
		{500, "error"},
		{http.StatusServiceUnavailable, "error"},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.status), func(t *testing.T) {
			spy := &recoverSpyLogger{}
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tt.status) })
			handler := router.WithLogging(next, spy)

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			require.Len(t, spy.entries, 2)
			assert.Equal(t, "debug", spy.entries[0].level)
			assert.Equal(t, "request started", spy.entries[0].msg)
			assert.Equal(t, tt.wantLevel, spy.entries[1].level)
			assert.Equal(t, "request handled", spy.entries[1].msg)
			assert.Equal(t, tt.status, spy.entries[1].fields["status"])
		})
	}
}

func TestRecoveryAfterResponseStarted(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name: "implicit status before panic",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("partial"))
				panic("boom")
			},
			wantStatus: http.StatusOK,
			wantBody:   "partial",
		},
		{
			name: "explicit status before panic",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				panic("boom")
			},
			wantStatus: http.StatusAccepted,
			wantBody:   "",
		},
		{
			name: "later status cannot replace implicit status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ok"))
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spy := &recoverSpyLogger{}
			handler := router.WithLogging(router.WithRecover(tt.handler, spy), spy)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantBody, rec.Body.String())
			last := spy.entries[len(spy.entries)-1]
			require.Equal(t, tt.wantStatus, last.fields["status"])
			if tt.name != "later status cannot replace implicit status" {
				require.Equal(t, "panic recovered", spy.entries[1].msg)
			}
		})
	}
}
