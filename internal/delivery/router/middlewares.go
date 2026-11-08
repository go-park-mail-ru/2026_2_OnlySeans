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
func withCORS(next http.Handler, allowedOrigins []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				break
			}
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

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(authn Authenticator) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
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
}

type csrfMiddleware struct {
	protection *http.CrossOriginProtection
}

func newCSRFMiddleware(allowedOrigins []string) (*csrfMiddleware, error) {
	protection := http.NewCrossOriginProtection()
	for _, origin := range allowedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("add trusted origin %q: %w", origin, err)
		}
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, domain.ErrForbidden)
	}))
	return &csrfMiddleware{protection: protection}, nil
}

func (m *csrfMiddleware) withCSRF(next http.Handler) http.Handler {
	return m.protection.Handler(next)
}

func writeError(w http.ResponseWriter, status int, err error) {
	httpresponse.WriteError(w, status, err.Error(), nil)
}

func WithLogging(next http.Handler, appLogger logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		appLogger.Debug("request started", map[string]any{
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
			appLogger.Error("request handled", appLogMap)
		case wrapped.status >= 400:
			appLogger.Warn("request handled", appLogMap)
		default:
			appLogger.Info("request handled", appLogMap)
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

func RequireCatalogEditor(checker CatalogEditor, appLogger logger.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok {
				appLogger.Warn(
					"catalog modification rejected: missing user context",
					map[string]any{
						"method": r.Method,
						"path":   r.URL.Path,
					},
				)
				writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
				return
			}

			allowed, err := checker.CanEditPersons(r.Context(), userID)
			if err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					appLogger.Warn(
						"catalog modification rejected: user not found",
						map[string]any{
							"user_id": userID,
							"method":  r.Method,
							"path":    r.URL.Path,
						},
					)
					writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized)
					return
				}

				appLogger.Error("catalog permission check failed", map[string]any{"error": err, "user_id": userID, "method": r.Method, "path": r.URL.Path})
				writeError(w, http.StatusInternalServerError, domain.ErrInternal)
				return
			}

			if !allowed {
				appLogger.Warn(
					"catalog modification forbidden",
					map[string]any{
						"user_id": userID,
						"method":  r.Method,
						"path":    r.URL.Path,
					},
				)
				writeError(w, http.StatusForbidden, domain.ErrForbidden)
				return
			}

			next(w, r)
		}
	}
}
