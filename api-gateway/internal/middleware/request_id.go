package middleware

import (
	"crypto/rand"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

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
