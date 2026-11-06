package auth

import (
	"context"
	"errors"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"net/http"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const SessionCookieName = "session_id"

type Handler struct {
	register     *usecase.RegisterUserUseCase
	login        *usecase.LoginUserUseCase
	logout       *usecase.LogoutUserUseCase
	authenticate *usecase.AuthenticateUserUseCase
	getUser      *usecase.GetUserUseCase
	appLogger    logger.Logger
	CookieSecure bool
}

type HandlerUseCases struct {
	Register     *usecase.RegisterUserUseCase
	Login        *usecase.LoginUserUseCase
	Logout       *usecase.LogoutUserUseCase
	Authenticate *usecase.AuthenticateUserUseCase
	GetUser      *usecase.GetUserUseCase
}

func NewHandler(cases HandlerUseCases, appLogger logger.Logger) (*Handler, error) {
	if cases.Register == nil {
		return nil, errors.New("register use case is nil")
	}
	if cases.Login == nil {
		return nil, errors.New("login use case is nil")
	}
	if cases.Logout == nil {
		return nil, errors.New("logout use case is nil")
	}
	if cases.Authenticate == nil {
		return nil, errors.New("authenticate use case is nil")
	}
	if cases.GetUser == nil {
		return nil, errors.New("get user use case is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger use case is nil")
	}

	return &Handler{
		register:     cases.Register,
		login:        cases.Login,
		logout:       cases.Logout,
		authenticate: cases.Authenticate,
		getUser:      cases.GetUser,
		appLogger:    appLogger,
	}, nil
}

// Register handles POST /api/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	var req registerRequest
	if err := httprequest.DecodeJSON(w, r, &req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body", h.appLogger)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	result, err := h.register.Run(r.Context(), toRegisterUserInput(req))
	if err != nil {
		writeRegisterError(w, err, h.appLogger)
		return
	}

	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	httpresponse.WriteJSON(w, http.StatusCreated, toAuthResponse(result.User), h.appLogger)
}

func writeRegisterError(w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidPassword):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, domain.ErrUserExists):
		httpresponse.WriteError(w, http.StatusConflict, domain.ErrUserExists.Error(), appLogger)
	case errors.Is(err, domain.ErrUsernameTaken):
		httpresponse.WriteError(w, http.StatusConflict, domain.ErrUsernameTaken.Error(), appLogger)
	default:
		appLogger.Error("register user failed", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), appLogger)
	}
}

// Login handles POST /api/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	var req loginRequest
	if err := httprequest.DecodeJSON(w, r, &req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body", h.appLogger)
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	result, err := h.login.Run(r.Context(), toLoginUserInput(req))
	if err != nil {
		writeLoginError(w, err, r.RemoteAddr, h.appLogger)
		return
	}

	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	httpresponse.WriteJSON(w, http.StatusOK, toAuthResponse(result.User), h.appLogger)
}

func writeLoginError(w http.ResponseWriter, err error, remoteAddr string, appLogger logger.Logger) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, domain.ErrInvalidCredentials):
		appLogger.Warn("login rejected", map[string]any{"remote_addr": remoteAddr})
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrInvalidCredentials.Error(), appLogger)
	default:
		appLogger.Error("login user failed", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), appLogger)
	}
}

// Logout handles POST /api/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.logout.Run(r.Context(), usecase.NewLogoutInput(cookie.Value)); err != nil {
			h.appLogger.Error("logout user failed", map[string]any{"error": err})
			httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Authenticate(ctx context.Context, sessionID string) (domain.UserID, error) {
	userID, err := h.authenticate.Run(ctx, usecase.NewAuthenticateInput(sessionID))
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			return 0, err
		}

		h.appLogger.Error("authenticate session failed", map[string]any{"error": err})

		return 0, domain.ErrInternal
	}

	return userID.ID, nil
}

func (h *Handler) getUserRole(ctx context.Context, userID domain.UserID) (domain.Role, error) {
	user, err := h.getUser.Run(ctx, usecase.NewGetUserInput(userID))
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", domain.ErrUserNotFound
	}
	return user.Role, nil
}

func (h *Handler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
			return
		}

		role, err := h.getUserRole(r.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
				return
			}
			h.appLogger.Error("get user role failed", map[string]any{"error": err})
			httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
			return
		}
		if role != domain.RoleAdmin {
			httpresponse.WriteError(w, http.StatusForbidden, domain.ErrForbidden.Error(), h.appLogger)
			return
		}

		next(w, r)
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, sessionID string, expiresAt time.Time) {
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

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
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

func (h *Handler) Authorised(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	user, err := h.getUser.Run(r.Context(), usecase.NewGetUserInput(id))
	if err != nil {
		h.appLogger.Error("get authorised user failed", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
		return
	}
	if user == nil {
		h.appLogger.Error("get authorised user returned nil user", map[string]any{"user_id": id})
		httpresponse.WriteError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, toAuthResponse(usecase.NewUserDTO(user.ID, user.Email, user.Username)), h.appLogger)
}

func (h *Handler) CanEditPersons(ctx context.Context, userID domain.UserID) (bool, error) {
	user, err := h.getUser.Run(ctx, usecase.NewGetUserInput(userID))
	if err != nil {
		return false, err
	}
	return user.Role.CanEditPersons(), nil
}
