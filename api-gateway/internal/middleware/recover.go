package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

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
