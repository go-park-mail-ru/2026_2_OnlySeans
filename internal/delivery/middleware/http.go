package middleware

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

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

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(w, r)

		next.ServeHTTP(w, r)
	})
}

func requestWithID(w http.ResponseWriter, r *http.Request) *http.Request {
	id := logger.RequestID(r.Context())
	if id == "" {
		id = rand.Text()
		r = r.WithContext(logger.ContextWithRequestID(r.Context(), id))
	}

	w.Header().Set("X-Request-ID", id)

	return r
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

func WithRecover(next http.Handler, appLogger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracked, ok := w.(*responseWriter)
		if !ok {
			tracked = &responseWriter{ResponseWriter: w, status: http.StatusOK}
		}

		defer func() {
			if rec := recover(); rec != nil {
				appLogger.ErrorContext(r.Context(), "panic recovered", map[string]any{
					"panic":  fmt.Sprint(rec),
					"method": r.Method,
					"path":   r.URL.Path,
				})

				if !tracked.wroteHeader {
					httpresponse.WriteErrorContext(r.Context(), tracked, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
				}
			}
		}()

		next.ServeHTTP(tracked, r)
	})
}
