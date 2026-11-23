package reviews

import (
	"context"

	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/gorilla/mux"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	defaultLimit  = 20
	minLimit      = 1
	maxLimit      = 100
	defaultOffset = 0
	decimalBase   = 10
	int64Bits     = 64
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
	idVar       = "id"
)

var (
	ErrInvalidID     = errors.New("id must be a positive integer")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("offset must be a non-negative integer")
)

var clientErrors = []struct {
	err     error
	status  int
	message string
}{
	{
		err:     ErrInvalidID,
		status:  http.StatusBadRequest,
		message: ErrInvalidID.Error(),
	},
	{
		err:     ErrInvalidLimit,
		status:  http.StatusBadRequest,
		message: fmt.Sprintf("limit must be an integer from %d to %d", minLimit, maxLimit),
	},
	{
		err:     ErrInvalidOffset,
		status:  http.StatusBadRequest,
		message: ErrInvalidOffset.Error(),
	},
	{
		err:     httprequest.ErrInvalidBody,
		status:  http.StatusBadRequest,
		message: httprequest.ErrInvalidBody.Error(),
	},
	{
		err:     domain.ErrInvalidVersion,
		status:  http.StatusBadRequest,
		message: "version must be a positive integer",
	},
	{
		err:     domain.ErrVersionConflict,
		status:  http.StatusConflict,
		message: "review was changed by someone else, reload it",
	},
	{
		err:     domain.ErrInvalidTitle,
		status:  http.StatusBadRequest,
		message: fmt.Sprintf("title must be from %d to %d characters", domain.MinReviewTitleLen, domain.MaxReviewTitleLen),
	},
	{
		err:     domain.ErrInvalidContent,
		status:  http.StatusBadRequest,
		message: fmt.Sprintf("content must be from %d to %d characters", domain.MinReviewContentLen, domain.MaxReviewContentLen),
	},
	{
		err:     domain.ErrUnauthorized,
		status:  http.StatusUnauthorized,
		message: domain.ErrUnauthorized.Error(),
	},
	{
		err:     domain.ErrForbidden,
		status:  http.StatusForbidden,
		message: domain.ErrForbidden.Error(),
	},
	{
		err:     domain.ErrReviewNotFound,
		status:  http.StatusNotFound,
		message: domain.ErrReviewNotFound.Error(),
	},
	{
		err:     domain.ErrFilmNotFound,
		status:  http.StatusNotFound,
		message: domain.ErrFilmNotFound.Error(),
	},
	{
		err:     domain.ErrReviewExists,
		status:  http.StatusConflict,
		message: domain.ErrReviewExists.Error(),
	},
}

type Handler struct {
	operations Operations
	appLogger  logger.Logger
}

func NewHandlerWithOperations(operations Operations, appLogger logger.Logger) (*Handler, error) {
	if operations.List == nil || operations.Create == nil || operations.Update == nil || operations.Delete == nil {
		return nil, errors.New("reviews operations are required")
	}
	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}
	return &Handler{operations: operations, appLogger: appLogger}, nil
}

type reviewRequest struct {
	Title            string `json:"title"`
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"contains_spoilers"`
	Version          int    `json:"version"`
}

func (h *Handler) ListFilmReviews(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidID)
		writeError(r.Context(), w, ErrInvalidID, h.appLogger)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "pagination", err)
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	reviews, total, err := h.operations.List(r.Context(), domain.FilmID(id), limit, offset)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, reviewsPage{
		Reviews: toReviewResponseDTOs(reviews),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, h.appLogger)
}

func (h *Handler) CreateReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidID)
		writeError(r.Context(), w, ErrInvalidID, h.appLogger)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "session", domain.ErrUnauthorized)
		writeError(r.Context(), w, domain.ErrUnauthorized, h.appLogger)
		return
	}
	input, _, err := decodeReview(w, r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	reviewID, err := h.operations.Create(r.Context(), userID, domain.FilmID(id), input)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, createReviewResponse{ID: reviewID}, h.appLogger)
}

func (h *Handler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidID)
		writeError(r.Context(), w, ErrInvalidID, h.appLogger)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "session", domain.ErrUnauthorized)
		writeError(r.Context(), w, domain.ErrUnauthorized, h.appLogger)
		return
	}
	input, version, err := decodeReview(w, r)
	if err != nil {
		httprequest.LogRejected(h.appLogger, r, "body", err)
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	review, err := h.operations.Update(r.Context(), userID, domain.ReviewID(id), input, version)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, reviewResponse{Review: toReviewResponseDTO(review)}, h.appLogger)
}

func (h *Handler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "id", ErrInvalidID)
		writeError(r.Context(), w, ErrInvalidID, h.appLogger)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		httprequest.LogRejected(h.appLogger, r, "session", domain.ErrUnauthorized)
		writeError(r.Context(), w, domain.ErrUnauthorized, h.appLogger)
		return
	}

	err := h.operations.Delete(r.Context(), userID, domain.ReviewID(id))
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func decodeReview(w http.ResponseWriter, r *http.Request) (domain.ReviewInput, int, error) {
	var req reviewRequest
	if err := httprequest.DecodeJSON(w, r, &req); err != nil {
		return domain.ReviewInput{}, 0, err
	}
	return domain.ReviewInput{
		Title:            req.Title,
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}, req.Version, nil
}

func pagination(r *http.Request) (limit, offset int, err error) {
	limit, ok := queryInt(r, limitParam, defaultLimit)
	if !ok || limit < minLimit || limit > maxLimit {
		return 0, 0, ErrInvalidLimit
	}

	offset, ok = queryInt(r, offsetParam, defaultOffset)
	if !ok || offset < 0 {
		return 0, 0, ErrInvalidOffset
	}

	return limit, offset, nil
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)[idVar], decimalBase, int64Bits)
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
	for _, known := range clientErrors {
		if errors.Is(err, known.err) {
			httpresponse.WriteErrorContext(ctx, w, known.status, known.message, appLogger)
			return
		}
	}

	httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
}
