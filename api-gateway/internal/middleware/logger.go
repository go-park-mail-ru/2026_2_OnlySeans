package middleware

import (
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func WrapMiddlewares(handler http.HandlerFunc, middlewares []Middleware) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}

	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func WithLogging(next http.Handler, appLogger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		r = requestWithID(w, r)
		appLogger.DebugContext(r.Context(), "request started", map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
		})

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		appLogMap := map[string]any{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.status,
			"duration_ms": time.Since(start).Milliseconds(),
		}

		switch {
		case wrapped.status >= 500:
			appLogger.ErrorContext(r.Context(), "request handled", appLogMap)
		case wrapped.status >= 400:
			appLogger.WarnContext(r.Context(), "request handled", appLogMap)
		default:
			appLogger.InfoContext(r.Context(), "request handled", appLogMap)
		}
	})
}
