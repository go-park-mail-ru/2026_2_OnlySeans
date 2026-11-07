package ratings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
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

type HandlerUseCases struct {
	SetRating               *usecase.SetFilmRatingUseCase
	GetRating               *usecase.GetFilmRatingUseCase
	GetRatingsForCollection *usecase.GetCollectionRatingsUseCase
	DeleteRating            *usecase.DeleteFilmRatingUseCase
}

type Handler struct {
	useCases  HandlerUseCases
	appLogger logger.Logger
}

func NewHandler(cases HandlerUseCases, appLogger logger.Logger) (*Handler, error) {
	if cases.SetRating == nil {
		return nil, errors.New("set film rating use case is nil")
	}
	if cases.GetRating == nil {
		return nil, errors.New("get film rating use case is nil")
	}
	if cases.GetRatingsForCollection == nil {
		return nil, errors.New("get collection ratings use case is nil")
	}
	if cases.DeleteRating == nil {
		return nil, errors.New("delete film rating use case is nil")
	}
	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}
	return &Handler{
		useCases:  cases,
		appLogger: appLogger,
	}, nil
}

func (h *Handler) SetFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}
	accountID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized, h.appLogger)
		return
	}

	score, err := decodeScore(w, r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	result, err := h.useCases.SetRating.Run(r.Context(), *usecase.NewSetFilmRatingInput(accountID, domain.FilmID(filmID), score))
	if err != nil {
		writeError(w, fmt.Errorf("set rating for film %d: %w", filmID, err), h.appLogger)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, ratingResponse{Rating: toFilmRatingResponseDTO(result.Rating)}, h.appLogger)
}

func (h *Handler) GetFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}

	result, err := h.useCases.GetRating.Run(r.Context(), *usecase.NewGetFilmRatingInput(domain.FilmID(filmID)))
	if err != nil {
		writeError(w, fmt.Errorf("get rating for film %d: %w", filmID, err), h.appLogger)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, ratingSummaryResponse{Rating: toFilmRatingSummaryResponseDTO(result.Summary)}, h.appLogger)
}

func (h *Handler) GetRatingsForCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidCollectionID, h.appLogger)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err, h.appLogger)
		return
	}

	result, err := h.useCases.GetRatingsForCollection.Run(r.Context(), *usecase.NewGetCollectionRatingsInput(domain.CollectionID(collectionID), limit, offset))
	if err != nil {
		writeError(w, fmt.Errorf("get ratings for collection %d: %w", collectionID, err), h.appLogger)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, collectionRatingsResponse{
		Ratings: toFilmRatingSummaryResponseDTOs(result.Ratings),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	}, h.appLogger)
}

func (h *Handler) DeleteFilmRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.appLogger)
		return
	}

	accountID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized, h.appLogger)
		return
	}

	if err := h.useCases.DeleteRating.Run(r.Context(), *usecase.NewDeleteFilmRatingInput(accountID, domain.FilmID(filmID))); err != nil {
		writeError(w, fmt.Errorf("delete rating for film %d: %w", filmID, err), h.appLogger)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeScore(w http.ResponseWriter, r *http.Request) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request setFilmRatingRequest
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

func writeError(w http.ResponseWriter, err error, appLogger logger.Logger) {
	switch {
	case errors.Is(err, ErrInvalidFilmID):
		httpresponse.WriteError(w, http.StatusBadRequest, ErrInvalidFilmID.Error(), appLogger)
	case errors.Is(err, ErrInvalidCollectionID):
		httpresponse.WriteError(w, http.StatusBadRequest, ErrInvalidCollectionID.Error(), appLogger)
	case errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidOffset):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, ErrInvalidBody):
		httpresponse.WriteError(w, http.StatusBadRequest, ErrInvalidBody.Error(), appLogger)
	case errors.Is(err, domain.ErrInvalidFilmRating):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), appLogger)
	case errors.Is(err, domain.ErrUnauthorized):
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), appLogger)
	case errors.Is(err, domain.ErrFilmNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, domain.ErrFilmNotFound.Error(), appLogger)
	case errors.Is(err, domain.ErrCollectionNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, domain.ErrCollectionNotFound.Error(), appLogger)
	case errors.Is(err, domain.ErrRatingNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, domain.ErrRatingNotFound.Error(), appLogger)
	default:
		appLogger.Error("unexpected error", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, internalErrMsg, appLogger)
	}
}
