package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	UseCase *UseCase
}

func NewHandler(uc *UseCase) *Handler {
	return &Handler{UseCase: uc}
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
	User  *User  `json:"user"`
	Token string `json:"token,omitempty"` // пусто, пока модуль сессий (SessionIssuer) не подключён
}

type errorResponse struct {
	Error string `json:"error"`
}

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
	default:
		return http.StatusBadRequest
	}
}

// Register обрабатывает POST /api/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}

	result, err := h.UseCase.Register(req.Email, req.Username, req.Password)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{User: result.User, Token: result.Token})
}

// Login обрабатывает POST /api/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "метод не поддерживается")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректное тело запроса")
		return
	}

	result, err := h.UseCase.Login(req.Email, req.Password)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, authResponse{User: result.User, Token: result.Token})
}
