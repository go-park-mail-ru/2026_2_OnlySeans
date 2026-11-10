package persons_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	uc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCatalogRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, path       string
		repoErr, filmErr error
		status           int
	}{
		{"film participants", "/api/films/5/people", nil, nil, 200},
		{"filmography", "/api/people/7", nil, nil, 200},
		{"invalid film ID", "/api/films/abc/people", nil, nil, 400},
		{"invalid person ID", "/api/people/0", nil, nil, 400},
		{"missing film", "/api/films/5/people", nil, domain.ErrFilmNotFound, 404},
		{"missing person", "/api/people/7", domain.ErrPersonNotFound, nil, 404},
		{"catalog failure", "/api/films/5/people", errors.New("database unavailable"), nil, 500},
		{"filmography film failure", "/api/people/7", nil, errors.New("database unavailable"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
			require.NoError(t, err)
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockPersonCatalogRepository(ctrl)
			films := mocks.NewMockCatalogFilmRepository(ctrl)
			person := domain.CatalogPerson{ID: 7, Name: "Actor"}
			key := "private-storage-key"
			film := domain.Film{ID: 5, Title: "Film", FilmType: domain.FilmTypeMovie, ReleaseYear: 2026, PosterStorageKey: &key, Genres: []domain.Genre{{ID: 1, Name: "Drama", Slug: "drama"}}}
			if tc.path == "/api/films/5/people" {
				lookup := films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(5)).Return(film, tc.filmErr)
				if tc.filmErr == nil {
					repo.EXPECT().FilmPersons(gomock.Any(), int64(5)).Return([]domain.PersonCredit{{Person: person, Role: "actor", Character: "Hero"}}, tc.repoErr).After(lookup)
				}
			} else if tc.path == "/api/people/7" {
				lookup := repo.EXPECT().Get(gomock.Any(), int64(7)).Return(person, []domain.FilmCredit{{FilmID: 5, Role: "actor", Character: "Hero"}}, tc.repoErr)
				if tc.repoErr == nil {
					films.EXPECT().GetFilmByID(gomock.Any(), domain.FilmID(5)).Return(film, tc.filmErr).After(lookup)
				}
			}
			router := mux.NewRouter()
			h := delivery.NewCatalogHandler(uc.NewGetFilmPersonsUseCase(repo, films), uc.NewGetPersonCatalogUseCase(repo, films), appLogger)
			router.HandleFunc("/api/films/{id}/people", h.GetFilmPersons).Methods(http.MethodGet)
			router.HandleFunc("/api/people/{id}", h.GetPersonCatalog).Methods(http.MethodGet)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			require.NotContains(t, rec.Body.String(), "private-storage-key")
			if tc.status != 200 {
				return
			}
			var data map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &data))
			if tc.path == "/api/films/5/people" {
				require.JSONEq(t, `[{"person":{"id":7,"name":"Actor","original_name":"","photo_source":"","photo_credit":"","birth_place":"","biography":"","source_url":""},"role":"actor","character":"Hero"}]`, string(data["people"]))
			} else {
				var credits []struct {
					Film struct {
						ID     int64
						Title  string
						Genres []struct{ ID int64 }
					}
					Role, Character string
				}
				require.NoError(t, json.Unmarshal(data["films"], &credits))
				require.Len(t, credits, 1)
				require.Equal(t, int64(5), credits[0].Film.ID)
				require.Equal(t, "Film", credits[0].Film.Title)
				require.Len(t, credits[0].Film.Genres, 1)
				require.Equal(t, "actor", credits[0].Role)
				require.Equal(t, "Hero", credits[0].Character)
			}
		})
	}
}
