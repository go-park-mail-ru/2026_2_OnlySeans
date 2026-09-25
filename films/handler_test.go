package films

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func newTestMux(t *testing.T, repo Repository) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	NewHandler(NewUseCase(repo)).RegisterRoutes(mux)
	return mux
}

func newSeededMux(t *testing.T) *http.ServeMux {
	t.Helper()

	repo, err := NewSeededRepo()
	if err != nil {
		t.Fatalf("NewSeededRepo: %v", err)
	}
	return newTestMux(t, repo)
}

func serve(mux *http.ServeMux, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("некорректный JSON %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestHandler_ListFilms(t *testing.T) {
	mux := newSeededMux(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantIDs    []int64
		wantError  error
	}{
		{"параметры по умолчанию", "/api/films", http.StatusOK, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, nil},
		{"limit и offset", "/api/films?limit=3&offset=1", http.StatusOK, []int64{2, 3, 4}, nil},
		{"страница за пределами", "/api/films?offset=1000", http.StatusOK, []int64{}, nil},
		{"limit не число", "/api/films?limit=abc", http.StatusBadRequest, nil, ErrInvalidLimit},
		{"limit вне диапазона", "/api/films?limit=1000", http.StatusBadRequest, nil, ErrInvalidLimit},
		{"offset не число", "/api/films?offset=abc", http.StatusBadRequest, nil, ErrInvalidOffset},
		{"отрицательный offset", "/api/films?offset=-1", http.StatusBadRequest, nil, ErrInvalidOffset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(mux, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d (тело: %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
				t.Errorf("Content-Type = %q", rec.Header().Get("Content-Type"))
			}

			if tt.wantError != nil {
				if got := decode[errorResponse](t, rec).Error; got != tt.wantError.Error() {
					t.Errorf("error = %q, ожидали %q", got, tt.wantError.Error())
				}
				return
			}

			page := decode[FilmsPage](t, rec)
			if page.Films == nil {
				t.Fatal("films должен быть массивом, а не null")
			}
			if got := filmIDs(page.Films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, ожидали %v", got, tt.wantIDs)
			}
			if page.Total != len(seedFilms) {
				t.Errorf("total = %d, ожидали %d", page.Total, len(seedFilms))
			}
		})
	}
}

func TestHandler_ListCollections(t *testing.T) {
	rec := serve(newSeededMux(t), http.MethodGet, "/api/collections")
	if rec.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидали %d (тело: %s)", rec.Code, http.StatusOK, rec.Body)
	}

	body := decode[collectionsResponse](t, rec)
	if len(body.Collections) != len(seedCollections) {
		t.Fatalf("получили %d подборок, ожидали %d", len(body.Collections), len(seedCollections))
	}
	for i, collection := range body.Collections {
		if got, want := filmIDs(collection.Films), seedCollections[i].filmIDs; !slices.Equal(got, want) {
			t.Errorf("фильмы подборки %q = %v, ожидали %v", collection.Slug, got, want)
		}
	}
}

func TestHandler_GetCollection(t *testing.T) {
	mux := newSeededMux(t)

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantError  error
	}{
		{"существующая подборка", "/api/collections/family", http.StatusOK, nil},
		{"несуществующая подборка", "/api/collections/unknown", http.StatusNotFound, ErrCollectionNotFound},
		{"некорректный slug", "/api/collections/BAD_SLUG", http.StatusBadRequest, ErrInvalidSlug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(mux, http.MethodGet, tt.target)
			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d (тело: %s)", rec.Code, tt.wantStatus, rec.Body)
			}

			if tt.wantError != nil {
				if got := decode[errorResponse](t, rec).Error; got != tt.wantError.Error() {
					t.Errorf("error = %q, ожидали %q", got, tt.wantError.Error())
				}
				return
			}

			collection := decode[collectionResponse](t, rec).Collection
			if collection.Slug != "family" || collection.Title == "" {
				t.Errorf("подборка = %+v", collection.Collection)
			}
			if got, want := filmIDs(collection.Films), []int64{7, 8, 3, 9}; !slices.Equal(got, want) {
				t.Errorf("ids = %v, ожидали %v", got, want)
			}
		})
	}
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	mux := newSeededMux(t)

	for _, target := range []string{"/api/films", "/api/collections", "/api/collections/family"} {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
			if rec := serve(mux, method, target); rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: статус = %d, ожидали %d", method, target, rec.Code, http.StatusMethodNotAllowed)
			}
		}
	}
}

func TestHandler_InternalErrorIsHidden(t *testing.T) {
	previous := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(previous) })

	mux := newTestMux(t, brokenRepo{})

	for _, target := range []string{"/api/films", "/api/collections", "/api/collections/family"} {
		rec := serve(mux, http.MethodGet, target)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s: статус = %d, ожидали %d", target, rec.Code, http.StatusInternalServerError)
			continue
		}
		if got := decode[errorResponse](t, rec).Error; got != internalErrorMessage {
			t.Errorf("%s: error = %q, детали ошибки не должны уходить клиенту", target, got)
		}
	}
}
