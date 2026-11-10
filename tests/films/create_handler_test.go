package films_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testMaxFilmRequestBodyBytes = 1 << 20

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
		{"second object", valid + " {}", nil, http.StatusCreated, true, ""},
		{"second null", valid + " null", nil, http.StatusCreated, true, ""},
		{"broken suffix", valid + " garbage", nil, http.StatusCreated, true, ""},
		{"invalid age", strings.Replace(valid, `"age_limit":0`, `"age_limit":7`, 1), nil, http.StatusBadRequest, false, "invalid age limit"},
		{"invalid title", strings.Replace(valid, "  Alien  ", "   ", 1), nil, http.StatusBadRequest, false, "invalid film input"},
		{"storage failure", valid, errors.New("private database credentials"), http.StatusInternalServerError, true, "internal server error"},
		{"body exactly at limit", valid + strings.Repeat(" ", testMaxFilmRequestBodyBytes-len(valid)), nil, http.StatusCreated, true, ""},
		{"oversized suffix", valid + strings.Repeat(" ", testMaxFilmRequestBodyBytes), nil, http.StatusCreated, true, ""},
		{"oversized first object", `{"title":"Alien","description":"` + strings.Repeat("a", testMaxFilmRequestBodyBytes) + `"}`, nil, http.StatusRequestEntityTooLarge, false, "request body exceeds 1 MiB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockFilmRepository(gomock.NewController(t))
			var captured domain.CreateFilmInput
			if tt.called {
				repo.EXPECT().CreateFilm(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
					captured = input
					if tt.storageErr != nil {
						return domain.FilmID(0), tt.storageErr
					}
					return domain.FilmID(42), nil
				}).Times(1)
			}
			uc, err := usecase.NewCreateFilmUseCase(repo, newFilmsTestLogger())
			require.NoError(t, err)
			appLogger := newFilmsTestLogger()
			list, err := usecase.NewListFilmsUseCase(repo, appLogger)
			require.NoError(t, err)
			get, err := usecase.NewGetFilmUseCase(repo, appLogger)
			require.NoError(t, err)
			update, err := usecase.NewUpdateFilmUseCase(repo, newFilmsTestLogger())
			require.NoError(t, err)
			deleteFilm, err := usecase.NewDeleteFilmUseCase(repo, newFilmsTestLogger())
			require.NoError(t, err)
			listCollections, err := usecase.NewListCollectionsUseCase(mocks.NewMockCollectionRepository(gomock.NewController(t)), appLogger)
			require.NoError(t, err)
			getCollection, err := usecase.NewGetCollectionUseCase(mocks.NewMockCollectionRepository(gomock.NewController(t)), appLogger)
			require.NoError(t, err)
			handler, err := films.NewHandler(films.HandlerUseCases{
				CreateFilm: uc, ListFilms: list, GetFilm: get, UpdateFilm: update,
				DeleteFilm: deleteFilm, ListCollections: listCollections, GetCollection: getCollection,
			}, appLogger)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/api/films", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler.CreateFilm(rec, req)

			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
			if tt.status == http.StatusCreated {
				var body struct {
					ID domain.FilmID `json:"id"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, domain.FilmID(42), body.ID)
				assert.Equal(t, "Alien", captured.Title)
				if tt.name == "all fields" {
					require.NotNil(t, captured.OriginalTitle)
					require.NotNil(t, captured.DurationMin)
					require.NotNil(t, captured.Description)
					assert.Equal(t, "Alien", *captured.OriginalTitle)
					assert.Equal(t, int16(117), *captured.DurationMin)
					assert.Equal(t, "Space", *captured.Description)
				}
			} else {
				var body map[string]string
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body["error"], tt.message)
				assert.NotContains(t, rec.Body.String(), "private database credentials")
			}
		})
	}
}
