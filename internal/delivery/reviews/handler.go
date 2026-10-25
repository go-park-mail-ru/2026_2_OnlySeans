package reviews

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
)

const (
	defaultLimit  = 20
	minLimit      = 1
	maxLimit      = 100
	defaultOffset = 0
	maxBodyBytes  = 1 << 20
)

const (
	limitParam  = "limit"
	offsetParam = "offset"
	idVar       = "id"
)

const internalErrorMessage = "internal server error"

var (
	ErrInvalidID     = errors.New("id must be a positive integer")
	ErrInvalidLimit  = errors.New("invalid limit")
	ErrInvalidOffset = errors.New("offset must be a non-negative integer")
	ErrInvalidBody   = errors.New("invalid request body")
)

var clientErrors = []struct {
	err    error
	status int
}{
	{err: ErrInvalidID, status: http.StatusBadRequest},
	{err: ErrInvalidLimit, status: http.StatusBadRequest},
	{err: ErrInvalidOffset, status: http.StatusBadRequest},
	{err: ErrInvalidBody, status: http.StatusBadRequest},
	{err: domain.ErrInvalidTitle, status: http.StatusBadRequest},
	{err: domain.ErrInvalidContent, status: http.StatusBadRequest},
	{err: domain.ErrUnauthorized, status: http.StatusUnauthorized},
	{err: domain.ErrForbidden, status: http.StatusForbidden},
	{err: domain.ErrReviewNotFound, status: http.StatusNotFound},
	{err: domain.ErrFilmNotFound, status: http.StatusNotFound},
	{err: domain.ErrReviewExists, status: http.StatusConflict},
}

type HandlerUseCases struct {
	List   *usecase.ListReviewsUseCase
	Create *usecase.CreateReviewUseCase
	Update *usecase.UpdateReviewUseCase
	Delete *usecase.DeleteReviewUseCase
}

type Handler struct {
	list   *usecase.ListReviewsUseCase
	create *usecase.CreateReviewUseCase
	update *usecase.UpdateReviewUseCase
	delete *usecase.DeleteReviewUseCase
}

func NewHandler(cases HandlerUseCases) (*Handler, error) {
	if cases.List == nil {
		return nil, errors.New("list reviews use case is nil")
	}
	if cases.Create == nil {
		return nil, errors.New("create review use case is nil")
	}
	if cases.Update == nil {
		return nil, errors.New("update review use case is nil")
	}
	if cases.Delete == nil {
		return nil, errors.New("delete review use case is nil")
	}
	return &Handler{
		list:   cases.List,
		create: cases.Create,
		update: cases.Update,
		delete: cases.Delete,
	}, nil
}

type reviewRequest struct {
	Title            string `json:"title"`
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"contains_spoilers"`
}

// ListByFilm handles GET /api/films/{id}/reviews.
func (h *Handler) ListByFilm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.list.Run(r.Context(), usecase.NewListReviewsInput(domain.FilmID(id), limit, offset))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reviewsPage{
		Reviews: toReviewResponseDTOs(result.Reviews),
		Total:   result.Total,
		Limit:   limit,
		Offset:  offset,
	})
}

// Create handles POST /api/films/{id}/reviews.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	input, err := decodeReview(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.create.Run(r.Context(), usecase.NewCreateReviewInput(userID, domain.FilmID(id), input))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, reviewResponse{Review: toReviewResponseDTO(result.Review)})
}

// Update handles PUT /api/reviews/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}
	input, err := decodeReview(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	result, err := h.update.Run(r.Context(), usecase.NewUpdateReviewInput(userID, domain.ReviewID(id), input))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reviewResponse{Review: toReviewResponseDTO(result.Review)})
}

// Delete handles DELETE /api/reviews/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	if err := h.delete.Run(r.Context(), usecase.NewDeleteReviewInput(userID, domain.ReviewID(id))); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// decodeReview читает тело не больше 1 МБ и не пропускает неизвестные поля,
// например попытку подставить author_id.
func decodeReview(w http.ResponseWriter, r *http.Request) (domain.ReviewInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req reviewRequest
	if err := decoder.Decode(&req); err != nil {
		return domain.ReviewInput{}, ErrInvalidBody
	}
	return domain.ReviewInput{
		Title:            req.Title,
		Content:          req.Content,
		ContainsSpoilers: req.ContainsSpoilers,
	}, nil
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
		log.Printf("reviews: write response: %v", err)
	}
}

// writeError отвечает статусом по таблице clientErrors. Для 400 клиент видит
// подробный текст, для остальных — текст самой ошибки без внутренних обёрток.
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

	log.Printf("reviews: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: internalErrorMessage})
}
