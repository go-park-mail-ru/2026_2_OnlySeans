package ratings

import (
	"context"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	idVar          = "id"
	maxBodyBytes   = 1 << 20
	internalErrMsg = "internal server error"
	defaultLimit   = 20
	minLimit       = 1
	maxLimit       = 100
	defaultOffset  = 0
	limitParam     = "limit"
	offsetParam    = "offset"
)

var (
	ErrInvalidFilmID       = errors.New("film id must be a positive integer")
	ErrInvalidCollectionID = errors.New("collection id must be a positive integer")
	ErrInvalidLimit        = errors.New("invalid limit")
	ErrInvalidOffset       = errors.New("offset must be a non-negative integer")
	ErrInvalidBody         = errors.New("invalid request body")
)

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

func (h *Handler) SetFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	accountID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "user_id", catalogclient.ErrUnauthorized)
		writeError(r.Context(), w, catalogclient.ErrUnauthorized, h.appLogger)

		return
	}

	score, err := decodeScore(w, r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		writeError(r.Context(), w, err, h.appLogger)

		return
	}

	result, err := h.client.SetFilmRating(r.Context(), accountID, filmID, score)
	if err != nil {
		writeError(r.Context(), w, fmt.Errorf("set rating for film %d: %w", filmID, err), h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, ratingResponse{
		Rating: toFilmRatingResponseDTO(result.Rating),
	}, h.appLogger)
}

func (h *Handler) GetFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	result, err := h.client.GetFilmRating(r.Context(), filmID)
	if err != nil {
		writeError(r.Context(), w, fmt.Errorf("get rating for film %d: %w", filmID, err), h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, ratingSummaryResponse{
		Rating: toFilmRatingSummaryResponseDTO(result.Rating),
	}, h.appLogger)
}

func (h *Handler) GetRatingsForCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, ok := pathID(r)
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

	result, err := h.client.GetCollectionRatings(r.Context(), collectionID, limit, offset)
	if err != nil {
		writeError(r.Context(), w, fmt.Errorf("get ratings for collection %d: %w", collectionID, err), h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, collectionRatingsResponse{
		Ratings: toFilmRatingSummaryResponseDTOs(result.Ratings),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	}, h.appLogger)
}

func (h *Handler) DeleteFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidFilmID)
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)

		return
	}

	accountID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "user_id", catalogclient.ErrUnauthorized)
		writeError(r.Context(), w, catalogclient.ErrUnauthorized, h.appLogger)

		return
	}

	if _, err := h.client.DeleteFilmRating(r.Context(), accountID, filmID); err != nil {
		writeError(r.Context(), w, fmt.Errorf("delete rating for film %d: %w", filmID, err), h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeScore(w http.ResponseWriter, r *http.Request) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request setFilmRatingRequestDTO
	if err := decoder.Decode(&request); err != nil || request.Score == nil {
		return 0, ErrInvalidBody
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return 0, ErrInvalidBody
	}

	return *request.Score, nil
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)[idVar], 10, 64)
	return id, err == nil && id > 0
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

func queryInt(r *http.Request, key string, fallback int) (int, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback, true
	}

	value, err := strconv.Atoi(raw)

	return value, err == nil
}

func writeError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, ErrInvalidFilmID):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, ErrInvalidFilmID.Error(), appLogger)
	case errors.Is(err, ErrInvalidCollectionID):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, ErrInvalidCollectionID.Error(), appLogger)
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidOffset):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, ErrInvalidBody):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, ErrInvalidBody.Error(), appLogger)
	case errors.Is(err, catalogclient.ErrInvalidFilmRating), errors.Is(err, catalogclient.ErrInvalidArgument):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, catalogclient.ErrUnauthorized):
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, catalogclient.ErrUnauthorized.Error(), appLogger)
	case errors.Is(err, catalogclient.ErrFilmNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusNotFound, catalogclient.ErrFilmNotFound.Error(), appLogger)
	case errors.Is(err, catalogclient.ErrCollectionNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusNotFound, catalogclient.ErrCollectionNotFound.Error(), appLogger)
	case errors.Is(err, catalogclient.ErrRatingNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusNotFound, catalogclient.ErrRatingNotFound.Error(), appLogger)
	default:
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, internalErrMsg, appLogger)
	}
}
