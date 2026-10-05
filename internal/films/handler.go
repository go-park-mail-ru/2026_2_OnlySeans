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

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type filmsPage struct {
	Films  []entities.Film `json:"films"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type filmResponse struct {
	Film entities.Film `json:"film"`
}

type collectionsPage struct {
	Collections []entities.CollectionWithFilms `json:"collections"`
	Total       int                            `json:"total"`
	Limit       int                            `json:"limit"`
	Offset      int                            `json:"offset"`
}

type collectionPage struct {
	Collection entities.CollectionWithFilms `json:"collection"`
	Total      int                          `json:"total"`
	Limit      int                          `json:"limit"`
	Offset     int                          `json:"offset"`
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

	films, total, err := h.service.ListFilms(r.Context(), limit, offset)
	if err != nil {
		writeError(w, fmt.Errorf("list films: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, filmsPage{
		Films:  films,
		Total:  total,
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

	film, err := h.service.GetFilmByID(r.Context(), entities.FilmID(id))
	if err != nil {
		writeError(w, fmt.Errorf("get film %d: %w", id, err))
		return
	}

	writeJSON(w, http.StatusOK, filmResponse{Film: film})
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	collections, total, err := h.service.ListCollections(r.Context(), limit, offset)
	if err != nil {
		writeError(w, fmt.Errorf("list collections: %w", err))
		return
	}

	result := make([]entities.CollectionWithFilms, 0, len(collections))
	for _, collection := range collections {
		withFilms, _, err := h.collectionWithFilms(r.Context(), collection, maxLimit, defaultOffset)
		if err != nil {
			writeError(w, err)
			return
		}
		result = append(result, withFilms)
	}

	writeJSON(w, http.StatusOK, collectionsPage{
		Collections: result,
		Total:       total,
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

	collection, err := h.service.GetCollectionByID(r.Context(), entities.CollectionID(id))
	if err != nil {
		writeError(w, fmt.Errorf("get collection %d: %w", id, err))
		return
	}

	withFilms, total, err := h.collectionWithFilms(r.Context(), collection, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, collectionPage{
		Collection: withFilms,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	})
}

func (h *Handler) collectionWithFilms(ctx context.Context, collection entities.Collection, limit, offset int) (entities.CollectionWithFilms, int, error) {
	films, total, err := h.service.ListCollectionFilms(ctx, collection.ID, limit, offset)
	if err != nil {
		return entities.CollectionWithFilms{}, 0, fmt.Errorf("list films of collection %d: %w", collection.ID, err)
	}

	return entities.CollectionWithFilms{
		Collection: collection,
		Films:      films,
	}, total, nil
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
