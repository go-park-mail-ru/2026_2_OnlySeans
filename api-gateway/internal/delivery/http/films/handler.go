package films

import (
	"context"
	"encoding/json"

	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	defaultLimit            = 20
	maxFilmRequestBodyBytes = 1 << 20
	minLimit                = 1
	maxLimit                = 100
	defaultOffset           = 0
)

const (
	limitParam          = "limit"
	offsetParam         = "offset"
	idVar               = "id"
	detailsEnabledValue = "true"
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
		err:    catalogclient.ErrFilmNotFound,
		status: http.StatusNotFound,
	},
	{
		err:    catalogclient.ErrCollectionNotFound,
		status: http.StatusNotFound,
	},
	{
		err:    catalogclient.ErrInvalidFilmInput,
		status: http.StatusBadRequest,
	},
}

type Handler struct {
	client    catalogclient.CatalogClient
	appLogger logger.Logger
}

func NewHandler(client catalogclient.CatalogClient, appLogger logger.Logger) (*Handler, error) {
	if client == nil || appLogger == nil {
		return nil, errors.New("catalog client and logger are required")
	}

	return &Handler{
		client:    client,
		appLogger: appLogger,
	}, nil
}

type filmsPage struct {
	Films  []filmCardResponseDTO `json:"films"`
	Total  int64                 `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

type detailedFilmsPage struct {
	Films  []filmResponseDTO `json:"films"`
	Total  int64             `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type filmResponse struct {
	Film filmResponseDTO `json:"film"`
}

type collectionsPage struct {
	Collections []collectionResponseDTO `json:"collections"`
	Total       int64                   `json:"total"`
	Limit       int                     `json:"limit"`
	Offset      int                     `json:"offset"`
}

type collectionPage struct {
	Collection collectionWithFilmsResponseDTO `json:"collection"`
	Total      int64                          `json:"total"`
	Limit      int                            `json:"limit"`
	Offset     int                            `json:"offset"`
}

func (h *Handler) ListFilms(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "pagination", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	result, err := h.client.ListFilms(r.Context(), limit, offset)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	if r.URL.Query().Get("details") == detailsEnabledValue {
		films := make([]filmResponseDTO, len(result.Films))
		for i, film := range result.Films {
			films[i] = toFilmResponseDTO(film)
		}

		httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, detailedFilmsPage{
			Films:  films,
			Total:  result.Total,
			Limit:  limit,
			Offset: offset,
		}, h.appLogger)

		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, filmsPage{
		Films:  toFilmCardResponseDTOs(result.Films),
		Total:  result.Total,
		Limit:  limit,
		Offset: offset,
	}, h.appLogger)
}

func (h *Handler) GetFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	result, err := h.client.GetFilm(r.Context(), id)
	if err != nil {
		writeError(r.Context(), w, fmt.Errorf("get film %d: %w", id, err), h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, filmResponse{
		Film: toFilmResponseDTO(result.Film),
	}, h.appLogger)
}

func (h *Handler) CreateFilm(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFilmRequestBodyBytes)
	input, err := decodeCreateFilm(r)

	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	created, err := h.client.CreateFilm(r.Context(), input)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, createFilmResponse{
		ID: created.GetFilmId(),
	}, h.appLogger)
}

func (h *Handler) UpdateFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFilmRequestBodyBytes)
	input, err := decodeCreateFilm(r)

	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	if _, err := h.client.UpdateFilm(r.Context(), id, input); err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	if _, err := h.client.DeleteFilm(r.Context(), id); err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListCollections(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "pagination", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	result, err := h.client.ListCollections(r.Context(), limit, offset)
	if err != nil {
		writeError(r.Context(), w, fmt.Errorf("list collections: %w", err), h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, collectionsPage{
		Collections: toCollectionResponseDTOs(result.Collections),
		Total:       result.Total,
		Limit:       limit,
		Offset:      offset,
	}, h.appLogger)
}

func (h *Handler) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidCollectionID)
		writeError(r.Context(), w, ErrInvalidCollectionID, h.appLogger)

		return
	}

	limit, offset, err := pagination(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "pagination", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	result, err := h.client.GetCollection(r.Context(), id, limit, offset)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, collectionPage{
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

func writeError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	var sizeErr *http.MaxBytesError
	if errors.As(err, &sizeErr) {
		httpresponse.WriteErrorContext(ctx, w, http.StatusRequestEntityTooLarge, "request body exceeds 1 MiB", appLogger)
		return
	}

	for _, known := range clientErrors {
		if errors.Is(err, known.err) {
			msg := known.err.Error()
			if known.status == http.StatusBadRequest {
				msg = err.Error()
			}

			httpresponse.WriteErrorContext(ctx, w, known.status, msg, appLogger)

			return
		}
	}

	if errors.Is(err, catalogclient.ErrInvalidArgument) {
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
		return
	}

	httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, "internal server error", appLogger)
}

func (h *Handler) GetFilmJSON(ctx context.Context, filmID int64) (json.RawMessage, error) {
	result, err := h.client.GetFilm(ctx, filmID)
	if err != nil {
		return nil, err
	}

	return json.Marshal(toFilmResponseDTO(result.GetFilm()))
}
