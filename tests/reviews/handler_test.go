package reviews_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

const (
	anna  domain.UserID = 7
	boris domain.UserID = 8
	guest domain.UserID = 0
)

var (
	errTitleMessage   = errors.New("title must be from 3 to 255 characters")
	errContentMessage = errors.New("content must be from 50 to 20000 characters")
	errLimitMessage   = errors.New("limit must be an integer from 1 to 100")
)

// reviewJSON — отзыв в том виде, в каком его получает фронтенд.
type reviewJSON struct {
	ID               domain.ReviewID `json:"id"`
	AuthorID         domain.UserID   `json:"author_id"`
	FilmID           domain.FilmID   `json:"film_id"`
	Title            string          `json:"title"`
	Content          string          `json:"content"`
	ContainsSpoilers bool            `json:"contains_spoilers"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type reviewResponse struct {
	Review reviewJSON `json:"review"`
}

type reviewsPage struct {
	Reviews []reviewJSON `json:"reviews"`
	Total   int          `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newTestRouter(t *testing.T, films domain.FilmRepository) *mux.Router {
	t.Helper()

	cases := newReviewCases(t, films)
	h, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   cases.list,
		Create: cases.create,
		Update: cases.update,
		Delete: cases.delete,
	})
	require.NoError(t, err)

	router := mux.NewRouter()
	router.HandleFunc("/api/films/{id}/reviews", h.ListByFilm).Methods(http.MethodGet)
	router.HandleFunc("/api/films/{id}/reviews", h.Create).Methods(http.MethodPost)
	router.HandleFunc("/api/reviews/{id}", h.Update).Methods(http.MethodPut)
	router.HandleFunc("/api/reviews/{id}", h.Delete).Methods(http.MethodDelete)
	return router
}

// serve выполняет запрос от имени user; guest — без пользователя в контексте,
// как будто middleware RequireAuth не пропустил бы запрос.
func serve(router http.Handler, method, target, body string, user domain.UserID) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if user != guest {
		req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), user))
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "invalid JSON %q", rec.Body.String())
	return body
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, want error) {
	t.Helper()

	assert.Equal(t, status, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	got := decode[errorResponse](t, rec).Error
	assert.True(t, strings.HasPrefix(got, want.Error()), "error = %q, want prefix %q", got, want.Error())
}

func reviewBody(title, content string, spoilers bool) string {
	body, _ := json.Marshal(map[string]any{
		"title":             title,
		"content":           content,
		"contains_spoilers": spoilers,
	})
	return string(body)
}

var validBody = reviewBody(validTitle, validContent, false)

func filmReviewsPath(film domain.FilmID) string {
	return fmt.Sprintf("/api/films/%d/reviews", film)
}

func reviewPath(id domain.ReviewID) string {
	return fmt.Sprintf("/api/reviews/%d", id)
}

// createReview создаёт отзыв через ручку и возвращает его.
func createReview(t *testing.T, router http.Handler, user domain.UserID) reviewJSON {
	t.Helper()

	rec := serve(router, http.MethodPost, filmReviewsPath(existingFilm), validBody, user)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decode[reviewResponse](t, rec).Review
}

func silenceLog(t *testing.T) {
	t.Helper()

	previous := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(previous) })
}

func TestNewHandler_NilUseCases(t *testing.T) {
	_, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{})
	assert.Error(t, err)
}

func TestHandler_Create(t *testing.T) {
	router := newTestRouter(t, fakeFilms{})

	rec := serve(router, http.MethodPost, filmReviewsPath(existingFilm), reviewBody(validTitle, validContent, true), anna)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

	review := decode[reviewResponse](t, rec).Review
	assert.Equal(t, domain.ReviewID(1), review.ID)
	assert.Equal(t, anna, review.AuthorID)
	assert.Equal(t, existingFilm, review.FilmID)
	assert.Equal(t, validTitle, review.Title)
	assert.Equal(t, validContent, review.Content)
	assert.True(t, review.ContainsSpoilers)
	assert.False(t, review.CreatedAt.IsZero())
}

func TestHandler_CreateErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		body    string
		user    domain.UserID
		status  int
		wantErr error
	}{
		{"id фильма не число", "/api/films/abc/reviews", validBody, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"id фильма ноль", "/api/films/0/reviews", validBody, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"без пользователя", filmReviewsPath(existingFilm), validBody, guest, http.StatusUnauthorized, domain.ErrUnauthorized},
		{"сломанный JSON", filmReviewsPath(existingFilm), `{"title":`, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody},
		{"пустое тело", filmReviewsPath(existingFilm), "", anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody},
		{"строка вместо флага", filmReviewsPath(existingFilm), `{"title":"Отлично","content":"x","contains_spoilers":"да"}`, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody},
		{"попытка подставить автора", filmReviewsPath(existingFilm), `{"title":"Отлично","content":"x","author_id":99}`, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody},
		{"короткий заголовок", filmReviewsPath(existingFilm), reviewBody("ab", validContent, false), anna, http.StatusBadRequest, errTitleMessage},
		{"короткий текст", filmReviewsPath(existingFilm), reviewBody(validTitle, "мало", false), anna, http.StatusBadRequest, errContentMessage},
		{"несуществующий фильм", filmReviewsPath(missingFilm), validBody, anna, http.StatusNotFound, domain.ErrFilmNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(newTestRouter(t, fakeFilms{}), http.MethodPost, tt.target, tt.body, tt.user)
			assertError(t, rec, tt.status, tt.wantErr)
		})
	}
}

func TestHandler_CreateTwice(t *testing.T) {
	router := newTestRouter(t, fakeFilms{})
	createReview(t, router, anna)

	rec := serve(router, http.MethodPost, filmReviewsPath(existingFilm), validBody, anna)
	assertError(t, rec, http.StatusConflict, domain.ErrReviewExists)
}

func TestHandler_CreateBodyTooLarge(t *testing.T) {
	huge := reviewBody(validTitle, strings.Repeat("a", 2<<20), false)

	rec := serve(newTestRouter(t, fakeFilms{}), http.MethodPost, filmReviewsPath(existingFilm), huge, anna)
	assertError(t, rec, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody)
}

func TestHandler_ListByFilm(t *testing.T) {
	router := newTestRouter(t, fakeFilms{})

	rec := serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"reviews":[]`, "пустой список должен приходить как [], а не null")

	first := createReview(t, router, anna)
	second := createReview(t, router, boris)

	rec = serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest)
	require.Equal(t, http.StatusOK, rec.Code)
	page := decode[reviewsPage](t, rec)
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, 20, page.Limit, "limit по умолчанию")
	assert.Equal(t, 0, page.Offset)
	assert.Equal(t, []domain.ReviewID{second.ID, first.ID}, jsonReviewIDs(page.Reviews), "новые отзывы первыми")

	rec = serve(router, http.MethodGet, filmReviewsPath(existingFilm)+"?limit=1&offset=1", "", guest)
	require.Equal(t, http.StatusOK, rec.Code)
	page = decode[reviewsPage](t, rec)
	assert.Equal(t, 2, page.Total)
	assert.Equal(t, 1, page.Limit)
	assert.Equal(t, 1, page.Offset)
	assert.Equal(t, []domain.ReviewID{first.ID}, jsonReviewIDs(page.Reviews))
}

func TestHandler_ListByFilmErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		status  int
		wantErr error
	}{
		{"id фильма не число", "/api/films/abc/reviews", http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"id фильма отрицательный", "/api/films/-1/reviews", http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"limit ноль", filmReviewsPath(existingFilm) + "?limit=0", http.StatusBadRequest, errLimitMessage},
		{"limit больше максимума", filmReviewsPath(existingFilm) + "?limit=101", http.StatusBadRequest, errLimitMessage},
		{"limit не число", filmReviewsPath(existingFilm) + "?limit=abc", http.StatusBadRequest, errLimitMessage},
		{"offset отрицательный", filmReviewsPath(existingFilm) + "?offset=-1", http.StatusBadRequest, reviewsdelivery.ErrInvalidOffset},
		{"offset не число", filmReviewsPath(existingFilm) + "?offset=abc", http.StatusBadRequest, reviewsdelivery.ErrInvalidOffset},
		{"несуществующий фильм", filmReviewsPath(missingFilm), http.StatusNotFound, domain.ErrFilmNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(newTestRouter(t, fakeFilms{}), http.MethodGet, tt.target, "", guest)
			assertError(t, rec, tt.status, tt.wantErr)
		})
	}
}

func TestHandler_Update(t *testing.T) {
	router := newTestRouter(t, fakeFilms{})
	created := createReview(t, router, anna)

	rec := serve(router, http.MethodPut, reviewPath(created.ID), reviewBody("Передумала", validContent, true), anna)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	updated := decode[reviewResponse](t, rec).Review
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "Передумала", updated.Title)
	assert.True(t, updated.ContainsSpoilers)

	rec = serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest)
	page := decode[reviewsPage](t, rec)
	require.Len(t, page.Reviews, 1)
	assert.Equal(t, "Передумала", page.Reviews[0].Title, "изменения видны в списке")
}

func TestHandler_UpdateErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		body    string
		user    domain.UserID
		status  int
		wantErr error
	}{
		{"id отзыва не число", "/api/reviews/abc", validBody, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"без пользователя", reviewPath(1), validBody, guest, http.StatusUnauthorized, domain.ErrUnauthorized},
		{"сломанный JSON", reviewPath(1), `{`, anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidBody},
		{"короткий текст", reviewPath(1), reviewBody(validTitle, "мало", false), anna, http.StatusBadRequest, errContentMessage},
		{"чужой отзыв", reviewPath(1), validBody, boris, http.StatusForbidden, domain.ErrForbidden},
		{"несуществующий отзыв", reviewPath(99), validBody, anna, http.StatusNotFound, domain.ErrReviewNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, fakeFilms{})
			createReview(t, router, anna)

			rec := serve(router, http.MethodPut, tt.target, tt.body, tt.user)
			assertError(t, rec, tt.status, tt.wantErr)
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	router := newTestRouter(t, fakeFilms{})
	created := createReview(t, router, anna)

	rec := serve(router, http.MethodDelete, reviewPath(created.ID), "", anna)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String(), "у ответа 204 нет тела")

	rec = serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest)
	assert.Zero(t, decode[reviewsPage](t, rec).Total)

	rec = serve(router, http.MethodPost, filmReviewsPath(existingFilm), validBody, anna)
	assert.Equal(t, http.StatusCreated, rec.Code, "после удаления можно написать отзыв заново")
}

func TestHandler_DeleteErrors(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		user    domain.UserID
		status  int
		wantErr error
	}{
		{"id отзыва не число", "/api/reviews/abc", anna, http.StatusBadRequest, reviewsdelivery.ErrInvalidID},
		{"без пользователя", reviewPath(1), guest, http.StatusUnauthorized, domain.ErrUnauthorized},
		{"чужой отзыв", reviewPath(1), boris, http.StatusForbidden, domain.ErrForbidden},
		{"несуществующий отзыв", reviewPath(99), anna, http.StatusNotFound, domain.ErrReviewNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, fakeFilms{})
			createReview(t, router, anna)

			rec := serve(router, http.MethodDelete, tt.target, "", tt.user)
			assertError(t, rec, tt.status, tt.wantErr)

			list := serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest)
			assert.Equal(t, 1, decode[reviewsPage](t, list).Total, "отзыв не должен удалиться")
		})
	}
}

func TestHandler_InternalErrorIsHidden(t *testing.T) {
	silenceLog(t)
	router := newTestRouter(t, brokenFilms{})

	for _, rec := range []*httptest.ResponseRecorder{
		serve(router, http.MethodGet, filmReviewsPath(existingFilm), "", guest),
		serve(router, http.MethodPost, filmReviewsPath(existingFilm), validBody, anna),
	} {
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "internal server error", decode[errorResponse](t, rec).Error)
		assert.NotContains(t, rec.Body.String(), errStorage.Error(), "подробности ошибки не уходят клиенту")
	}
}

func jsonReviewIDs(list []reviewJSON) []domain.ReviewID {
	ids := make([]domain.ReviewID, 0, len(list))
	for _, review := range list {
		ids = append(ids, review.ID)
	}
	return ids
}
