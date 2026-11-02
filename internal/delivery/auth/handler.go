package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	AppLogger    logger.Logger
}

func NewHandler(cases HandlerUseCases) (*Handler, error) {
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

	if cases.AppLogger == nil {
		return nil, errors.New("logger use case is nil")
	}

	return &Handler{
		register:     cases.Register,
		login:        cases.Login,
		logout:       cases.Logout,
		authenticate: cases.Authenticate,
		getUser:      cases.GetUser,
		appLogger:    cases.AppLogger,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, body any, appLogger logger.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		appLogger.Error("write response failed", map[string]any{"error": err})
	}
}

func writeError(w http.ResponseWriter, status int, msg string, appLogger logger.Logger) {
	writeJSON(w, status, errorResponse{Error: msg}, appLogger)
}

// Register handles POST /api/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", h.appLogger)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Username = strings.TrimSpace(req.Username)

	result, err := h.register.Run(r.Context(), toRegisterUserInput(req))
	if err != nil {
		writeRegisterError(w, err, h.appLogger)
		return
	}

	h.appLogger.Info("user registered", map[string]any{"user_id": result.User.ID})
	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	writeJSON(w, http.StatusCreated, toAuthResponse(result.User), h.appLogger)
}

func writeRegisterError(w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, domain.ErrUserExists):
		writeError(w, http.StatusConflict, domain.ErrUserExists.Error(), appLogger)
	case errors.Is(err, domain.ErrUsernameTaken):
		writeError(w, http.StatusConflict, domain.ErrUsernameTaken.Error(), appLogger)
	default:
		appLogger.Error("register user failed", map[string]any{"error": err})
		writeError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), appLogger)
	}
}

// Login handles POST /api/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", h.appLogger)
		return
	}
	req.Email = strings.TrimSpace(req.Email)

	result, err := h.login.Run(r.Context(), toLoginUserInput(req))
	if err != nil {
		writeLoginError(w, err, r.RemoteAddr, h.appLogger)
		return
	}

	h.appLogger.Info("user logged in", map[string]any{"user_id": result.User.ID})
	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	writeJSON(w, http.StatusOK, toAuthResponse(result.User), h.appLogger)
}

func writeLoginError(w http.ResponseWriter, err error, remoteAddr string, appLogger logger.Logger) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, domain.ErrInvalidCredentials):
		appLogger.Warn("login rejected", map[string]any{"remote_addr": remoteAddr})
		writeError(w, http.StatusUnauthorized, domain.ErrInvalidCredentials.Error(), appLogger)
	default:
		appLogger.Error("login user failed", map[string]any{"error": err})
		writeError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), appLogger)
	}
}

// Logout handles POST /api/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported", h.appLogger)
		return
	}

	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.logout.Run(r.Context(), usecase.NewLogoutInput(cookie.Value)); err != nil {
			h.appLogger.Error("logout user failed", map[string]any{"error": err})
			writeError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
			return
		}
	}

	h.appLogger.Info("user loggedd out", map[string]any{})
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
		writeError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	user, err := h.getUser.Run(r.Context(), usecase.NewGetUserInput(id))
	if err != nil {
		h.appLogger.Error("get authorised user failed", map[string]any{"error": err})
		writeError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
		return
	}
	if user == nil {
		h.appLogger.Error("get authorised user returned nil user", map[string]any{"user_id": id})
		writeError(w, http.StatusInternalServerError, domain.ErrInternal.Error(), h.appLogger)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(usecase.NewUserDTO(user.ID, user.Email, user.Username)), h.appLogger)
}
