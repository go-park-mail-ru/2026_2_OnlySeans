package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

const sessionCookieName = "session_id"

type Handler struct {
	UseCase      *UseCase
	CookieSecure bool
}

func NewHandler(uc *UseCase) (*Handler, error) {
	if uc == nil {
		return nil, ErrNilUseCase
	}

	return &Handler{UseCase: uc}, nil
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

type userCtxKey struct{}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// statusForError переводит ошибку из usecase в HTTP-код.
func statusForError(err error) int {
	switch {
	case errors.Is(err, ErrUserExists):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrInternal):
		return http.StatusInternalServerError
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusBadRequest
	}
}

// Register обрабатывает POST /api/register
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

	result, err := h.UseCase.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}

	h.setSessionCookie(w, result.Session)
	writeJSON(w, http.StatusCreated, authResponse{User: *result.User})
}

// Login обрабатывает POST /api/login
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

	result, err := h.UseCase.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}

	h.setSessionCookie(w, result.Session)
	writeJSON(w, http.StatusOK, authResponse{User: *result.User})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, s *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
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
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Logout обрабатывает POST /api/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		if err := h.UseCase.Logout(cookie.Value); err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}
	}

	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Me обрабатывает GET /api/me (только через RequireAuth)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not supported")
		return
	}

	user, ok := UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrUnauthorized.Error())
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: *user})
}

// RequireAuth пропускает запрос дальше только с валидной сессией
// и кладёт пользователя в контекст.
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)

		if err != nil {
			writeError(w, http.StatusUnauthorized, ErrUnauthorized.Error())
			return
		}

		user, err := h.UseCase.Authenticate(r.Context(), cookie.Value)
		if err != nil {
			writeError(w, statusForError(err), err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey{}, user)
		next(w, r.WithContext(ctx))
	}
}

func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userCtxKey{}).(*User)
	return user, ok
}
