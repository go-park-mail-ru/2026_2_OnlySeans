package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	commonmiddleware "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/middleware"
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
				writeError(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized)
				return
			}

			userID, err := authn.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, domain.ErrUnauthorized) {
					writeError(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized)
					return
				}

				writeError(r.Context(), w, http.StatusInternalServerError, domain.ErrInternal)
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
		writeError(r.Context(), w, http.StatusForbidden, domain.ErrForbidden)
	}))

	return &csrfMiddleware{protection: protection}, nil
}

func (m *csrfMiddleware) withCSRF(next http.Handler) http.Handler {
	return m.protection.Handler(next)
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	httpresponse.WriteErrorContext(ctx, w, status, err.Error(), nil)
}

func WithRequestID(next http.Handler) http.Handler {
	return commonmiddleware.WithRequestID(next)
}

func WithLogging(next http.Handler, appLogger logger.Logger) http.Handler {
	return commonmiddleware.WithLogging(next, appLogger)
}

func WithRecover(next http.Handler, appLogger logger.Logger) http.Handler {
	return commonmiddleware.WithRecover(next, appLogger)
}

type CatalogEditor interface {
	CanEditPersons(context.Context, domain.UserID) (bool, error)
}

func RequireCatalogEditor(checker CatalogEditor, appLogger logger.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userID, ok := auth.UserIDFromContext(r.Context())
			if !ok {
				appLogger.WarnContext(r.Context(),
					"catalog modification rejected: missing user context",
					map[string]any{
						"method": r.Method,
						"path":   r.URL.Path,
					},
				)

				writeError(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized)
				return
			}

			allowed, err := checker.CanEditPersons(r.Context(), userID)
			if err != nil {
				if errors.Is(err, domain.ErrUserNotFound) {
					appLogger.WarnContext(r.Context(),
						"catalog modification rejected: user not found",
						map[string]any{
							"user_id": userID,
							"method":  r.Method,
							"path":    r.URL.Path,
						},
					)

					writeError(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized)
					return
				}

				appLogger.ErrorContext(r.Context(), "catalog permission check failed", map[string]any{"error": err, "user_id": userID, "method": r.Method, "path": r.URL.Path})
				writeError(r.Context(), w, http.StatusInternalServerError, domain.ErrInternal)
				return
			}

			if !allowed {
				appLogger.WarnContext(r.Context(),
					"catalog modification forbidden",
					map[string]any{
						"user_id": userID,
						"method":  r.Method,
						"path":    r.URL.Path,
					},
				)

				writeError(r.Context(), w, http.StatusForbidden, domain.ErrForbidden)
				return
			}

			next(w, r)
		}
	}
}
