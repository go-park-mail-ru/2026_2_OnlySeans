package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string) (domain.UserID, error)
}

var (
	ErrCrossOrigin = errors.New("cross-origin request blocked")
)

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

// добавляет ко всем ответам заголовки, запрещающие браузеру исполнять или встраивать ответы
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")                                     // запрет браузеру угадывать тип ответа
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'") // что документу разрешено. ничего не исполняет и не встраивается во фреймы
		w.Header().Set("X-Frame-Options", "DENY")                                               // запрет встраивать ответы во фреймы
		next.ServeHTTP(w, r)
	})
}

// отклоняет POST, PUT и DELETE с чужих сайтов
func withCSRF(next http.Handler, allowedOrigin string) (http.Handler, error) {
	protection := http.NewCrossOriginProtection()

	err := protection.AddTrustedOrigin(allowedOrigin) // добавляем фронтенд в доверенные
	if err != nil {
		return nil, fmt.Errorf("add trusted origin %q: %w", allowedOrigin, err)
	}

	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusForbidden, ErrCrossOrigin)
	}))

	return protection.Handler(next), nil
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
