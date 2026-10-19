package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reviewBody = `{"title":"Отлично","content":"Посмотрела фильм два раза, и второй раз понравился даже больше первого.","contains_spoilers":false}`

func TestRouter_ReviewsJourney(t *testing.T) {
	client := newTestClient(t)

	status, body := client.do(http.MethodGet, "/api/films/1/reviews", "")
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `[]`, string(body["reviews"]))

	status, _ = client.do(http.MethodPost, "/api/films/1/reviews", reviewBody)
	assert.Equal(t, http.StatusUnauthorized, status, "без входа отзыв не создаётся")

	status, _ = client.do(http.MethodPost, "/api/register", `{"email":"anna@example.com","username":"anna","password":"Password1"}`)
	require.Equal(t, http.StatusCreated, status)

	status, body = client.do(http.MethodPost, "/api/films/1/reviews", reviewBody)
	require.Equal(t, http.StatusCreated, status)
	var review struct {
		ID       int `json:"id"`
		AuthorID int `json:"author_id"`
		FilmID   int `json:"film_id"`
	}
	require.NoError(t, json.Unmarshal(body["review"], &review))
	assert.Equal(t, 1, review.ID)
	assert.Equal(t, 1, review.AuthorID, "автор берётся из сессии")
	assert.Equal(t, 1, review.FilmID)

	status, _ = client.do(http.MethodPost, "/api/films/1/reviews", reviewBody)
	assert.Equal(t, http.StatusConflict, status)

	status, _ = client.do(http.MethodPost, "/api/films/999/reviews", reviewBody)
	assert.Equal(t, http.StatusNotFound, status)

	// Второй пользователь: регистрация заменяет cookie сессии в клиенте.
	status, _ = client.do(http.MethodPost, "/api/register", `{"email":"boris@example.com","username":"boris","password":"Password1"}`)
	require.Equal(t, http.StatusCreated, status)

	status, _ = client.do(http.MethodPut, "/api/reviews/1", reviewBody)
	assert.Equal(t, http.StatusForbidden, status, "чужой отзыв не меняется")
	status, _ = client.do(http.MethodDelete, "/api/reviews/1", "")
	assert.Equal(t, http.StatusForbidden, status, "чужой отзыв не удаляется")

	status, _ = client.do(http.MethodPost, "/api/login", `{"email":"anna@example.com","password":"Password1"}`)
	require.Equal(t, http.StatusOK, status)

	status, _ = client.do(http.MethodPut, "/api/reviews/1", `{"title":"Передумала","content":"Посмотрела фильм два раза, и второй раз понравился даже больше первого.","contains_spoilers":true}`)
	assert.Equal(t, http.StatusOK, status)

	status, body = client.do(http.MethodGet, "/api/films/1/reviews", "")
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `1`, string(body["total"]))

	status, _ = client.do(http.MethodDelete, "/api/reviews/1", "")
	assert.Equal(t, http.StatusNoContent, status)

	status, _ = client.do(http.MethodPost, "/api/logout", "")
	require.Equal(t, http.StatusNoContent, status)

	status, _ = client.do(http.MethodDelete, "/api/reviews/1", "")
	assert.Equal(t, http.StatusUnauthorized, status, "после выхода удалять нельзя")
}

func TestRouter_ReviewsMethodNotAllowed(t *testing.T) {
	client := newTestClient(t)

	status, _ := client.do(http.MethodDelete, "/api/films/1/reviews", "")
	assert.Equal(t, http.StatusMethodNotAllowed, status)
	status, _ = client.do(http.MethodGet, "/api/reviews/1", "")
	assert.Equal(t, http.StatusMethodNotAllowed, status)
}

func TestRouter_ReviewsPreflight(t *testing.T) {
	handler := newTestRouter(t)

	tests := []struct {
		path   string
		method string
	}{
		{"/api/films/1/reviews", http.MethodPost},
		{"/api/reviews/1", http.MethodPut},
		{"/api/reviews/1", http.MethodDelete},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", tt.method)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code, tt.path)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), tt.method, tt.path)
	}
}
