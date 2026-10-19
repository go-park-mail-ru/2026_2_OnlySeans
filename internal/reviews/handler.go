//что прислали и что ответить

package reviews

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
	"github.com/gorilla/mux"
)

type ReviewUsecase interface {
	Create(ctx context.Context, authorID auth.UserID, filmID entities.FilmID, input ReviewInput) (entities.Review, error)
	ListByFilm(ctx context.Context, filmID entities.FilmID, limit, offset int) ([]entities.Review, int, error)
	Update(ctx context.Context, userID auth.UserID, id entities.ReviewID, input ReviewInput) (entities.Review, error)
	Delete(ctx context.Context, userID auth.UserID, id entities.ReviewID) error
}

var _ ReviewUsecase = (*Usecase)(nil) //проверка совпадения

type Handler struct {
	usecase ReviewUsecase
}

func NewHandler(usecase ReviewUsecase) (*Handler, error) {

	if usecase == nil {
		return nil, ErrNilUsecase
	}
	return &Handler{usecase: usecase}, nil
}

const (
	defaultLimit = 20
	minLimit     = 1
	maxLimit     = 100
	maxBodyBytes = 1 << 20 //1МБ
)

// запрос
type reviewRequest struct {
	Title            string `json:"title"`
	Content          string `json:"content"`
	ContainsSpoilers bool   `json:"contains_spoilers"`
}

// ответы
// один отзыв для создания и изменения
type reviewResponse struct {
	Review entities.Review `json:"review"`
}

// страница списка
type reviewsPage struct {
	Reviews []entities.Review `json:"reviews"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// отправляем ответ с нужным статусом
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status) //код ответа

	if err := json.NewEncoder(w).Encode(body); err != nil { //создать кодировщик(пишет в ответ), превратить структуру в джисон и записать
		log.Printf("reviews: write response: %v", err)
	}
}

var clientErrors = []struct {
	err    error
	status int
}{
	{err: ErrInvalidID, status: http.StatusBadRequest},
	{err: ErrInvalidLimit, status: http.StatusBadRequest},
	{err: ErrInvalidOffset, status: http.StatusBadRequest},
	{err: ErrInvalidBody, status: http.StatusBadRequest},
	{err: ErrInvalidTitle, status: http.StatusBadRequest},
	{err: ErrInvalidContent, status: http.StatusBadRequest},
	{err: auth.ErrUnauthorized, status: http.StatusUnauthorized},
	{err: ErrForbidden, status: http.StatusForbidden},
	{err: ErrReviewNotFound, status: http.StatusNotFound},
	{err: films.ErrFilmNotFound, status: http.StatusNotFound},
	{err: ErrReviewExists, status: http.StatusConflict},
}

func writeError(w http.ResponseWriter, err error) {
	for _, known := range clientErrors {
		if errors.Is(err, known.err) {
			writeJSON(w, known.status, errorResponse{Error: err.Error()})
			return
		}
	}
	log.Printf("reviews: %v", err)
	writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
}

// номер из адреса и проверка что норм
func pathID(r *http.Request) (int64, bool) {
	raw := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(raw, 10, 64)
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

// limit offset
func pagination(r *http.Request) (int, int, error) {
	limit, ok := queryInt(r, "limit", defaultLimit)
	if !ok || limit < minLimit || limit > maxLimit {
		return 0, 0, ErrInvalidLimit
	}
	offset, ok := queryInt(r, "offset", 0)
	if !ok || offset < 0 {
		return 0, 0, ErrInvalidOffset
	}
	return limit, offset, nil
}

// превращаем тело http запрос из т json в структуру, умеющую работать с usecase
func decodeInput(w http.ResponseWriter, r *http.Request) (ReviewInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes) //ограничение размера
	decoder := json.NewDecoder(r.Body)                    //читает json из тела
	decoder.DisallowUnknownFields()                       //ключ которого нет - ошибка
	var req reviewRequest                                 //куда читаем
	if err := decoder.Decode(&req); err != nil {
		return ReviewInput{}, ErrInvalidBody
	}
	return ReviewInput{Title: req.Title, Content: req.Content, ContainsSpoilers: req.ContainsSpoilers}, nil
}

func (h *Handler) ListByFilm(w http.ResponseWriter, r *http.Request) {
	//номер фильма из адресса
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}

	//параметры страницы
	limit, offset, err := pagination(r)
	if err != nil {
		writeError(w, err)
		return
	}

	//вызов usecase
	reviews, total, err := h.usecase.ListByFilm(r.Context(), entities.FilmID(id), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reviewsPage{Reviews: reviews, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}

	//пользователь
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, auth.ErrUnauthorized)
		return
	}

	//тело запроса
	input, err := decodeInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	review, err := h.usecase.Create(r.Context(), userID, entities.FilmID(id), input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, reviewResponse{Review: review})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {

	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, auth.ErrUnauthorized)
		return
	}

	input, err := decodeInput(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	review, err := h.usecase.Update(r.Context(), userID, entities.ReviewID(id), input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, reviewResponse{Review: review})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {

	id, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidID)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, auth.ErrUnauthorized)
		return
	}

	err := h.usecase.Delete(r.Context(), userID, entities.ReviewID(id))
	if err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
