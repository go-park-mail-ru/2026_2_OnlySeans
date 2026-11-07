package profile

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	profilerepo "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/profile"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/utils"
)

type PasswordService interface {
	domain.PasswordHasher
	domain.PasswordVerifier
}

type Handler struct {
	repo      *profilerepo.Repo
	passwords PasswordService
	log       logger.Logger
}

func NewHandler(repo *profilerepo.Repo, passwords PasswordService, log logger.Logger) *Handler {
	return &Handler{repo: repo, passwords: passwords, log: log}
}

func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	id, ok := h.accountID(w, r)
	if !ok {
		return
	}
	data, err := h.repo.Overview(r.Context(), id)
	if err != nil {
		h.failure(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSON(w, http.StatusOK, data, h.log)
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := h.accountID(w, r)
	if !ok {
		return
	}
	var form struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := httprequest.DecodeJSON(w, r, &form); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body", h.log)
		return
	}
	form.Username, form.Email = strings.TrimSpace(form.Username), strings.TrimSpace(form.Email)
	if err := utils.ValidateUsername(form.Username); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid username", h.log)
		return
	}
	if err := utils.ValidateEmail(form.Email); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid email", h.log)
		return
	}
	account, err := h.repo.UpdateAccount(r.Context(), id, form.Username, form.Email)
	if err != nil {
		h.failure(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{"user": account}, h.log)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := h.accountID(w, r)
	if !ok {
		return
	}
	var form struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := httprequest.DecodeJSON(w, r, &form); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body", h.log)
		return
	}
	if err := utils.ValidatePassword(form.New); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid new password", h.log)
		return
	}
	if form.Current == form.New {
		httpresponse.WriteError(w, http.StatusBadRequest, "same password", h.log)
		return
	}
	hash, err := h.repo.PasswordHash(r.Context(), id)
	if err != nil {
		h.failure(w, err)
		return
	}
	if err := h.passwords.Verify(form.Current, hash); err != nil {
		if errors.Is(err, authusecase.ErrPasswordMismatch) {
			httpresponse.WriteError(w, http.StatusBadRequest, "wrong current password", h.log)
			return
		}
		h.failure(w, err)
		return
	}
	newHash, err := h.passwords.Hash(form.New)
	if err != nil {
		h.failure(w, err)
		return
	}
	if err := h.repo.ChangePassword(r.Context(), id, hash, newHash); err != nil {
		h.failure(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetFavorite(w http.ResponseWriter, r *http.Request) {
	id, ok := h.accountID(w, r)
	if !ok {
		return
	}
	filmID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || filmID < 1 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid film id", h.log)
		return
	}
	if err := h.repo.SetFavorite(r.Context(), id, filmID, r.Method == http.MethodPut); err != nil {
		h.failure(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) accountID(w http.ResponseWriter, r *http.Request) (domain.UserID, bool) {
	id, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, "unauthorized", h.log)
	}
	return id, ok
}

func (h *Handler) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpresponse.WriteError(w, http.StatusUnauthorized, "unauthorized", h.log)
	case errors.Is(err, domain.ErrUsernameTaken):
		httpresponse.WriteError(w, http.StatusConflict, "username taken", h.log)
	case errors.Is(err, domain.ErrUserExists):
		httpresponse.WriteError(w, http.StatusConflict, "email taken", h.log)
	case errors.Is(err, profilerepo.ErrPasswordChanged):
		httpresponse.WriteError(w, http.StatusConflict, "password changed", h.log)
	case errors.Is(err, domain.ErrFilmNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "film not found", h.log)
	default:
		h.log.Error("profile request failed", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.log)
	}
}
