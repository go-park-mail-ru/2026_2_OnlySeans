package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string) (domain.UserID, error)
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

	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
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
func withCORS(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(authn Authenticator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
			return
		}

		userID, err := authn.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
				return
			}
			writeError(w, http.StatusInternalServerError, domain.ErrInternal)
			return
		}

		next(w, r.WithContext(auth.ContextWithUserID(r.Context(), userID)))
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	httpresponse.WriteError(w, status, err.Error(), nil)
}

func withLogging(next http.Handler, appLogger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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
			appLogger.Error("request handled", appLogMap)
		case wrapped.status >= 400:
			appLogger.Warn("request handled", appLogMap)
		default:
			appLogger.Info("request handled", appLogMap)
		}
	})
}

func withRecover(next http.Handler, appLogger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracked, ok := w.(*responseWriter)
		if !ok {
			tracked = &responseWriter{ResponseWriter: w, status: http.StatusOK}
		}
		defer func() {
			if rec := recover(); rec != nil {
				appLogger.Error("panic recovered", map[string]any{
					"panic":  fmt.Sprint(rec),
					"method": r.Method,
					"path":   r.URL.Path,
				})
				if !tracked.wroteHeader {
					httpresponse.WriteError(tracked, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
				}
			}
		}()
		next.ServeHTTP(tracked, r)
	})
}

// CatalogEditor checks the current account role after session authentication.
type CatalogEditor interface {
	CanEditPersons(context.Context, domain.UserID) (bool, error)
}

func RequireCatalogEditor(checker CatalogEditor, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
			return
		}
		allowed, err := checker.CanEditPersons(r.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
				return
			}
			writeError(w, http.StatusInternalServerError, domain.ErrInternal)
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, domain.ErrForbidden)
			return
		}
		next(w, r)
	}
}
