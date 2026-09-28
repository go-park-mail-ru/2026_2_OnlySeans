package films

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

const (
	defaultLimit  = 20
	maxLimit      = 100
	defaultOffset = 0
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
	slugVar     = "slug"
)

const (
	filmsPath       = "/api/films"
	collectionsPath = "/api/collections"
	collectionPath  = "/api/collections/{" + slugVar + "}"
)

const internalErrorMessage = "internal server error"

var (
	ErrInvalidLimit  = fmt.Errorf("limit must be an integer from 1 to %d", maxLimit)
	ErrInvalidOffset = errors.New("offset must be a non-negative integer")
	ErrInvalidSlug   = errors.New("invalid collection slug")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{3,128}$`)

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
		err:    ErrInvalidSlug,
		status: http.StatusBadRequest,
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

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.
		HandleFunc(filmsPath, h.ListFilms).
		Methods(http.MethodGet)
	router.
		HandleFunc(collectionsPath, h.ListCollections).
		Methods(http.MethodGet)
	router.
		HandleFunc(collectionPath, h.GetCollection).
		Methods(http.MethodGet)
}

type filmsPage struct {
	Films  []entities.Film `json:"films"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type collectionsResponse struct {
	Collections []entities.CollectionWithFilms `json:"collections"`
}

type collectionResponse struct {
	Collection entities.CollectionWithFilms `json:"collection"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ListFilms(w http.ResponseWriter, r *http.Request) {
	limit, ok := queryInt(r, limitParam, defaultLimit)
	if !ok || limit < 1 || limit > maxLimit {
		writeError(w, ErrInvalidLimit)
		return
	}
	offset, ok := queryInt(r, offsetParam, defaultOffset)
	if !ok || offset < 0 {
		writeError(w, ErrInvalidOffset)
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

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.service.ListCollections(r.Context())
	if err != nil {
		writeError(w, fmt.Errorf("list collections: %w", err))
		return
	}

	result := make([]entities.CollectionWithFilms, 0, len(collections))
	for _, collection := range collections {
		withFilms, err := h.collectionWithFilms(r.Context(), collection)
		if err != nil {
			writeError(w, err)
			return
		}
		result = append(result, withFilms)
	}

	writeJSON(w, http.StatusOK, collectionsResponse{Collections: result})
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)[slugVar]
	if !slugPattern.MatchString(slug) {
		writeError(w, ErrInvalidSlug)
		return
	}

	collection, err := h.service.GetCollectionBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, fmt.Errorf("get collection %q: %w", slug, err))
		return
	}

	withFilms, err := h.collectionWithFilms(r.Context(), collection)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, collectionResponse{Collection: withFilms})
}

func (h *Handler) collectionWithFilms(ctx context.Context, collection entities.Collection) (entities.CollectionWithFilms, error) {
	films, err := h.service.ListCollectionFilms(ctx, collection.ID)
	if err != nil {
		return entities.CollectionWithFilms{}, fmt.Errorf("list films of collection %q: %w", collection.Slug, err)
	}

	return entities.CollectionWithFilms{
		Collection: collection,
		Films:      films,
	}, nil
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
			writeJSON(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}

	log.Printf("films: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: internalErrorMessage})
}
