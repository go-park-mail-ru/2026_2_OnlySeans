package ratings_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	ratingsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	ratingsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/ratings"
	ratingsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
)

func newRatingsRouter(t *testing.T) http.Handler {
	t.Helper()

	films, err := filmsservice.NewSeededService()
	require.NoError(t, err)
	repo := ratingsservice.NewInMemoryRepo()
	set, err := ratingsusecase.NewSetFilmRatingUseCase(repo, films)
	require.NoError(t, err)
	get, err := ratingsusecase.NewGetFilmRatingUseCase(repo, films)
	require.NoError(t, err)
	getForCollection, err := ratingsusecase.NewGetCollectionRatingsUseCase(repo, films)
	require.NoError(t, err)
	deleteRating, err := ratingsusecase.NewDeleteFilmRatingUseCase(repo, films)
	require.NoError(t, err)
	handler, err := ratingsdelivery.NewHandler(ratingsdelivery.HandlerUseCases{
		SetRating:               set,
		GetRating:               get,
		GetRatingsForCollection: getForCollection,
		DeleteRating:            deleteRating,
	})
	require.NoError(t, err)

	r := mux.NewRouter()
	r.HandleFunc("/api/films/{id}/rating", handler.Get).Methods(http.MethodGet)
	r.HandleFunc("/api/films/{id}/rating", handler.Set).Methods(http.MethodPut)
	r.HandleFunc("/api/films/{id}/rating", handler.Delete).Methods(http.MethodDelete)
	r.HandleFunc("/api/collections/{id}/ratings", handler.GetForCollection).Methods(http.MethodGet)
	return r
}

func ratingRequest(handler http.Handler, method, path, body string, userID domain.UserID) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID > 0 {
		req = req.WithContext(authdelivery.ContextWithUserID(req.Context(), userID))
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestRatingHandler_SetAndGet(t *testing.T) {
	handler := newRatingsRouter(t)

	rec := ratingRequest(handler, http.MethodPut, "/api/films/1/rating", `{"score":8}`, 7)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var setResponse struct {
		Rating struct {
			FilmID domain.FilmID `json:"film_id"`
			Score  int16         `json:"score"`
		} `json:"rating"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &setResponse))
	require.Equal(t, domain.FilmID(1), setResponse.Rating.FilmID)
	require.Equal(t, int16(8), setResponse.Rating.Score)

	rec = ratingRequest(handler, http.MethodGet, "/api/films/1/rating", "", 0)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var summaryResponse struct {
		Rating struct {
			FilmID       domain.FilmID `json:"film_id"`
			AverageScore float64       `json:"average_score"`
			Votes        int64         `json:"votes"`
		} `json:"rating"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summaryResponse))
	require.Equal(t, domain.FilmID(1), summaryResponse.Rating.FilmID)
	require.Equal(t, float64(8), summaryResponse.Rating.AverageScore)
	require.Equal(t, int64(1), summaryResponse.Rating.Votes)

	rec = ratingRequest(handler, http.MethodPut, "/api/films/1/rating", `{"score":10}`, 7)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	rec = ratingRequest(handler, http.MethodGet, "/api/films/1/rating", "", 0)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summaryResponse))
	require.Equal(t, float64(10), summaryResponse.Rating.AverageScore)
	require.Equal(t, int64(1), summaryResponse.Rating.Votes)
}

func TestRatingHandler_RejectsInvalidRequests(t *testing.T) {
	handler := newRatingsRouter(t)

	tests := []struct {
		name   string
		path   string
		body   string
		userID domain.UserID
		status int
	}{
		{"invalid film id", "/api/films/0/rating", `{"score":5}`, 7, http.StatusBadRequest},
		{"missing auth context", "/api/films/1/rating", `{"score":5}`, 0, http.StatusUnauthorized},
		{"score too small", "/api/films/1/rating", `{"score":0}`, 7, http.StatusBadRequest},
		{"score too large", "/api/films/1/rating", `{"score":11}`, 7, http.StatusBadRequest},
		{"unknown field", "/api/films/1/rating", `{"score":5,"account_id":7}`, 7, http.StatusBadRequest},
		{"unknown film", "/api/films/999/rating", `{"score":5}`, 7, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := ratingRequest(handler, http.MethodPut, tt.path, tt.body, tt.userID)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
		})
	}
}

func TestRatingHandler_GetUnknownFilm(t *testing.T) {
	handler := newRatingsRouter(t)
	rec := ratingRequest(handler, http.MethodGet, "/api/films/999/rating", "", 0)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestRatingHandler_GetCollectionRatings(t *testing.T) {
	handler := newRatingsRouter(t)
	for _, rating := range []struct {
		filmID domain.FilmID
		userID domain.UserID
		score  int
	}{
		{filmID: 1, userID: 7, score: 8},
		{filmID: 1, userID: 8, score: 9},
		{filmID: 2, userID: 7, score: 6},
	} {
		rec := ratingRequest(handler, http.MethodPut,
			"/api/films/"+strconv.FormatInt(int64(rating.filmID), 10)+"/rating",
			`{"score":`+strconv.Itoa(rating.score)+`}`,
			rating.userID,
		)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}

	rec := ratingRequest(handler, http.MethodGet, "/api/collections/1/ratings", "", 0)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var response struct {
		Ratings []struct {
			FilmID       domain.FilmID `json:"film_id"`
			AverageScore float64       `json:"average_score"`
			Votes        int64         `json:"votes"`
		} `json:"ratings"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Ratings, 6)
	require.Equal(t, domain.FilmID(1), response.Ratings[0].FilmID)
	require.Equal(t, float64(8.5), response.Ratings[0].AverageScore)
	require.Equal(t, int64(2), response.Ratings[0].Votes)
	require.Equal(t, domain.FilmID(2), response.Ratings[1].FilmID)
	require.Equal(t, float64(6), response.Ratings[1].AverageScore)
	require.Equal(t, int64(1), response.Ratings[1].Votes)
	require.Equal(t, domain.FilmID(3), response.Ratings[2].FilmID)
	require.Equal(t, float64(0), response.Ratings[2].AverageScore)
	require.Equal(t, int64(0), response.Ratings[2].Votes)
}

func TestRatingHandler_GetCollectionRatingsErrors(t *testing.T) {
	handler := newRatingsRouter(t)
	tests := []struct {
		path   string
		status int
	}{
		{path: "/api/collections/abc/ratings", status: http.StatusBadRequest},
		{path: "/api/collections/0/ratings", status: http.StatusBadRequest},
		{path: "/api/collections/999/ratings", status: http.StatusNotFound},
	}
	for _, tt := range tests {
		rec := ratingRequest(handler, http.MethodGet, tt.path, "", 0)
		require.Equal(t, tt.status, rec.Code, rec.Body.String())
	}
}

func TestRatingHandler_Delete(t *testing.T) {
	handler := newRatingsRouter(t)

	rec := ratingRequest(handler, http.MethodPut, "/api/films/1/rating", `{"score":8}`, 7)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = ratingRequest(handler, http.MethodDelete, "/api/films/1/rating", "", 7)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.Empty(t, rec.Body.String())

	rec = ratingRequest(handler, http.MethodGet, "/api/films/1/rating", "", 0)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var summaryResponse struct {
		Rating struct {
			AverageScore float64 `json:"average_score"`
			Votes        int64   `json:"votes"`
		} `json:"rating"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summaryResponse))
	require.Equal(t, float64(0), summaryResponse.Rating.AverageScore)
	require.Equal(t, int64(0), summaryResponse.Rating.Votes)

	rec = ratingRequest(handler, http.MethodDelete, "/api/films/1/rating", "", 7)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestRatingHandler_DeleteErrors(t *testing.T) {
	handler := newRatingsRouter(t)
	tests := []struct {
		name   string
		path   string
		userID domain.UserID
		status int
	}{
		{name: "invalid film id", path: "/api/films/0/rating", userID: 7, status: http.StatusBadRequest},
		{name: "missing auth", path: "/api/films/1/rating", status: http.StatusUnauthorized},
		{name: "unknown film", path: "/api/films/999/rating", userID: 7, status: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := ratingRequest(handler, http.MethodDelete, tt.path, "", tt.userID)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
		})
	}
}
