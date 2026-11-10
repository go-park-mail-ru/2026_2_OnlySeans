package auth

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ProfileHandler struct {
	GetProfileUseCase    *usecase.GetProfileUseCase
	UpdateProfileUseCase *usecase.UpdateProfileUseCase
	appLogger            logger.Logger
}

func NewProfileHandler(get *usecase.GetProfileUseCase, update *usecase.UpdateProfileUseCase, appLogger logger.Logger) *ProfileHandler {
	return &ProfileHandler{
		GetProfileUseCase:    get,
		UpdateProfileUseCase: update,
		appLogger:            appLogger,
	}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	profile, err := h.GetProfileUseCase.Run(r.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		message := httpresponse.InternalErrorMessage

		if errors.Is(err, domain.ErrUserNotFound) || errors.Is(err, domain.ErrUnauthorized) {
			status, message = http.StatusUnauthorized, domain.ErrUnauthorized.Error()
		}

		httpresponse.WriteErrorContext(r.Context(), w, status, message, h.appLogger)

		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, toProfileResponse(profile), h.appLogger)
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Username string `json:"username"`
	}

	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "Проверьте имя и почту.", h.appLogger)
		return
	}

	user, err := h.UpdateProfileUseCase.Run(r.Context(), id, body.Email, body.Username)
	if err != nil {
		status, message := http.StatusInternalServerError, "Не удалось сохранить профиль."

		switch {
		case errors.Is(err, domain.ErrInvalidEmail):
			status, message = http.StatusBadRequest, "Укажите корректную почту."
		case errors.Is(err, domain.ErrInvalidUsername):
			status, message = http.StatusBadRequest, "Имя должно содержать от 2 до 32 допустимых символов."
		case errors.Is(err, domain.ErrUserExists):
			status, message = http.StatusConflict, "Эта почта уже используется."

			if r.URL.Path == "/api/me/profile" {
				message = "email taken"
			}

		case errors.Is(err, domain.ErrUsernameTaken):
			status, message = http.StatusConflict, "Это имя уже используется."

			if r.URL.Path == "/api/me/profile" {
				message = "username taken"
			}

		case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrUnauthorized):
			status, message = http.StatusUnauthorized, domain.ErrUnauthorized.Error()
		default:
			h.appLogger.ErrorContext(r.Context(), "update profile failed", map[string]any{"error": err.Error()})
		}

		httpresponse.WriteErrorContext(r.Context(), w, status, message, h.appLogger)

		return
	}

	dto := usecase.NewUserDTO(user.ID, user.Email, user.Username)
	dto.AvatarFilename = user.AvatarFilename
	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, toAuthResponse(dto), h.appLogger)
}
