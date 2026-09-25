package films

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

const internalErrorMessage = "внутренняя ошибка сервера"

var clientErrors = []struct {
	err    error
	status int
}{
	{ErrInvalidLimit, http.StatusBadRequest},
	{ErrInvalidOffset, http.StatusBadRequest},
	{ErrInvalidSlug, http.StatusBadRequest},
	{ErrCollectionNotFound, http.StatusNotFound},
}

type Handler struct {
	uc *UseCase
}

func NewHandler(uc *UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/films", h.ListFilms)
	mux.HandleFunc("GET /api/collections", h.ListCollections)
	mux.HandleFunc("GET /api/collections/{slug}", h.GetCollection)
}

type collectionsResponse struct {
	Collections []CollectionWithFilms `json:"collections"`
}

type collectionResponse struct {
	Collection CollectionWithFilms `json:"collection"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ListFilms(w http.ResponseWriter, r *http.Request) {
	limit, ok := queryInt(r, "limit", DefaultLimit)
	if !ok {
		writeError(w, ErrInvalidLimit)
		return
	}
	offset, ok := queryInt(r, "offset", 0)
	if !ok {
		writeError(w, ErrInvalidOffset)
		return
	}

	page, err := h.uc.ListFilms(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.uc.ListCollections(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collectionsResponse{Collections: collections})
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	collection, err := h.uc.GetCollection(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collectionResponse{Collection: collection})
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
		log.Printf("films: запись ответа: %v", err)
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
