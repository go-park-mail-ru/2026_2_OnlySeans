package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	RoleAdmin  = "admin"
	RoleEditor = "editor"
)

type Authenticator interface {
	Authenticate(ctx context.Context, sessionID string) (int64, error)
}

type AuthMiddleware struct {
	authClient authclient.AuthClient
	userClient userclient.UserClient
	appLogger  logger.Logger
}

func NewAuthMiddleware(ac authclient.AuthClient, uc userclient.UserClient, l logger.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: ac,
		userClient: uc,
		appLogger:  l,
	}
}

func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := m.appLogger

		cookie, err := r.Cookie("session_id")
		if err != nil || cookie.Value == "" {
			httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, ErrUnauthorized.Error(), l)
			return
		}

		session, err := m.authClient.CheckSession(ctx, cookie.Value)
		if errors.Is(err, authclient.ErrSessionNotFound) {
			httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, ErrUnauthorized.Error(), l)
			return
		}

		if err != nil {
			httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, ErrInternal.Error(), l)
			return
		}

		ctxWithUser := ContextWithUserID(ctx, session.UserID)
		ctxWithUser = ContextWithRole(ctxWithUser, session.Role)
		next.ServeHTTP(w, r.WithContext(ctxWithUser))
	}
}

func (m *AuthMiddleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := m.appLogger

		userID, ok := UserIDFromContext(ctx)
		if !ok {
			l.WarnContext(ctx, "admin access rejected: missing user context", map[string]any{"method": r.Method, "path": r.URL.Path})
			httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, ErrUnauthorized.Error(), l)
			return
		}

		role, ok := m.getUserRole(w, r, userID)
		if !ok {
			return
		}

		if role != RoleAdmin {
			l.WarnContext(ctx, "admin access forbidden", map[string]any{"user_id": userID, "method": r.Method, "path": r.URL.Path})
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
			l.WarnContext(ctx, "catalog modification rejected: missing user context", map[string]any{"method": r.Method, "path": r.URL.Path})
			writeError(ctx, w, http.StatusUnauthorized, ErrUnauthorized)
			return
		}

		role, ok := m.getUserRole(w, r, userID)
		if !ok {
			return
		}

		if role != RoleAdmin && role != RoleEditor {
			l.WarnContext(ctx, "catalog modification forbidden", map[string]any{"user_id": userID, "method": r.Method, "path": r.URL.Path})
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
		m.appLogger.WarnContext(ctx, "role check rejected: user not found", map[string]any{"user_id": userID, "method": r.Method, "path": r.URL.Path})
		writeError(ctx, w, http.StatusUnauthorized, ErrUnauthorized)
		return "", false
	}
	if err != nil {
		m.appLogger.ErrorContext(ctx, "get user role failed", map[string]any{"error": err, "user_id": userID, "method": r.Method, "path": r.URL.Path})
		writeError(ctx, w, http.StatusInternalServerError, ErrInternal)
		return "", false
	}

	return user.GetRole(), true
}
