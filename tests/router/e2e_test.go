package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/auth"
)

type testClient struct {
	t       *testing.T
	baseURL *url.URL
	http    *http.Client
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()

	server := httptest.NewServer(newTestRouter(t))
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &testClient{t: t, baseURL: baseURL, http: &http.Client{Jar: jar}}
}

func (c *testClient) do(method, path, body string) (int, map[string]json.RawMessage) {
	c.t.Helper()

	req, err := http.NewRequest(method, c.baseURL.String()+path, strings.NewReader(body))
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

func (c *testClient) sessionID() string {
	for _, cookie := range c.http.Jar.Cookies(c.baseURL) {
		if cookie.Name == auth.SessionCookieName {
			return cookie.Value
		}
	}
	return ""
}

func TestRouter_UserJourney(t *testing.T) {
	client := newTestClient(t)

	status, _ := client.do(http.MethodPost, "/api/register", `{"email":"anna@example.com","username":"anna","password":"Password1"}`)
	require.Equal(t, http.StatusCreated, status)
	registerSession := client.sessionID()
	require.NotEmpty(t, registerSession)

	status, _ = client.do(http.MethodPost, "/api/register", `{"email":"anna@example.com","username":"clone","password":"Password1"}`)
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, registerSession, client.sessionID())

	status, body := client.do(http.MethodPost, "/api/login", `{"email":"anna@example.com","password":"WrongPass1"}`)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.JSONEq(t, `"invalid email or password"`, string(body["error"]))
	assert.Equal(t, registerSession, client.sessionID())

	status, body = client.do(http.MethodPost, "/api/login", `{"email":"anna@example.com","password":"Password1"}`)
	require.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"id":1,"email":"anna@example.com","username":"anna"}`, string(body["user"]))
	assert.NotEmpty(t, client.sessionID())
	assert.NotEqual(t, registerSession, client.sessionID())

	status, body = client.do(http.MethodGet, "/api/films?limit=2&offset=1", "")
	require.Equal(t, http.StatusOK, status)
	var films []struct {
		ID int `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body["films"], &films))
	require.Len(t, films, 2)
	assert.Equal(t, 2, films[0].ID)
	assert.Equal(t, 3, films[1].ID)

	status, _ = client.do(http.MethodGet, "/api/collections/2?limit=2", "")
	assert.Equal(t, http.StatusOK, status)

	status, _ = client.do(http.MethodPost, "/api/logout", "")
	require.Equal(t, http.StatusNoContent, status)
	assert.Empty(t, client.sessionID())
}

func TestRouter_UnknownRoutes(t *testing.T) {
	client := newTestClient(t)

	for _, path := range []string{"/", "/api", "/api/unknown", "/api/films/1/extra", "/films", "/api/collections/2/films"} {
		status, _ := client.do(http.MethodGet, path, "")
		assert.Equal(t, http.StatusNotFound, status, path)
	}
}

func TestRouter_PreflightOnEveryRoute(t *testing.T) {
	handler := newTestRouter(t)

	for _, path := range []string{"/api/register", "/api/login", "/api/logout", "/api/films", "/api/films/1", "/api/collections", "/api/collections/2"} {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code, path)
		assert.Empty(t, rec.Body.String(), path)
		assert.Equal(t, testOrigin, rec.Header().Get("Access-Control-Allow-Origin"), path)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), http.MethodPost, path)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type", path)
		assert.Equal(t, "Origin", rec.Header().Get("Vary"), path)
	}
}
