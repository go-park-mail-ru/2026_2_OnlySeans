package films_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/films"
)

var errStorage = errors.New("storage is down")

const (
	seedFilmCount        = 12
	internalErrorMessage = "internal server error"
)

var seedCollectionFilms = []struct {
	slug    string
	filmIDs []entities.FilmID
}{
	{"best-of-all-time", []entities.FilmID{1, 2, 3, 4, 5, 7}},
	{"family", []entities.FilmID{7, 8, 3, 9}},
	{"mind-benders", []entities.FilmID{5, 6, 4, 10}},
	{"weekend-series", []entities.FilmID{11, 12}},
}

type filmsPage struct {
	Films  []entities.Film `json:"films"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type filmResponse struct {
	Film entities.Film `json:"film"`
}

type collectionsPage struct {
	Collections []entities.CollectionWithFilms `json:"collections"`
	Total       int                            `json:"total"`
	Limit       int                            `json:"limit"`
	Offset      int                            `json:"offset"`
}

type collectionPage struct {
	Collection entities.CollectionWithFilms `json:"collection"`
	Total      int                          `json:"total"`
	Limit      int                          `json:"limit"`
	Offset     int                          `json:"offset"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type brokenService struct {
	failCollectionFilms bool
}

func (b brokenService) ListFilms(context.Context, int, int) ([]entities.Film, int, error) {
	return nil, 0, errStorage
}

func (b brokenService) GetFilmByID(context.Context, entities.FilmID) (entities.Film, error) {
	return entities.Film{}, errStorage
}

func (b brokenService) ListCollections(context.Context, int, int) ([]entities.Collection, int, error) {
	if b.failCollectionFilms {
		return []entities.Collection{
			{
				ID:   1,
				Slug: "family",
			},
		}, 1, nil
	}
	return nil, 0, errStorage
}

func (b brokenService) GetCollectionByID(_ context.Context, id entities.CollectionID) (entities.Collection, error) {
	if b.failCollectionFilms {
		return entities.Collection{
			ID:   id,
			Slug: "family",
		}, nil
	}
	return entities.Collection{}, errStorage
}

func (b brokenService) ListCollectionFilms(context.Context, entities.CollectionID, int, int) ([]entities.Film, int, error) {
	return nil, 0, errStorage
}

func newTestRouter(service films.Service) *mux.Router {
	router := mux.NewRouter()
	h := films.NewHandler(service)

	router.HandleFunc("/api/films", h.ListFilms).Methods(http.MethodGet)
	router.HandleFunc("/api/films/{id}", h.GetFilm).Methods(http.MethodGet)
	router.HandleFunc("/api/collections", h.ListCollections).Methods(http.MethodGet)
	router.HandleFunc("/api/collections/{id}", h.GetCollection).Methods(http.MethodGet)

	return router
}

func newSeededRouter(t *testing.T) *mux.Router {
	t.Helper()

	service, err := films.NewSeededService()
	if err != nil {
		t.Fatalf("NewSeededService: %v", err)
	}
	return newTestRouter(service)
}

func serve(router http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return body
}

func assertErrorBody(t *testing.T, rec *httptest.ResponseRecorder, want error) {
	t.Helper()

	got := decode[errorResponse](t, rec).Error
	if !strings.HasPrefix(got, want.Error()) {
		t.Errorf("error = %q, want prefix %q", got, want.Error())
	}
}

func silenceLog(t *testing.T) {
	t.Helper()

	previous := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(previous) })
}

func TestHandler_ListFilms(t *testing.T) {
	router := newSeededRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantIDs    []entities.FilmID
		wantError  error
	}{
		{
			name:       "default params",
			target:     "/api/films",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			name:       "limit and offset",
			target:     "/api/films?limit=3&offset=1",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{2, 3, 4},
		},
		{
			name:       "max limit",
			target:     "/api/films?limit=100",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			name:       "page out of range",
			target:     "/api/films?offset=1000",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{},
		},
		{
			name:       "limit is not a number",
			target:     "/api/films?limit=abc",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidLimit,
		},
		{
			name:       "limit is zero",
			target:     "/api/films?limit=0",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidLimit,
		},
		{
			name:       "limit above max",
			target:     "/api/films?limit=101",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidLimit,
		},
		{
			name:       "offset is not a number",
			target:     "/api/films?offset=abc",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidOffset,
		},
		{
			name:       "negative offset",
			target:     "/api/films?offset=-1",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(router, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
			}

			if tt.wantError != nil {
				assertErrorBody(t, rec, tt.wantError)
				return
			}

			page := decode[filmsPage](t, rec)
			if page.Films == nil {
				t.Fatal("films must be an array, not null")
			}
			if got := filmIDs(page.Films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", got, tt.wantIDs)
			}
			if page.Total != seedFilmCount {
				t.Errorf("total = %d, want %d", page.Total, seedFilmCount)
			}
		})
	}
}

func TestHandler_GetFilm(t *testing.T) {
	router := newSeededRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantID     entities.FilmID
		wantError  error
	}{
		{
			name:       "existing film",
			target:     "/api/films/7",
			wantStatus: http.StatusOK,
			wantID:     7,
		},
		{
			name:       "unknown film",
			target:     "/api/films/999",
			wantStatus: http.StatusNotFound,
			wantError:  films.ErrFilmNotFound,
		},
		{
			name:       "id is not a number",
			target:     "/api/films/abc",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidFilmID,
		},
		{
			name:       "zero id",
			target:     "/api/films/0",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidFilmID,
		},
		{
			name:       "negative id",
			target:     "/api/films/-1",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidFilmID,
		},
		{
			name:       "id overflows int64",
			target:     "/api/films/99999999999999999999",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidFilmID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(router, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantError != nil {
				assertErrorBody(t, rec, tt.wantError)
				return
			}

			film := decode[filmResponse](t, rec).Film
			if film.ID != tt.wantID || film.Title == "" || len(film.Genres) == 0 {
				t.Errorf("film = %+v", film)
			}
		})
	}
}

func TestHandler_ListCollections(t *testing.T) {
	router := newSeededRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantFirst  int
		wantCount  int
		wantError  error
	}{
		{
			name:       "default params",
			target:     "/api/collections",
			wantStatus: http.StatusOK,
			wantFirst:  0,
			wantCount:  4,
		},
		{
			name:       "limit and offset",
			target:     "/api/collections?limit=2&offset=1",
			wantStatus: http.StatusOK,
			wantFirst:  1,
			wantCount:  2,
		},
		{
			name:       "tail shorter than limit",
			target:     "/api/collections?limit=3&offset=3",
			wantStatus: http.StatusOK,
			wantFirst:  3,
			wantCount:  1,
		},
		{
			name:       "page out of range",
			target:     "/api/collections?offset=100",
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name:       "limit above max",
			target:     "/api/collections?limit=101",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidLimit,
		},
		{
			name:       "negative offset",
			target:     "/api/collections?offset=-1",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(router, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantError != nil {
				assertErrorBody(t, rec, tt.wantError)
				return
			}

			page := decode[collectionsPage](t, rec)
			if page.Collections == nil {
				t.Fatal("collections must be an array, not null")
			}
			if page.Total != len(seedCollectionFilms) {
				t.Errorf("total = %d, want %d", page.Total, len(seedCollectionFilms))
			}
			if len(page.Collections) != tt.wantCount {
				t.Fatalf("got %d collections, want %d", len(page.Collections), tt.wantCount)
			}
			for i, collection := range page.Collections {
				want := seedCollectionFilms[tt.wantFirst+i]
				if collection.ID != entities.CollectionID(tt.wantFirst+i+1) || collection.Slug != want.slug {
					t.Errorf("collection %d = %d %q, want %q", i, collection.ID, collection.Slug, want.slug)
				}
				if got := filmIDs(collection.Films); !slices.Equal(got, want.filmIDs) {
					t.Errorf("films of %q = %v, want %v", collection.Slug, got, want.filmIDs)
				}
			}
		})
	}
}

func TestHandler_GetCollection(t *testing.T) {
	router := newSeededRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantIDs    []entities.FilmID
		wantError  error
	}{
		{
			name:       "existing collection",
			target:     "/api/collections/2",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{7, 8, 3, 9},
		},
		{
			name:       "limit and offset",
			target:     "/api/collections/2?limit=2&offset=1",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{8, 3},
		},
		{
			name:       "tail shorter than limit",
			target:     "/api/collections/2?limit=3&offset=3",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{9},
		},
		{
			name:       "page out of range",
			target:     "/api/collections/2?offset=50",
			wantStatus: http.StatusOK,
			wantIDs:    []entities.FilmID{},
		},
		{
			name:       "unknown collection",
			target:     "/api/collections/999",
			wantStatus: http.StatusNotFound,
			wantError:  films.ErrCollectionNotFound,
		},
		{
			name:       "slug instead of id",
			target:     "/api/collections/family",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidCollectionID,
		},
		{
			name:       "zero id",
			target:     "/api/collections/0",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidCollectionID,
		},
		{
			name:       "limit is zero",
			target:     "/api/collections/2?limit=0",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidLimit,
		},
		{
			name:       "offset is not a number",
			target:     "/api/collections/2?offset=abc",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidOffset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(router, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantError != nil {
				assertErrorBody(t, rec, tt.wantError)
				return
			}

			page := decode[collectionPage](t, rec)
			if page.Collection.ID != 2 || page.Collection.Slug != "family" || page.Collection.Title == "" {
				t.Errorf("collection = %+v", page.Collection.Collection)
			}
			if page.Collection.Films == nil {
				t.Fatal("films must be an array, not null")
			}
			if got := filmIDs(page.Collection.Films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", got, tt.wantIDs)
			}
			if page.Total != 4 {
				t.Errorf("total = %d, want 4", page.Total)
			}
		})
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	router := newSeededRouter(t)

	for _, target := range []string{"/api/films", "/api/films/1", "/api/collections", "/api/collections/2"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			if rec := serve(router, method, target); rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: status = %d, want %d", method, target, rec.Code, http.StatusMethodNotAllowed)
			}
		}
	}
}

func TestHandler_InternalErrorIsHidden(t *testing.T) {
	silenceLog(t)

	tests := []struct {
		name    string
		service brokenService
		target  string
	}{
		{
			name:    "list films",
			service: brokenService{},
			target:  "/api/films",
		},
		{
			name:    "get film",
			service: brokenService{},
			target:  "/api/films/1",
		},
		{
			name:    "list collections",
			service: brokenService{},
			target:  "/api/collections",
		},
		{
			name:    "list collections: films of collection",
			service: brokenService{failCollectionFilms: true},
			target:  "/api/collections",
		},
		{
			name:    "get collection",
			service: brokenService{},
			target:  "/api/collections/1",
		},
		{
			name:    "get collection: films of collection",
			service: brokenService{failCollectionFilms: true},
			target:  "/api/collections/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(newTestRouter(tt.service), http.MethodGet, tt.target)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
			}
			if got := decode[errorResponse](t, rec).Error; got != internalErrorMessage {
				t.Errorf("error = %q, internal details must not leak to client", got)
			}
		})
	}
}

func TestHandler_InvalidLimitMentionsBounds(t *testing.T) {
	rec := serve(newSeededRouter(t), http.MethodGet, "/api/films?limit=0")

	got := decode[errorResponse](t, rec).Error
	if !strings.Contains(got, "1 to 100") {
		t.Errorf("error = %q, want bounds in message", got)
	}
}
