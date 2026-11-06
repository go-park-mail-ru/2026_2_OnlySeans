package films

import (
	"errors"
	"fmt"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	defaultLimit            = 20
	maxFilmRequestBodyBytes = 1 << 20
	minLimit                = 1
	maxLimit                = 100
	defaultOffset           = 0
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
	idVar       = "id"
)

var (
	ErrInvalidLimit        = errors.New("invalid limit")
	ErrInvalidOffset       = errors.New("offset must be a non-negative integer")
	ErrInvalidFilmID       = errors.New("film id must be a positive integer")
	ErrInvalidCollectionID = errors.New("collection id must be a positive integer")
)

var clientErrors = []struct {
	err    error
	status int
}{
	{
		err:    ErrInvalidLimit,
		status: http.StatusBadRequest,
	},
	{
		err:    ErrInvalidOffset,
		status: http.StatusBadRequest,
	},
	{
		err:    ErrInvalidFilmID,
		status: http.StatusBadRequest,
	},
	{
		err:    ErrInvalidCollectionID,
		status: http.StatusBadRequest,
	},
	{
		err:    domain.ErrFilmNotFound,
		status: http.StatusNotFound,
	},
	{
		err:    domain.ErrCollectionNotFound,
		status: http.StatusNotFound,
	},
	{
		err:    domain.ErrInvalidFilmInput,
		status: http.StatusBadRequest,
	},
}

type HandlerUseCases struct {
	ListFilms       *usecase.ListFilmsUseCase
	GetFilm         *usecase.GetFilmUseCase
	CreateFilm      *usecase.CreateFilmUseCase
	UpdateFilm      *usecase.UpdateFilmUseCase
	DeleteFilm      *usecase.DeleteFilmUseCase
	ListCollections *usecase.ListCollectionsUseCase
	GetCollection   *usecase.GetCollectionUseCase
}

type Handler struct {
	listFilms       *usecase.ListFilmsUseCase
	getFilm         *usecase.GetFilmUseCase
	createFilm      *usecase.CreateFilmUseCase
	updateFilm      *usecase.UpdateFilmUseCase
	deleteFilm      *usecase.DeleteFilmUseCase
	listCollections *usecase.ListCollectionsUseCase
	getCollection   *usecase.GetCollectionUseCase
	appLogger       logger.Logger
}

func NewHandler(cases HandlerUseCases, appLogger logger.Logger) (*Handler, error) {
	if cases.ListFilms == nil {
		return nil, errors.New("list films use case is nil")
	}
	if cases.GetFilm == nil {
		return nil, errors.New("get film use case is nil")
	}
	if cases.ListCollections == nil {
		return nil, errors.New("list collections use case is nil")
	}
	if cases.GetCollection == nil {
		return nil, errors.New("get collection use case is nil")
	}
	if cases.CreateFilm == nil {
		return nil, errors.New("create film use case is nil")
	}
	if cases.UpdateFilm == nil {
		return nil, errors.New("update film use case is nil")
	}
	if cases.DeleteFilm == nil {
		return nil, errors.New("delete film use case is nil")
	}
	if appLogger == nil {
		return nil, errors.New("logger use case is nil")
	}
	return &Handler{
		listFilms:       cases.ListFilms,
		getFilm:         cases.GetFilm,
		createFilm:      cases.CreateFilm,
		updateFilm:      cases.UpdateFilm,
		deleteFilm:      cases.DeleteFilm,
		listCollections: cases.ListCollections,
		getCollection:   cases.GetCollection,
		appLogger:       appLogger,
	}, nil
}

type filmsPage struct {
	Films  []filmCardResponseDTO `json:"films"`
	Total  int                   `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

type filmResponse struct {
	Film filmResponseDTO `json:"film"`
}

type collectionsPage struct {
	Collections []collectionResponseDTO `json:"collections"`
	Total       int                     `json:"total"`
	Limit       int                     `json:"limit"`
	Offset      int                     `json:"offset"`
}

type collectionPage struct {
	Collection collectionWithFilmsResponseDTO `json:"collection"`
	Total      int                            `json:"total"`
	Limit      int                            `json:"limit"`
	Offset     int                            `json:"offset"`
}

func (h *Handler) ListFilms(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	input := usecase.NewListFilmsInput(limit, offset)

	result, err := h.listFilms.Run(r.Context(), input)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, filmsPage{
		Films:  toFilmCardResponseDTOs(result.Films),
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	}, h.appLogger)
}

func (h *Handler) GetFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}

	input := usecase.NewGetFilmInput(domain.FilmID(id))
	result, err := h.getFilm.Run(r.Context(), input)
	if err != nil {
		writeError(w, fmt.Errorf("get film %d: %w", id, err), h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, filmResponse{Film: toFilmResponseDTO(result.Film)}, h.appLogger)
}

func (h *Handler) CreateFilm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFilmRequestBodyBytes)
	input, err := decodeCreateFilm(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	id, err := h.createFilm.Run(r.Context(), input)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, createFilmResponse{ID: id}, h.appLogger)
}

func (h *Handler) UpdateFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFilmRequestBodyBytes)
	input, err := decodeCreateFilm(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	if err := h.updateFilm.Run(r.Context(), domain.FilmID(id), input); err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}

	if err := h.deleteFilm.Run(r.Context(), domain.FilmID(id)); err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	input := usecase.NewListCollectionsInput(limit, offset)

	result, err := h.listCollections.Run(r.Context(), input)
	if err != nil {
		writeError(w, fmt.Errorf("list collections: %w", err), h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, collectionsPage{
		Collections: toCollectionResponseDTOs(result.Collections),
		Total:       result.Total,
		Limit:       limit,
		Offset:      offset,
	}, h.appLogger)
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidCollectionID, h.appLogger)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	input := usecase.NewGetCollectionInput(domain.CollectionID(id), limit, offset)

	result, err := h.getCollection.Run(r.Context(), input)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, collectionPage{
		Collection: toCollectionWithFilmsResponseDTO(result.Collection, result.Films),
		Total:      result.Total,
		Limit:      limit,
		Offset:     offset,
	}, h.appLogger)
}

func pagination(r *http.Request) (limit, offset int, err error) {
	limit, ok := queryInt(r, limitParam, defaultLimit)
	if !ok || limit < minLimit || limit > maxLimit {
		return 0, 0, fmt.Errorf("%w: must be an integer from %d to %d", ErrInvalidLimit, minLimit, maxLimit)
	}
	offset, ok = queryInt(r, offsetParam, defaultOffset)
	if !ok || offset < 0 {
		return 0, 0, ErrInvalidOffset
	}
	return limit, offset, nil
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)[idVar], 10, 64)
	return id, err == nil && id > 0
}

func queryInt(r *http.Request, key string, fallback int) (int, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil
}

func writeError(w http.ResponseWriter, err error, appLogger logger.Logger) {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		httpresponse.WriteError(w, http.StatusRequestEntityTooLarge, "request body exceeds 1 MiB", appLogger)
		return
	}

	for _, known := range clientErrors {
		if errors.Is(err, known.err) {
			msg := known.err.Error()
			if known.status == http.StatusBadRequest {
				msg = err.Error()
			}
			httpresponse.WriteError(w, known.status, msg, appLogger)
			return
		}
	}

	if appLogger != nil {
		appLogger.Error("unexpected error", map[string]any{"error": err})
	}
	httpresponse.WriteError(w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
}
