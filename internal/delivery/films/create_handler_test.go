package films

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type createHandlerRepository struct {
	called bool
	input  domain.CreateFilmInput
	err    error
}

func (s *createHandlerRepository) CreateFilm(_ context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	s.called = true
	s.input = input
	if s.err != nil {
		return 0, s.err
	}
	return 42, nil
}

func TestCreateFilmHandler(t *testing.T) {
	const valid = `{"title":"  Alien  ","film_type":"movie","release_year":1979,"age_limit":0}`
	tests := []struct {
		name       string
		body       string
		storageErr error
		status     int
		called     bool
		message    string
	}{
		{"created with age zero", valid, nil, http.StatusCreated, true, ""},
		{"trailing whitespace", valid + " \n\t", nil, http.StatusCreated, true, ""},
		{"all fields", `{"title":"Alien","original_title":"Alien","film_type":"movie","release_year":1979,"duration_min":117,"age_limit":16,"description":"Space"}`, nil, http.StatusCreated, true, ""},
		{"empty body", "", nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"broken JSON", "{", nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"wrong field type", `{"title":5,"age_limit":0}`, nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"integer overflow", `{"release_year":40000,"age_limit":0}`, nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"unknown field", strings.TrimSuffix(valid, "}") + `,"titel":"typo"}`, nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"age missing", `{"title":"Alien"}`, nil, http.StatusBadRequest, false, "age_limit is required"},
		{"age null", `{"age_limit":null}`, nil, http.StatusBadRequest, false, "age_limit is required"},
		{"null object", "null", nil, http.StatusBadRequest, false, "age_limit is required"},
		{"array", "[]", nil, http.StatusBadRequest, false, "invalid JSON request body"},
		{"second object", valid + " {}", nil, http.StatusBadRequest, false, "exactly one JSON object"},
		{"second null", valid + " null", nil, http.StatusBadRequest, false, "exactly one JSON object"},
		{"broken suffix", valid + " garbage", nil, http.StatusBadRequest, false, "exactly one JSON object"},
		{"invalid age", strings.Replace(valid, `"age_limit":0`, `"age_limit":7`, 1), nil, http.StatusBadRequest, false, "invalid age limit"},
		{"invalid title", strings.Replace(valid, "  Alien  ", "   ", 1), nil, http.StatusBadRequest, false, "invalid film input"},
		{"storage failure", valid, errors.New("private database credentials"), http.StatusInternalServerError, true, "internal server error"},
		{"body exactly at limit", valid + strings.Repeat(" ", maxCreateFilmBodyBytes-len(valid)), nil, http.StatusCreated, true, ""},
		{"oversized suffix", valid + strings.Repeat(" ", maxCreateFilmBodyBytes), nil, http.StatusRequestEntityTooLarge, false, "request body exceeds 1 MiB"},
		{"oversized first object", `{"title":"Alien","description":"` + strings.Repeat("a", maxCreateFilmBodyBytes) + `"}`, nil, http.StatusRequestEntityTooLarge, false, "request body exceeds 1 MiB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &createHandlerRepository{err: tt.storageErr}
			uc, err := usecase.NewCreateFilmUseCase(repo)
			require.NoError(t, err)
			handler := &Handler{createFilm: uc}
			req := httptest.NewRequest(http.MethodPost, "/api/films", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler.CreateFilm(rec, req)

			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.called, repo.called)
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
			if tt.status == http.StatusCreated {
				var body createFilmResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, domain.FilmID(42), body.ID)
				assert.Equal(t, "Alien", repo.input.Title)
				if tt.name == "all fields" {
					require.NotNil(t, repo.input.OriginalTitle)
					require.NotNil(t, repo.input.DurationMin)
					require.NotNil(t, repo.input.Description)
					assert.Equal(t, "Alien", *repo.input.OriginalTitle)
					assert.Equal(t, int16(117), *repo.input.DurationMin)
					assert.Equal(t, "Space", *repo.input.Description)
				}
			} else {
				var body errorResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body.Error, tt.message)
				assert.NotContains(t, rec.Body.String(), "private database credentials")
			}
		})
	}
}
