package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth/usecase"
)

const SessionCookieName = "session_id"

type Handler struct {
	register     RegisterUser
	login        LoginUser
	logout       LogoutUser
	authenticate AuthenticateUser
	getUser      GetUser
	CookieSecure bool
}

type RegisterUser interface {
	Run(ctx context.Context, input usecase.RegisterUserInput) (usecase.AuthOutput, error)
}

type LoginUser interface {
	Run(ctx context.Context, input usecase.LoginUserInput) (usecase.AuthOutput, error)
}

type LogoutUser interface {
	Run(ctx context.Context, input usecase.LogoutInput) error
}

type AuthenticateUser interface {
	Run(ctx context.Context, input usecase.AuthenticateInput) (domain.UserID, error)
}

type GetUser interface {
	Run(ctx context.Context, input usecase.GetUserInput) (usecase.UserDTO, error)
}

type HandlerUseCases struct {
	Register     RegisterUser
	Login        LoginUser
	Logout       LogoutUser
	Authenticate AuthenticateUser
	GetUser      GetUser
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

	return &Handler{
		register:     cases.Register,
		login:        cases.Login,
		logout:       cases.Logout,
		authenticate: cases.Authenticate,
		getUser:      cases.GetUser,
	}, nil
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User userResponse `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// Register handles POST /api/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.register.Run(r.Context(), usecase.RegisterUserInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		writeRegisterError(w, err)
		return
	}

	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	writeJSON(w, http.StatusCreated, authResponse{
		User: userResponse{
			ID:       int(result.User.ID),
			Email:    result.User.Email,
			Username: result.User.Username,
		},
	})
}

func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrUserExists):
		writeError(w, http.StatusConflict, domain.ErrUserExists.Error())
	default:
		log.Printf("auth: register user: %v", err)
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
	}
}

// Login handles POST /api/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.login.Run(r.Context(), usecase.LoginUserInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeLoginError(w, err)
		return
	}

	h.setSessionCookie(w, result.Session.ID, result.Session.ExpiresAt)
	writeJSON(w, http.StatusOK, authResponse{
		User: userResponse{
			ID:       int(result.User.ID),
			Email:    result.User.Email,
			Username: result.User.Username,
		},
	})
}

func writeLoginError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, domain.ErrInvalidCredentials.Error())
	default:
		log.Printf("auth: login user: %v", err)
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
	}
}

// Logout handles POST /api/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.logout.Run(r.Context(), usecase.LogoutInput{SessionID: cookie.Value}); err != nil {
			log.Printf("auth: logout user: %v", err)
			writeError(w, http.StatusInternalServerError, ErrInternal.Error())
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Authenticate(ctx context.Context, sessionID string) (domain.UserID, error) {
	userID, err := h.authenticate.Run(ctx, usecase.AuthenticateInput{SessionID: sessionID})
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			return 0, ErrUnauthorized
		}

		log.Printf("auth: authenticate session: %v", err)
		return 0, ErrInternal
	}

	return userID, nil
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

func toUserResponse(user usecase.UserDTO) userResponse {
	return userResponse{
		ID:       int(user.ID),
		Email:    user.Email,
		Username: user.Username,
	}
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
		writeError(w, http.StatusUnauthorized, ErrUnauthorized.Error())
		return
	}

	user, err := h.getUser.Run(r.Context(), usecase.GetUserInput{UserID: id})
	if err != nil {
		log.Printf("auth: get authorised user: %v", err)
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: toUserResponse(user)})
}
