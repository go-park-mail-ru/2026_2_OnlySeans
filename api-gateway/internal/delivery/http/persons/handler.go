package persons

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	personsclient "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/personscleint"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	defaultLimit = 20
	minLimit     = 1
	maxLimit     = 100

	defaultOffset = 0
	minOffset     = 0

	idVar       = "id"
	limitParam  = "limit"
	offsetParam = "offset"
)

var (
	ErrInvalidID        = errors.New("invalid person ID")
	ErrInvalidLimit     = errors.New("invalid limit")
	ErrInvalidOffset    = errors.New("invalid offset")
	ErrInvalidBirthDate = errors.New("invalid birth date format")
)

type Handler struct {
	personClient personsclient.PersonClient
	appLogger    logger.Logger
}

func NewHandler(client personsclient.PersonClient, appLogger logger.Logger) (*Handler, error) {
	if client == nil {
		return nil, errors.New("person client is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	return &Handler{
		personClient: client,
		appLogger:    appLogger,
	}, nil
}

func (h *Handler) ListPersons(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "pagination", err)
		h.writeError(r.Context(), w, err)

		return
	}

	result, err := h.personClient.ListPersons(r.Context(), limit, offset)
	if err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	response := personsPage{
		Persons: toPersonResponseDTOs(result.GetPersons()),
		Total:   result.GetTotal(),
		Limit:   limit,
		Offset:  offset,
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

func (h *Handler) GetPerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "id", err)
		h.writeError(r.Context(), w, err)

		return
	}

	person, err := h.personClient.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(person),
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

type personRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BirthDate string `json:"birth_date"`
}

func (h *Handler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	var request personRequest

	if err := httprequest.DecodeJSON(w, r, &request); err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		h.writeError(r.Context(), w, err)

		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "birth_date", err)
		h.writeError(r.Context(), w, err)

		return
	}

	id, err := h.personClient.CreatePerson(r.Context(), request.FirstName, request.LastName, birthDate)
	if err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	response := createPersonResponse{ID: id}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, response, h.appLogger)
}

func (h *Handler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "id", err)
		h.writeError(r.Context(), w, err)

		return
	}

	var request personRequest

	if err := httprequest.DecodeJSON(w, r, &request); err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		h.writeError(r.Context(), w, err)

		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "birth_date", err)
		h.writeError(r.Context(), w, err)

		return
	}

	person, err := h.personClient.UpdatePerson(r.Context(), id, request.FirstName, request.LastName, birthDate)
	if err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(person),
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

func (h *Handler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "id", err)
		h.writeError(r.Context(), w, err)

		return
	}

	if err := h.personClient.DeletePerson(r.Context(), id); err != nil {
		h.writeError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(mux.Vars(r)[idVar], 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidID
	}

	return id, nil
}

func pagination(r *http.Request) (int, int, error) {
	limit := defaultLimit
	offset := defaultOffset

	if value := r.URL.Query().Get(limitParam); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < minLimit || parsed > maxLimit {
			return 0, 0, ErrInvalidLimit
		}

		limit = parsed
	}

	if value := r.URL.Query().Get(offsetParam); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < minOffset {
			return 0, 0, ErrInvalidOffset
		}

		offset = parsed
	}

	return limit, offset, nil
}

func parseBirthDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return nil, ErrInvalidBirthDate
	}

	return &date, nil
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error) {
	writePersonError(ctx, w, err, h.appLogger)
}

func writePersonError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, ErrInvalidBirthDate),
		errors.Is(err, httprequest.ErrInvalidBody),
		errors.Is(err, personsclient.ErrInvalidArgument):
		status = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, catalogclient.ErrFilmNotFound), errors.Is(err, personsclient.ErrPersonNotFound),
		errors.Is(err, personsclient.ErrFilmNotFound):
		status = http.StatusNotFound
		message = err.Error()
	}

	httpresponse.WriteErrorContext(ctx, w, status, message, appLogger)
}
