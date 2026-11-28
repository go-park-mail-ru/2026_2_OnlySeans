package onlineconf

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	authorizationHeader    = "Authorization"
	bearerScheme           = "Bearer"
	authenticateHeader     = "WWW-Authenticate"
	cacheControlHeader     = "Cache-Control"
	privateNoStore         = "private, no-store"
	unauthorizedCode       = "unauthorized"
	forbiddenCode          = "forbidden"
	internalErrorCode      = "internal_error"
	authorizationPartCount = 2
	singleHeaderCount      = 1
)

type AccessChecker interface {
	AuthenticateAdmin(context.Context, string) (oc.AdminIdentity, error)
}

type Middleware struct {
	access           AccessChecker
	serviceTokenHash [sha256.Size]byte
	csrf             *http.CrossOriginProtection
	appLogger        logger.Logger
}

type adminContextKey struct{}

type accessErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func NewMiddleware(access AccessChecker, serviceToken string, allowedOrigins []string, appLogger logger.Logger) (*Middleware, error) {
	if access == nil {
		return nil, oc.ErrNilAccessChecker
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	if serviceToken == "" || strings.IndexFunc(serviceToken, unicode.IsSpace) >= 0 {
		return nil, oc.ErrInvalidServiceToken
	}

	protection := http.NewCrossOriginProtection()
	for _, origin := range allowedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("add onlineconf trusted origin: %w", err)
		}
	}

	m := &Middleware{access: access, serviceTokenHash: sha256.Sum256([]byte(serviceToken)), csrf: protection, appLogger: appLogger}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { m.writeAccessError(w, r, http.StatusForbidden) }))

	return m, nil
}

func AdminFromContext(ctx context.Context) (oc.AdminIdentity, bool) {
	admin, ok := ctx.Value(adminContextKey{}).(oc.AdminIdentity)
	return admin, ok && admin.ID >= oc.MinUserID && int64(admin.ID) <= oc.MaxSafeID && admin.Role == domain.RoleAdmin
}

func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	protected := m.csrf.Handler(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(cacheControlHeader, privateNoStore)

		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil || cookie.Value == "" {
			m.writeAccessError(w, r, http.StatusUnauthorized)
			return
		}

		admin, err := m.access.AuthenticateAdmin(r.Context(), cookie.Value)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUnauthorized):
				m.writeAccessError(w, r, http.StatusUnauthorized)
			case errors.Is(err, domain.ErrForbidden):
				m.writeAccessError(w, r, http.StatusForbidden)
			default:
				m.appLogger.ErrorContext(r.Context(), "onlineconf admin authentication failed", map[string]any{"method": r.Method, "path": r.URL.Path})
				m.writeAccessError(w, r, http.StatusInternalServerError)
			}

			return
		}

		if admin.ID < oc.MinUserID || int64(admin.ID) > oc.MaxSafeID || admin.Role != domain.RoleAdmin {
			m.writeAccessError(w, r, http.StatusForbidden)
			return
		}

		ctx := auth.ContextWithUserID(r.Context(), admin.ID)
		ctx = context.WithValue(ctx, adminContextKey{}, admin)
		protected.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) RequireService(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(cacheControlHeader, privateNoStore)

		headers := r.Header.Values(authorizationHeader)
		if len(headers) != singleHeaderCount {
			m.rejectService(w, r)
			return
		}

		parts := strings.Fields(headers[0])
		if len(parts) != authorizationPartCount || !strings.EqualFold(parts[0], bearerScheme) {
			m.rejectService(w, r)
			return
		}

		suppliedHash := sha256.Sum256([]byte(parts[1]))
		if subtle.ConstantTimeCompare(suppliedHash[:], m.serviceTokenHash[:]) != 1 {
			m.rejectService(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) rejectService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(authenticateHeader, bearerScheme)
	m.writeAccessError(w, r, http.StatusUnauthorized)
}

func (m *Middleware) writeAccessError(w http.ResponseWriter, r *http.Request, status int) {
	response := accessErrorResponse{Error: domain.ErrInternal.Error(), Code: internalErrorCode}

	switch status {
	case http.StatusUnauthorized:
		response = accessErrorResponse{Error: domain.ErrUnauthorized.Error(), Code: unauthorizedCode}
	case http.StatusForbidden:
		response = accessErrorResponse{Error: domain.ErrForbidden.Error(), Code: forbiddenCode}
	}

	httpresponse.WriteJSONContext(r.Context(), w, status, response, m.appLogger)
}
