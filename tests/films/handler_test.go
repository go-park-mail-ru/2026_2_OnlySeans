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

type collectionsResponse struct {
	Collections []entities.CollectionWithFilms `json:"collections"`
}

type collectionResponse struct {
	Collection entities.CollectionWithFilms `json:"collection"`
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

func (b brokenService) ListCollections(context.Context) ([]entities.Collection, error) {
	if b.failCollectionFilms {
		return []entities.Collection{
			{
				ID:   1,
				Slug: "family",
			},
		}, nil
	}
	return nil, errStorage
}

func (b brokenService) GetCollectionBySlug(_ context.Context, slug string) (entities.Collection, error) {
	if b.failCollectionFilms {
		return entities.Collection{
			ID:   1,
			Slug: slug,
		}, nil
	}
	return entities.Collection{}, errStorage
}

func (b brokenService) ListCollectionFilms(context.Context, entities.CollectionID) ([]entities.Film, error) {
	return nil, errStorage
}

func newTestRouter(service films.Service) *mux.Router {
	router := mux.NewRouter()
	films.NewHandler(service).RegisterRoutes(router)
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
				if got := decode[errorResponse](t, rec).Error; got != tt.wantError.Error() {
					t.Errorf("error = %q, want %q", got, tt.wantError.Error())
				}
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

func TestHandler_ListCollections(t *testing.T) {
	rec := serve(newSeededRouter(t), http.MethodGet, "/api/collections")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	body := decode[collectionsResponse](t, rec)
	if len(body.Collections) != len(seedCollectionFilms) {
		t.Fatalf("got %d collections, want %d", len(body.Collections), len(seedCollectionFilms))
	}
	for i, collection := range body.Collections {
		if collection.Slug != seedCollectionFilms[i].slug {
			t.Errorf("collection %d: slug = %q, want %q", i, collection.Slug, seedCollectionFilms[i].slug)
		}
		if got, want := filmIDs(collection.Films), seedCollectionFilms[i].filmIDs; !slices.Equal(got, want) {
			t.Errorf("films of %q = %v, want %v", collection.Slug, got, want)
		}
	}
}

func TestHandler_GetCollection(t *testing.T) {
	router := newSeededRouter(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantError  error
	}{
		{
			name:       "existing collection",
			target:     "/api/collections/family",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown collection",
			target:     "/api/collections/unknown",
			wantStatus: http.StatusNotFound,
			wantError:  films.ErrCollectionNotFound,
		},
		{
			name:       "invalid slug",
			target:     "/api/collections/BAD_SLUG",
			wantStatus: http.StatusBadRequest,
			wantError:  films.ErrInvalidSlug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(router, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantError != nil {
				if got := decode[errorResponse](t, rec).Error; got != tt.wantError.Error() {
					t.Errorf("error = %q, want %q", got, tt.wantError.Error())
				}
				return
			}

			collection := decode[collectionResponse](t, rec).Collection
			if collection.Slug != "family" || collection.Title == "" {
				t.Errorf("collection = %+v", collection.Collection)
			}
			if got, want := filmIDs(collection.Films), []entities.FilmID{7, 8, 3, 9}; !slices.Equal(got, want) {
				t.Errorf("ids = %v, want %v", got, want)
			}
		})
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	router := newSeededRouter(t)

	for _, target := range []string{"/api/films", "/api/collections", "/api/collections/family"} {
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
			target:  "/api/collections/family",
		},
		{
			name:    "get collection: films of collection",
			service: brokenService{failCollectionFilms: true},
			target:  "/api/collections/family",
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
