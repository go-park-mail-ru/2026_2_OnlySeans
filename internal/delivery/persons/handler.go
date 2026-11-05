package persons

import (
	"errors"
	"fmt"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

const (
	defaultLimit = 20
	minLimit     = 1
	maxLimit     = 100

	defaultOffset = 0

	idVar       = "id"
	limitParam  = "limit"
	offsetParam = "offset"
)

var (
	ErrInvalidID     = errors.New("invalid person ID")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("invalid offset")
)

type HandlerUseCases struct {
	CreatePerson *personusecase.CreatePersonUseCase
	GetPerson    *personusecase.GetPersonUseCase
	ListPersons  *personusecase.ListPersonsUseCase
	UpdatePerson *personusecase.UpdatePersonUseCase
	DeletePerson *personusecase.DeletePersonUseCase
}

type Handler struct {
	useCases HandlerUseCases
	logger   logger.Logger
}

func NewHandler(cases HandlerUseCases, appLogger logger.Logger) (*Handler, error) {
	if cases.CreatePerson == nil {
		return nil, errors.New("create person use case is nil")
	}
	if cases.GetPerson == nil {
		return nil, errors.New("get person use case is nil")
	}
	if cases.ListPersons == nil {
		return nil, errors.New("list person use case is nil")
	}
	if cases.UpdatePerson == nil {
		return nil, errors.New("update person use case is nil")
	}
	if cases.DeletePerson == nil {
		return nil, errors.New("delete person use case is nil")
	}
	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	return &Handler{
		useCases: cases,
		logger:   appLogger,
	}, nil
}

func (h *Handler) ListPersons(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	result, err := h.useCases.ListPersons.Run(r.Context(), personusecase.ListPersonsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := personsPage{
		Persons: toPersonResponseDTOs(result.Persons),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	}

	httpresponse.WriteJSON(w, http.StatusOK, response, h.logger)
}

func (h *Handler) GetPerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	result, err := h.useCases.GetPerson.Run(r.Context(), id)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(result.Person),
	}

	httpresponse.WriteJSON(w, http.StatusOK, response, h.logger)
}

type personRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BirthDate string `json:"birth_date"`
}

func (h *Handler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	var request personRequest

	if err := httprequest.DecodeJSON(w, r, &request); err != nil {
		h.writeError(w, err)
		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		h.writeError(w, err)
		return
	}

	result, err := h.useCases.CreatePerson.Run(r.Context(), personusecase.CreatePersonInput{
		Person: domain.PersonInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
			BirthDate: birthDate,
		},
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := createPersonResponse{ID: result.ID}

	httpresponse.WriteJSON(w, http.StatusCreated, response, h.logger)
}

func (h *Handler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	var request personRequest

	if err := httprequest.DecodeJSON(w, r, &request); err != nil {
		h.writeError(w, err)
		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		h.writeError(w, err)
		return
	}

	result, err := h.useCases.UpdatePerson.Run(r.Context(), personusecase.UpdatePersonInput{
		PersonID: id,
		Person: domain.PersonInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
			BirthDate: birthDate,
		},
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(result.Person),
	}

	httpresponse.WriteJSON(w, http.StatusOK, response, h.logger)
}

func (h *Handler) DeletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.writeError(w, err)
		return
	}

	err = h.useCases.DeletePerson.Run(r.Context(), personusecase.DeletePersonInput{
		PersonID: id,
	})
	if err != nil {
		h.writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (domain.PersonID, error) {
	idString := mux.Vars(r)[idVar]

	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidID
	}

	return domain.PersonID(id), nil
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
		if err != nil || parsed < 0 {
			return 0, 0, ErrInvalidOffset
		}
		offset = parsed
	}

	return limit, offset, nil
}

func parseBirthDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid birth date format", domain.ErrInvalidBirthDate)
	}

	return &date, nil
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := httpresponse.InternalErrorMessage

	switch {
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, httprequest.ErrInvalidBody),
		errors.Is(err, domain.ErrInvalidFirstName),
		errors.Is(err, domain.ErrInvalidLastName),
		errors.Is(err, domain.ErrInvalidBirthDate):
		status = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, domain.ErrPersonNotFound):
		status = http.StatusNotFound
		message = err.Error()
	}

	httpresponse.WriteError(w, status, message, h.logger)
}
