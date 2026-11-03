package persons

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	maxBodyBytes  = 1 << 20

	idVar       = "id"
	limitParam  = "limit"
	offsetParam = "offset"

	internalErrorMessage = "internal server error"
)

var (
	ErrInvalidID     = errors.New("invalid person ID")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("invalid offset")
	ErrInvalidBody   = errors.New("invalid request body")
)

type HandlerUseCases struct {
	Create    *personusecase.CreatePersonUseCase
	Get       *personusecase.GetPersonUseCase
	List      *personusecase.ListPersonsUseCase
	Update    *personusecase.UpdatePersonUseCase
	Delete    *personusecase.DeletePersonUseCase
	AppLogger logger.Logger
}

type Handler struct {
	useCases HandlerUseCases
	logger   logger.Logger
}

func NewHandler(useCases HandlerUseCases) (*Handler, error) {
	if useCases.Create == nil ||
		useCases.Get == nil ||
		useCases.List == nil ||
		useCases.Update == nil ||
		useCases.Delete == nil ||
		useCases.AppLogger == nil {
		return nil, errors.New("person use cases must not be nil")
	}

	return &Handler{
		useCases: useCases,
		logger:   useCases.AppLogger,
	}, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.useCases.List.Run(r.Context(), personusecase.ListPersonsInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	response := personsPage{
		Persons: toPersonResponseDTOs(result.Persons),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.useCases.Get.Run(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(result.Person),
	}

	writeJSON(w, http.StatusOK, response)
}

type personRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	BirthDate string `json:"birth_date"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request personRequest

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.useCases.Create.Run(r.Context(), personusecase.CreatePersonInput{
		Person: domain.PersonInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
			BirthDate: birthDate,
		},
	})
	if err != nil {
		writeError(w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(result.Person),
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var request personRequest

	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	birthDate, err := parseBirthDate(request.BirthDate)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.useCases.Update.Run(r.Context(), personusecase.UpdatePersonInput{
		PersonID: id,
		Person: domain.PersonInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
			BirthDate: birthDate,
		},
	})
	if err != nil {
		writeError(w, err)
		return
	}

	response := personResponse{
		Person: toPersonResponseDTO(result.Person),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	err = h.useCases.Delete.Run(r.Context(), personusecase.DeletePersonInput{
		PersonID: id,
	})
	if err != nil {
		writeError(w, err)
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

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidBody, err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidBody
	}

	return nil
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return
	}
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := internalErrorMessage

	switch {
	case errors.Is(err, ErrInvalidID),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, ErrInvalidBody),
		errors.Is(err, domain.ErrInvalidFirstName),
		errors.Is(err, domain.ErrInvalidLastName),
		errors.Is(err, domain.ErrInvalidBirthDate):
		status = http.StatusBadRequest
		message = err.Error()

	case errors.Is(err, domain.ErrPersonNotFound):
		status = http.StatusNotFound
		message = err.Error()
	}

	writeJSON(w, status, errorResponse{
		Error: message,
	})
}
