package ratings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
)

const (
	idVar          = "id"
	maxBodyBytes   = 1 << 20
	internalErrMsg = "internal server error"
)

var (
	ErrInvalidFilmID       = errors.New("film id must be a positive integer")
	ErrInvalidCollectionID = errors.New("collection id must be a positive integer")
	ErrInvalidBody         = errors.New("invalid request body")
)

type HandlerUseCases struct {
	SetRating               *usecase.SetFilmRatingUseCase
	GetRating               *usecase.GetFilmRatingUseCase
	GetRatingsForCollection *usecase.GetCollectionRatingsUseCase
	DeleteRating            *usecase.DeleteFilmRatingUseCase
}

type Handler struct {
	setRating               *usecase.SetFilmRatingUseCase
	getRating               *usecase.GetFilmRatingUseCase
	getRatingsForCollection *usecase.GetCollectionRatingsUseCase
	deleteRating            *usecase.DeleteFilmRatingUseCase
}

func NewHandler(cases HandlerUseCases) (*Handler, error) {
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
	return &Handler{
		setRating:               cases.SetRating,
		getRating:               cases.GetRating,
		getRatingsForCollection: cases.GetRatingsForCollection,
		deleteRating:            cases.DeleteRating,
	}, nil
}

type setFilmRatingRequest struct {
	Score *int16 `json:"score"`
}

func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID)
		return
	}
	accountID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	score, err := decodeScore(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.setRating.Run(r.Context(), usecase.NewSetFilmRatingInput(accountID, domain.FilmID(filmID), score))
	if err != nil {
		writeError(w, fmt.Errorf("set rating for film %d: %w", filmID, err))
		return
	}
	writeJSON(w, http.StatusOK, ratingResponse{Rating: toFilmRatingResponseDTO(result.Rating)})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID)
		return
	}

	result, err := h.getRating.Run(r.Context(), usecase.NewGetFilmRatingInput(domain.FilmID(filmID)))
	if err != nil {
		writeError(w, fmt.Errorf("get rating for film %d: %w", filmID, err))
		return
	}
	writeJSON(w, http.StatusOK, ratingSummaryResponse{Rating: toFilmRatingSummaryResponseDTO(result.Summary)})
}

func (h *Handler) GetForCollection(w http.ResponseWriter, r *http.Request) {
	collectionID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidCollectionID)
		return
	}

	result, err := h.getRatingsForCollection.Run(r.Context(), usecase.NewGetCollectionRatingsInput(domain.CollectionID(collectionID)))
	if err != nil {
		writeError(w, fmt.Errorf("get ratings for collection %d: %w", collectionID, err))
		return
	}
	writeJSON(w, http.StatusOK, collectionRatingsResponse{
		Ratings: toFilmRatingSummaryResponseDTOs(result.Ratings),
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID)
		return
	}

	accountID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.deleteRating.Run(r.Context(), usecase.NewDeleteFilmRatingInput(accountID, domain.FilmID(filmID))); err != nil {
		writeError(w, fmt.Errorf("delete rating for film %d: %w", filmID, err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeScore(w http.ResponseWriter, r *http.Request) (int16, error) {
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("ratings: write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidFilmID):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrInvalidFilmID.Error()})
	case errors.Is(err, ErrInvalidCollectionID):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrInvalidCollectionID.Error()})
	case errors.Is(err, ErrInvalidBody):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ErrInvalidBody.Error()})
	case errors.Is(err, domain.ErrInvalidFilmRating):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: domain.ErrInvalidFilmRating.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: domain.ErrUnauthorized.Error()})
	case errors.Is(err, domain.ErrFilmNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: domain.ErrFilmNotFound.Error()})
	case errors.Is(err, domain.ErrCollectionNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: domain.ErrCollectionNotFound.Error()})
	case errors.Is(err, domain.ErrRatingNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: domain.ErrRatingNotFound.Error()})
	default:
		log.Printf("ratings: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: internalErrMsg})
	}
}
