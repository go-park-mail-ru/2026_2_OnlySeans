package library

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/library"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	saveFailedMsg   = "Не удалось сохранить библиотеку."
	invalidDataMsg  = "Проверьте данные запроса."
	filmNotFoundMsg = "Фильм не найден."
)

type Handler struct {
	repo *service.PostgresRepo
	log  logger.Logger
}

func NewHandler(repo *service.PostgresRepo, log logger.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) Register(protect func(http.HandlerFunc) http.HandlerFunc) func(*mux.Router) {
	return func(r *mux.Router) {
		r.HandleFunc("/api/me/preferences", protect(h.Preferences)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/preferences", protect(h.SavePreference)).Methods(http.MethodPut)
		r.HandleFunc("/api/me/library", protect(h.Get)).Methods(http.MethodGet)
		r.HandleFunc("/api/me/library/lists/{list}/{film}", protect(h.SetSaved)).Methods(http.MethodPut)
	}
}

func user(r *http.Request) domain.UserID {
	id, _ := auth.UserIDFromContext(r.Context())
	return id
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, saveFailedMsg
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, service.ErrInvalid), errors.Is(err, httprequest.ErrInvalidBody):
		status, message = http.StatusBadRequest, invalidDataMsg
	case errors.As(err, &pgErr) && pgErr.Code == storage.ForeignKeyViolation:
		status, message = http.StatusNotFound, filmNotFoundMsg
	default:
		h.log.Error("library request failed", map[string]any{"error": err.Error()})
	}
	httpresponse.WriteError(w, status, message, h.log)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.Get(r.Context(), user(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, data, h.log)
}

func (h *Handler) SetSaved(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Saved *bool `json:"saved"`
	}
	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		h.fail(w, err)
		return
	}
	film, err := strconv.ParseInt(mux.Vars(r)["film"], 10, 64)
	if err != nil || body.Saved == nil {
		h.fail(w, service.ErrInvalid)
		return
	}
	if err = h.repo.SetSaved(r.Context(), user(r), service.List(mux.Vars(r)["list"]), film, *body.Saved); err != nil {
		h.fail(w, err)
		return
	}
	h.Get(w, r)
}
