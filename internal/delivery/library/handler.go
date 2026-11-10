package library

import (
	"context"

	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	libraryusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/library"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	saveFailedMsg   = "Не удалось сохранить библиотеку."
	invalidDataMsg  = "Проверьте данные запроса."
	filmNotFoundMsg = "Фильм не найден."
)

type HandlerUseCases struct {
	GetLibrary     *libraryusecase.GetLibraryUseCase
	SetSaved       *libraryusecase.SetSavedUseCase
	GetPreferences *libraryusecase.GetPreferencesUseCase
	SavePreference *libraryusecase.SavePreferenceUseCase
}

type Handler struct {
	useCases  HandlerUseCases
	appLogger logger.Logger
}

func NewHandler(cases HandlerUseCases, appLogger logger.Logger) *Handler {
	return &Handler{useCases: cases, appLogger: appLogger}
}

func user(r *http.Request) domain.UserID {
	id, _ := auth.UserIDFromContext(r.Context())
	return id
}

func (h *Handler) fail(ctx context.Context, w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, saveFailedMsg
	switch {
	case errors.Is(err, domain.ErrInvalidLibraryData), errors.Is(err, httprequest.ErrInvalidBody):
		status, message = http.StatusBadRequest, invalidDataMsg
	case errors.Is(err, domain.ErrUnauthorized):
		status, message = http.StatusUnauthorized, domain.ErrUnauthorized.Error()
	case errors.Is(err, domain.ErrFilmNotFound):
		status, message = http.StatusNotFound, filmNotFoundMsg
	default:
		h.appLogger.ErrorContext(ctx, "library request failed", map[string]any{"error": err.Error()})
	}
	httpresponse.WriteErrorContext(ctx, w, status, message, h.appLogger)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	data, err := h.useCases.GetLibrary.Run(r.Context(), user(r))
	if err != nil {
		h.fail(r.Context(), w, err)
		return
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, data, h.appLogger)
}

func (h *Handler) SetSaved(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Saved *bool `json:"saved"`
	}
	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		h.fail(r.Context(), w, err)
		return
	}
	film, err := strconv.ParseInt(mux.Vars(r)["film"], 10, 64)
	if err != nil || body.Saved == nil {
		h.fail(r.Context(), w, domain.ErrInvalidLibraryData)
		return
	}
	if err = h.useCases.SetSaved.Run(r.Context(), user(r), domain.List(mux.Vars(r)["list"]), film, *body.Saved); err != nil {
		h.fail(r.Context(), w, err)
		return
	}
	h.Get(w, r)
}
