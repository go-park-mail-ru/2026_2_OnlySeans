package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "session_id"

type Handler struct {
	Users        UserRepository
	Sessions     SessionStore
	CookieSecure bool
}

func NewHandler(users UserRepository, sessions SessionStore) (*Handler, error) {
	if users == nil {
		return nil, ErrNilRepo
	}

	if sessions == nil {
		return nil, ErrNilSessionStore
	}

	return &Handler{Users: users, Sessions: sessions}, nil
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User User `json:"user"`
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

	for _, err := range []error{
		ValidateEmail(req.Email),
		ValidateUsername(req.Username),
		ValidatePassword(req.Password),
	} {
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	id, err := h.Users.Create(r.Context(), req.Email, req.Username, string(hash))
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			writeError(w, http.StatusConflict, ErrUserExists.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	user, err := h.Users.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	h.startSession(w, r, user, http.StatusCreated)
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

	if err := ValidateEmail(req.Email); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Password == "" {
		writeError(w, http.StatusBadRequest, ErrInvalidPassword.Error()+": password required")
		return
	}

	user, err := h.Users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, ErrInvalidCredentials.Error())
		return
	}

	h.startSession(w, r, user, http.StatusOK)
}

// Logout handles POST /api/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		if err := h.Sessions.Delete(r.Context(), cookie.Value); err != nil {
			writeError(w, http.StatusInternalServerError, ErrInternal.Error())
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Authenticate(ctx context.Context, sessionID string) (UserID, error) {
	session, err := h.Sessions.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return 0, ErrUnauthorized
		}

		return 0, ErrInternal
	}

	return session.UserID, nil
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, user *User, status int) {
	session, err := h.Sessions.Create(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	h.setSessionCookie(w, session)
	writeJSON(w, status, authResponse{User: *user})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, s *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    s.ID,
		Path:     "/",
		Expires:  s.ExpiresAt,
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
		writeError(w, http.StatusUnauthorized, ErrUnauthorized.Error())
		return
	}

	user, err := h.Users.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternal.Error())
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: *user})
}
