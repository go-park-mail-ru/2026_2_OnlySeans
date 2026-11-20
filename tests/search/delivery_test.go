package search_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	searchdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/search"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	searchusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/search"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

type searchUseCaseFunc func(context.Context, searchusecase.SearchSuggestionUsecaseInput) (*searchusecase.SearchSuggestionUsecaseOutput, error)

func (f searchUseCaseFunc) Run(ctx context.Context, input searchusecase.SearchSuggestionUsecaseInput) (*searchusecase.SearchSuggestionUsecaseOutput, error) {
	return f(ctx, input)
}

func TestSearchRouteJSONResponses(t *testing.T) {
	cases := []struct {
		name   string
		url    string
		status int
		calls  int
		limit  int
		err    error
	}{
		{"missing query", "/api/search/suggest", 400, 0, 0, domain.ErrInvalidSearchQuery},
		{"invalid limit format", "/api/search/suggest?q=matrix&limit=abc", 400, 0, 0, domain.ErrInvalidSearchLimit},
		{"limit overflow", "/api/search/suggest?q=matrix&limit=999999999999999999999999", 400, 0, 0, domain.ErrInvalidSearchLimit},
		{"default limit", "/api/search/suggest?q=matrix", 200, 1, 3, nil},
		{"explicit limit", "/api/search/suggest?q=matrix&limit=5", 200, 1, 5, nil},
		{"invalid query from usecase", "/api/search/suggest?q=x", 400, 1, 3, domain.ErrInvalidSearchQuery},
		{"zero limit from usecase", "/api/search/suggest?q=matrix&limit=0", 400, 1, 0, domain.ErrInvalidSearchLimit},
		{"internal error", "/api/search/suggest?q=matrix", 500, 1, 3, errors.New("private database details")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0

			handler, err := searchdelivery.NewSearchHandler(searchUseCaseFunc(func(_ context.Context, input searchusecase.SearchSuggestionUsecaseInput) (*searchusecase.SearchSuggestionUsecaseOutput, error) {
				calls++

				if input.Limit != tc.limit {
					t.Errorf("limit = %d, want %d", input.Limit, tc.limit)
				}

				if tc.err != nil {
					return nil, tc.err
				}

				return &searchusecase.SearchSuggestionUsecaseOutput{}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			// Exercise the actual application router, including common middleware.
			r, err := router.NewRouter(router.Handlers{Search: handler}, nil, testutil.DiscardLogger())
			if err != nil {
				t.Fatal(err)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.url, nil))

			if rec.Code != tc.status || calls != tc.calls {
				t.Fatalf("status/calls = %d/%d, want %d/%d; body=%s", rec.Code, calls, tc.status, tc.calls, rec.Body.String())
			}

			if rec.Header().Get("Content-Type") != "application/json" && rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("unexpected content type: %s", rec.Header().Get("Content-Type"))
			}

			var body map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}

			if tc.status == http.StatusOK {
				if string(body["films"]) != "[]" || string(body["persons"]) != "[]" {
					t.Fatalf("empty lists must be arrays: %s", rec.Body.String())
				}
			} else {
				message := "internal server error"
				if tc.status == http.StatusBadRequest {
					message = tc.err.Error()
				}

				var got string
				if err := json.Unmarshal(body["error"], &got); err != nil || got != message {
					t.Fatalf("error body = %s, want %q", rec.Body.String(), message)
				}
			}
		})
	}
}

type failedResponseWriter struct {
	header http.Header
	writes int
}

func (w *failedResponseWriter) Header() http.Header { return w.header }
func (w *failedResponseWriter) WriteHeader(int)     {}
func (w *failedResponseWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("connection closed")
}

func TestSearchResponseWriteFailureDoesNotAppendError(t *testing.T) {
	handler, err := searchdelivery.NewSearchHandler(searchUseCaseFunc(func(context.Context, searchusecase.SearchSuggestionUsecaseInput) (*searchusecase.SearchSuggestionUsecaseOutput, error) {
		return &searchusecase.SearchSuggestionUsecaseOutput{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	r := mux.NewRouter()
	searchdelivery.RegisterRoutes(r, handler, testutil.DiscardLogger())

	w := &failedResponseWriter{header: make(http.Header)}
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/search/suggest?q=matrix", nil))

	if w.writes != 1 {
		t.Fatalf("response writes = %d, want 1", w.writes)
	}
}
