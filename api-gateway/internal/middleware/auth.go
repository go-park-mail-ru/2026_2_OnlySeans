package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/sessionpolicy"
)

const (
	RoleAdmin         = "admin"
	RoleEditor        = "editor"
	sessionCookieName = "session_id"
	sessionCookiePath = "/"
)

type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string) (int64, error)
}

type AuthMiddleware struct {
	CookieSecure bool
	authClient   authclient.AuthClient
	userClient   userclient.UserClient
	appLogger    logger.Logger
}

func NewAuthMiddleware(ac authclient.AuthClient, uc userclient.UserClient, l logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: ac,
		userClient: uc,
		appLogger:  l,
	}
}

type sessionCheckKey struct{}

type sessionCheckResult struct {
	session authclient.SessionInfo
	err     error
}

func (m *AuthMiddleware) checkSession(w http.ResponseWriter, r *http.Request) (authclient.SessionInfo, error) {
	if cached, ok := r.Context().Value(sessionCheckKey{}).(sessionCheckResult); ok {
		return cached.session, cached.err
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return authclient.SessionInfo{}, authclient.ErrSessionNotFound
	}

	session, err := m.authClient.CheckSession(r.Context(), cookie.Value)
	if err != nil {
		return authclient.SessionInfo{}, err
	}

	if sessionpolicy.NeedsRefresh(session.LastRefreshedAt, time.Now()) {
		refreshed, refreshErr := m.authClient.RefreshSession(r.Context(), cookie.Value)
		if refreshErr != nil {
			return authclient.SessionInfo{}, refreshErr
		}

		if refreshed == nil || refreshed.GetId() != cookie.Value || refreshed.GetExpiresAt() == nil ||
			!refreshed.GetExpiresAt().AsTime().After(time.Now()) {
			return authclient.SessionInfo{}, authclient.ErrInternal
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    cookie.Value,
			Path:     sessionCookiePath,
			Expires:  refreshed.GetExpiresAt().AsTime(),
			HttpOnly: true,
			Secure:   m.CookieSecure,
			SameSite: http.SameSiteLaxMode,
		})

		session.UserID = refreshed.GetUserId()
		session.Role = refreshed.GetRole()
		session.ExpiresAt = refreshed.GetExpiresAt().AsTime()
		session.LastRefreshedAt = refreshed.GetLastRefreshedAt().AsTime()
	}

	return session, nil
}

func (m *AuthMiddleware) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		switch r.URL.Path {
		case "/api/login", "/api/register", "/api/refresh", "/api/logout":
			next.ServeHTTP(w, r)
			return
		}

		session, err := m.checkSession(w, r)
		ctx := context.WithValue(r.Context(), sessionCheckKey{}, sessionCheckResult{session: session, err: err})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := m.checkSession(w, r)
		if errors.Is(err, authclient.ErrSessionNotFound) {
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, ErrUnauthorized.Error(), m.appLogger)
			return
		}

		if err != nil {
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusInternalServerError, ErrInternal.Error(), m.appLogger)
			return
		}

		ctx := ContextWithUserID(r.Context(), session.UserID)
		ctx = ContextWithRole(ctx, session.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := m.appLogger

		userID, ok := UserIDFromContext(ctx)
		if !ok {
			l.WarnContext(ctx, "admin access rejected: missing user context", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, ErrUnauthorized.Error(), l)

			return
		}

		role, ok := m.getUserRole(w, r, userID)
		if !ok {
			return
		}

		if role != RoleAdmin {
			l.WarnContext(ctx, "admin access forbidden", map[string]any{
				"user_id": userID,
				"method":  r.Method,
				"path":    r.URL.Path,
			})
			httpresponse.WriteErrorContext(ctx, w, http.StatusForbidden, ErrForbidden.Error(), l)

			return
		}

		next(w, r)
	}
}

func writeError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	httpresponse.WriteErrorContext(ctx, w, status, err.Error(), nil)
}

func (m *AuthMiddleware) RequireCatalogEditor(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := m.appLogger

		userID, ok := UserIDFromContext(ctx)
		if !ok {
			l.WarnContext(ctx, "catalog modification rejected: missing user context", map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			})
			writeError(ctx, w, http.StatusUnauthorized, ErrUnauthorized)

			return
		}

		role, ok := m.getUserRole(w, r, userID)
		if !ok {
			return
		}

		if role != RoleAdmin && role != RoleEditor {
			l.WarnContext(ctx, "catalog modification forbidden", map[string]any{
				"user_id": userID,
				"method":  r.Method,
				"path":    r.URL.Path,
			})
			writeError(ctx, w, http.StatusForbidden, ErrForbidden)

			return
		}

		next(w, r)
	}
}

func (m *AuthMiddleware) getUserRole(w http.ResponseWriter, r *http.Request, userID int64) (string, bool) {
	ctx := r.Context()
	user, err := m.userClient.GetByID(ctx, userID)

	if errors.Is(err, userclient.ErrUserNotFound) {
		m.appLogger.WarnContext(ctx, "role check rejected: user not found", map[string]any{
			"user_id": userID,
			"method":  r.Method,
			"path":    r.URL.Path,
		})
		writeError(ctx, w, http.StatusUnauthorized, ErrUnauthorized)

		return "", false
	}

	if err != nil {
		m.appLogger.ErrorContext(ctx, "get user role failed", map[string]any{
			"error":   err,
			"user_id": userID,
			"method":  r.Method,
			"path":    r.URL.Path,
		})
		writeError(ctx, w, http.StatusInternalServerError, ErrInternal)

		return "", false
	}

	return user.GetRole(), true
}
