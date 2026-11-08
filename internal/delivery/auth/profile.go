package auth

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

type ProfileHandler struct {
	update *usecase.UpdateProfileUseCase
	log    logger.Logger
}

func NewProfileHandler(update *usecase.UpdateProfileUseCase, log logger.Logger) *ProfileHandler {
	return &ProfileHandler{update: update, log: log}
}

func (h *ProfileHandler) Register(protect func(http.HandlerFunc) http.HandlerFunc) func(*mux.Router) {
	return func(r *mux.Router) {
		r.HandleFunc("/api/me", protect(h.Update)).Methods(http.MethodPut)
	}
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.log)
		return
	}
	var body struct {
		Email    string `json:"email"`
		Username string `json:"username"`
	}
	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "Проверьте имя и почту.", h.log)
		return
	}
	user, err := h.update.Run(r.Context(), id, body.Email, body.Username)
	if err != nil {
		status, message := http.StatusInternalServerError, "Не удалось сохранить профиль."
		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			status, message = http.StatusBadRequest, "Укажите корректную почту."
		case errors.Is(err, domain.ErrInvalidUsername):
			status, message = http.StatusBadRequest, "Имя должно содержать от 2 до 32 допустимых символов."
		case errors.Is(err, domain.ErrUserExists):
			status, message = http.StatusConflict, "Эта почта уже используется."
		case errors.Is(err, domain.ErrUsernameTaken):
			status, message = http.StatusConflict, "Это имя уже используется."
		case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrUnauthorized):
			status, message = http.StatusUnauthorized, domain.ErrUnauthorized.Error()
		default:
			h.log.Error("update profile failed", map[string]any{"error": err.Error()})
		}
		httpresponse.WriteError(w, status, message, h.log)
		return
	}
	dto := usecase.NewUserDTO(user.ID, user.Email, user.Username)
	dto.AvatarFilename = user.AvatarFilename
	httpresponse.WriteJSON(w, http.StatusOK, toAuthResponse(dto), h.log)
}
