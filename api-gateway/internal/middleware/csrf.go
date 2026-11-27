package middleware

import (
	"fmt"
	"net/http"
)

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
		writeError(r.Context(), w, http.StatusForbidden, ErrForbidden)
	}))

	return &csrfMiddleware{protection: protection}, nil
}

func (m *csrfMiddleware) withCSRF(next http.Handler) http.Handler {
	return m.protection.Handler(next)
}

func WithCORSAndCSRF(next http.Handler, allowedOrigins []string) (http.Handler, error) {
	csrf, err := newCSRFMiddleware(allowedOrigins)
	if err != nil {
		return nil, err
	}

	return withCORS(csrf.withCSRF(next), allowedOrigins), nil
}
