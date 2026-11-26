package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type openAPIRouteProbe struct{}

func (p openAPIRouteProbe) write(w http.ResponseWriter, r *http.Request, operation string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"operation": operation,
		"vars":      mux.Vars(r),
		"query":     r.URL.RawQuery,
		"body":      string(body),
	})
}

func (p openAPIRouteProbe) ListFilms(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "ListFilms")
}

func (p openAPIRouteProbe) GetFilm(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "GetFilm")
}

func (p openAPIRouteProbe) CreateFilm(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "CreateFilm")
}

func (p openAPIRouteProbe) UpdateFilm(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "UpdateFilm")
}

func (p openAPIRouteProbe) DeleteFilm(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "DeleteFilm")
}

func (p openAPIRouteProbe) ListCollections(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "ListCollections")
}

func (p openAPIRouteProbe) GetCollection(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "GetCollection")
}

func (p openAPIRouteProbe) Get(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "Get")
}

func (p openAPIRouteProbe) SetSaved(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "SetSaved")
}

func (p openAPIRouteProbe) Preferences(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "Preferences")
}

func (p openAPIRouteProbe) SavePreference(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "SavePreference")
}

func (p openAPIRouteProbe) GetAsset(w http.ResponseWriter, r *http.Request) {
	p.write(w, r, "GetAsset")
}

func TestOpenAPIRoutingPreservesHandlers(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		path      string
		body      string
		cookie    bool
		role      string
		omitFilms bool
		status    int
		operation string
		vars      map[string]string
		query     string
	}{
		{
			name:      "film ID is passed unchanged",
			method:    "GET",
			path:      "/api/films/42",
			status:    200,
			operation: "GetFilm",
			vars:      map[string]string{"id": "42"},
		},
		{
			name:      "invalid ID reaches original validator",
			method:    "GET",
			path:      "/api/films/not-a-number",
			status:    200,
			operation: "GetFilm",
			vars:      map[string]string{"id": "not-a-number"},
		},
		{
			name:      "overflow ID reaches original validator",
			method:    "GET",
			path:      "/api/films/999999999999999999999999999",
			status:    200,
			operation: "GetFilm",
			vars:      map[string]string{"id": "999999999999999999999999999"},
		},
		{
			name:      "invalid pagination reaches original validator",
			method:    "GET",
			path:      "/api/films?limit=bad&offset=-1&details=true",
			status:    200,
			operation: "ListFilms",
			vars:      map[string]string{},
			query:     "limit=bad&offset=-1&details=true",
		},
		{
			name:      "repeated pagination is preserved",
			method:    "GET",
			path:      "/api/films?limit=2&limit=3",
			status:    200,
			operation: "ListFilms",
			vars:      map[string]string{},
			query:     "limit=2&limit=3",
		},
		{
			name:   "anonymous write is rejected before handler",
			method: "POST",
			path:   "/api/films",
			body:   "{",
			status: 401,
		},
		{
			name:   "reader write is rejected before handler",
			method: "POST",
			path:   "/api/films",
			body:   "{",
			cookie: true,
			role:   "reader",
			status: 403,
		},
		{
			name:      "admin body is not consumed by generated server",
			method:    "POST",
			path:      "/api/films",
			body:      "{\"title\":\"Film\"}",
			cookie:    true,
			role:      "admin",
			status:    200,
			operation: "CreateFilm",
			vars:      map[string]string{},
		},
		{
			name:   "authentication precedes invalid path validation",
			method: "PUT",
			path:   "/api/films/bad",
			body:   "{}",
			status: 401,
		},
		{
			name:      "admin invalid ID reaches original handler",
			method:    "PUT",
			path:      "/api/films/bad",
			body:      "{}",
			cookie:    true,
			role:      "admin",
			status:    200,
			operation: "UpdateFilm",
			vars:      map[string]string{"id": "bad"},
		},
		{
			name:      "admin delete is routed",
			method:    "DELETE",
			path:      "/api/films/42",
			cookie:    true,
			role:      "admin",
			status:    200,
			operation: "DeleteFilm",
			vars:      map[string]string{"id": "42"},
		},
		{
			name:      "collection parameters are preserved",
			method:    "GET",
			path:      "/api/collections/3?limit=2&offset=1",
			status:    200,
			operation: "GetCollection",
			vars:      map[string]string{"id": "3"},
			query:     "limit=2&offset=1",
		},
		{
			name:   "library route remains authenticated",
			method: "GET",
			path:   "/api/me/library",
			status: 401,
		},
		{
			name:      "library is routed",
			method:    "GET",
			path:      "/api/me/library",
			cookie:    true,
			role:      "reader",
			status:    200,
			operation: "Get",
			vars:      map[string]string{},
		},
		{
			name:   "library malformed ID keeps authentication",
			method: "PUT",
			path:   "/api/me/library/lists/favorite/bad",
			body:   "{}",
			status: 401,
		},
		{
			name:      "library variables and body are preserved",
			method:    "PUT",
			path:      "/api/me/library/lists/favorite/42",
			body:      "{\"saved\":true}",
			cookie:    true,
			role:      "reader",
			status:    200,
			operation: "SetSaved",
			vars:      map[string]string{"list": "favorite", "film": "42"},
		},
		{
			name:      "preferences body is preserved",
			method:    "PUT",
			path:      "/api/me/preferences",
			body:      "{\"key\":\"frame:catalog-view\",\"value\":\"grid\"}",
			cookie:    true,
			role:      "reader",
			status:    200,
			operation: "SavePreference",
			vars:      map[string]string{},
		},
		{
			name:      "disabled handler is not exposed",
			method:    "GET",
			path:      "/api/films/42",
			omitFilms: true,
			status:    404,
		},
		{
			name:   "disabled profile does not require authentication",
			method: "GET",
			path:   "/api/me/profile",
			status: 404,
		},
		{
			name:      "nested static key is preserved",
			method:    "GET",
			path:      "/static/posters/nested/film.png",
			status:    200,
			operation: "GetAsset",
			vars:      map[string]string{"key": "posters/nested/film.png"},
		},
		{
			name:   "unknown path remains missing",
			method: "GET",
			path:   "/api/no-such-path",
			status: 404,
		},
		{
			name:   "unsupported method remains rejected",
			method: "PATCH",
			path:   "/api/films/42",
			status: 405,
		},
		{
			name:   "preflight remains handled by CORS",
			method: "OPTIONS",
			path:   "/api/no-such-path",
			status: 204,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			require.NoError(t, err)

			auth := authStub{
				role: tt.role,
			}
			users := userStub{
				role: tt.role,
			}

			handlers := router.Handlers{
				Auth:    authhttp.NewAuthHandler(auth, users, appLogger),
				Library: openAPIRouteProbe{},
				Assets:  openAPIRouteProbe{},
			}
			if !tt.omitFilms {
				handlers.Films = openAPIRouteProbe{}
			}

			handler, err := router.NewRouter(handlers, middleware.NewAuthMiddleware(auth, users, appLogger), []string{"http://localhost:8000"}, appLogger)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Origin", "http://localhost:8000")
			req.Header.Set("Content-Type", "application/json")

			if tt.cookie {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			require.Equal(t, tt.status, response.Code)
			require.Equal(t, "http://localhost:8000", response.Header().Get("Access-Control-Allow-Origin"))
			require.NotEmpty(t, response.Header().Get("X-Request-ID"))

			if tt.operation != "" {
				expected, err := json.Marshal(map[string]any{
					"operation": tt.operation,
					"vars":      tt.vars,
					"query":     tt.query,
					"body":      tt.body,
				})
				require.NoError(t, err)
				require.JSONEq(t, string(expected), response.Body.String())
			}
		})
	}
}
