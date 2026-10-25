package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/reviews"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
)

func newTestReviewsHandler(t *testing.T, films domain.FilmRepository) *reviewsdelivery.Handler {
	t.Helper()

	repo := reviewsservice.NewInMemoryReviewRepo()
	list, err := reviewsusecase.NewListReviewsUseCase(repo, films)
	require.NoError(t, err)
	create, err := reviewsusecase.NewCreateReviewUseCase(repo, films)
	require.NoError(t, err)
	update, err := reviewsusecase.NewUpdateReviewUseCase(repo)
	require.NoError(t, err)
	del, err := reviewsusecase.NewDeleteReviewUseCase(repo)
	require.NoError(t, err)

	handler, err := reviewsdelivery.NewHandler(reviewsdelivery.HandlerUseCases{
		List:   list,
		Create: create,
		Update: update,
		Delete: del,
	})
	require.NoError(t, err)
	return handler
}

// testClient ходит в настоящий HTTP-сервер с роутером и хранит cookie сессии.
type testClient struct {
	t       *testing.T
	baseURL string
	http    *http.Client
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()

	server := httptest.NewServer(newTestRouter(t))
	t.Cleanup(server.Close)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &testClient{t: t, baseURL: server.URL, http: &http.Client{Jar: jar}}
}

func (c *testClient) do(method, path, body string) (int, map[string]json.RawMessage) {
	c.t.Helper()

	req, err := http.NewRequest(method, c.baseURL+path, strings.NewReader(body))
	require.NoError(c.t, err)
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer resp.Body.Close()

	var parsed map[string]json.RawMessage
	if resp.StatusCode != http.StatusNoContent && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		require.NoError(c.t, json.NewDecoder(resp.Body).Decode(&parsed))
	}
	return resp.StatusCode, parsed
}

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
