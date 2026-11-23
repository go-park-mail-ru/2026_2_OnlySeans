package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	SessionCookieName = "session_id"
	roleUser          = "reader"
)

type AuthHandler struct {
	authClient   authclient.AuthClient
	userClient   userclient.UserClient
	appLogger    logger.Logger
	CookieSecure bool
}

func NewAuthHandler(ac authclient.AuthClient, uc userclient.UserClient, appLogger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authClient: ac,
		userClient: uc,
		appLogger:  appLogger,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := h.appLogger

	if r.Method != http.MethodPost {
		httpresponse.WriteErrorContext(ctx, w, http.StatusMethodNotAllowed, "method not supported", l)
		return
	}

	var req registerRequest
	if err := httprequest.DecodeJSON(w, r, &req); err != nil {
		httprequest.LogRejected(l, r, "body", err)
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, "invalid request body", l)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	userID, err := h.userClient.CreateUser(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		writeRegisterError(ctx, w, err, l)
		return
	}

	user, err := h.userClient.GetByID(ctx, userID)
	if err != nil {
		writeRegisterError(ctx, w, err, l)
		return
	}

	session, err := h.authClient.IssueSession(ctx, userID, roleUser)
	if err != nil {
		writeRegisterError(ctx, w, err, l)
		return
	}

	h.setSessionCookie(w, session.GetId(), session.GetExpiresAt().AsTime())
	httpresponse.WriteJSONContext(ctx, w, http.StatusCreated, toAuthResponse(user), l)
}

func writeRegisterError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, userclient.ErrInvalidEmail),
		errors.Is(err, userclient.ErrInvalidUsername),
		errors.Is(err, userclient.ErrInvalidPassword):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, userclient.ErrUserExists):
		httpresponse.WriteErrorContext(ctx, w, http.StatusConflict, userclient.ErrUserExists.Error(), appLogger)
	case errors.Is(err, userclient.ErrUsernameTaken):
		httpresponse.WriteErrorContext(ctx, w, http.StatusConflict, userclient.ErrUsernameTaken.Error(), appLogger)
	default:

		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, middleware.ErrInternal.Error(), appLogger)
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := h.appLogger

	if r.Method != http.MethodPost {
		httpresponse.WriteErrorContext(ctx, w, http.StatusMethodNotAllowed, "method not supported", l)
		return
	}

	var req loginRequest
	if err := httprequest.DecodeJSON(w, r, &req); err != nil {
		httprequest.LogRejected(l, r, "body", err)
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, "invalid request body", l)
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	session, err := h.authClient.Login(ctx, req.Email, req.Password)
	if err != nil {
		writeLoginError(ctx, w, err, l)
		return
	}

	user, err := h.userClient.GetByID(ctx, session.GetUserId())
	if err != nil {
		writeLoginError(ctx, w, err, l)
		return
	}

	h.setSessionCookie(w, session.GetId(), session.GetExpiresAt().AsTime())
	httpresponse.WriteJSONContext(ctx, w, http.StatusOK, toAuthResponse(user), l)
}

func writeLoginError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, userclient.ErrInvalidEmail),
		errors.Is(err, userclient.ErrInvalidPassword):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, userclient.ErrInvalidCredentials):
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, userclient.ErrInvalidCredentials.Error(), appLogger)
	default:

		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, middleware.ErrInternal.Error(), appLogger)
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := h.appLogger

	if r.Method != http.MethodPost {
		httpresponse.WriteErrorContext(ctx, w, http.StatusMethodNotAllowed, "method not supported", l)
		return
	}

	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.authClient.Logout(ctx, cookie.Value); err != nil {
			httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, middleware.ErrInternal.Error(), l)
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, sessionID string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *AuthHandler) Authorised(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := h.appLogger

	id, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, middleware.ErrUnauthorized.Error(), l)
		return
	}

	user, err := h.userClient.GetByID(ctx, id)
	if err != nil {
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, middleware.ErrInternal.Error(), l)
		return
	}

	httpresponse.WriteJSONContext(ctx, w, http.StatusOK, toAuthResponse(user), l)
}

func (h *AuthHandler) CanEditPersons(ctx context.Context, userID int64) (bool, error) {
	user, err := h.userClient.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	role := user.GetRole()
	return role == middleware.RoleAdmin || role == middleware.RoleEditor, nil
}
