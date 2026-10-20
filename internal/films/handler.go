package films

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films/usecase"
)

const (
	defaultLimit  = 20
	minLimit      = 1
	maxLimit      = 100
	defaultOffset = 0
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
	idVar       = "id"
)

const internalErrorMessage = "internal server error"

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
		err:    ErrFilmNotFound,
		status: http.StatusNotFound,
	},
	{
		err:    ErrCollectionNotFound,
		status: http.StatusNotFound,
	},
}

type ListFilms interface {
	Run(ctx context.Context, input usecase.ListInput) (usecase.ListFilmsOutput, error)
}

type ListCollections interface {
	Run(ctx context.Context, input usecase.ListInput) (usecase.ListCollectionsOutput, error)
}

type GetFilm interface {
	Run(ctx context.Context, id entities.FilmID) (entities.Film, error)
}

type GetCollection interface {
	Run(ctx context.Context, id entities.CollectionID, input usecase.ListInput) (usecase.GetCollectionOutput, error)
}

type HandlerUseCases struct {
	ListFilms       ListFilms
	GetFilm         GetFilm
	ListCollections ListCollections
	GetCollection   GetCollection
}

type Handler struct {
	listFilms       ListFilms
	getFilm         GetFilm
	listCollections ListCollections
	getCollection   GetCollection
}

func NewHandler(cases HandlerUseCases) (*Handler, error) {
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
	return &Handler{
		listFilms:       cases.ListFilms,
		getFilm:         cases.GetFilm,
		listCollections: cases.ListCollections,
		getCollection:   cases.GetCollection,
	}, nil
}

type filmsPage struct {
	Films  []filmResponseDTO `json:"films"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
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

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ListFilms(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	input := usecase.ListInput{
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.listFilms.Run(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, filmsPage{
		Films:  toFilmResponseDTOs(result.Films),
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) GetFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID)
		return
	}

	film, err := h.getFilm.Run(r.Context(), entities.FilmID(id))
	if err != nil {
		writeError(w, fmt.Errorf("get film %d: %w", id, err))
		return
	}

	writeJSON(w, http.StatusOK, filmResponse{Film: toFilmResponseDTO(film)})
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	input := usecase.ListInput{
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.listCollections.Run(r.Context(), input)
	if err != nil {
		writeError(w, fmt.Errorf("list collections: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, collectionsPage{
		Collections: toCollectionResponseDTOs(result.Collections),
		Total:       result.Total,
		Limit:       limit,
		Offset:      offset,
	})
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidCollectionID)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.getCollection.Run(r.Context(), entities.CollectionID(id), usecase.ListInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, collectionPage{
		Collection: toCollectionWithFilmsResponseDTO(result.Collection, result.Films),
		Total:      result.Total,
		Limit:      limit,
		Offset:     offset,
	})
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("films: write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	for _, known := range clientErrors {
		if errors.Is(err, known.err) {
			msg := known.err.Error()
			if known.status == http.StatusBadRequest {
				msg = err.Error()
			}
			writeJSON(w, known.status, errorResponse{Error: msg})
			return
		}
	}

	log.Printf("films: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: internalErrorMessage})
}
