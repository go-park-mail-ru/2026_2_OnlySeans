package films_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"go.uber.org/mock/gomock"
)

func TestListFilmsDetails(t *testing.T) {
	description := "A film description"
	duration := int16(120)
	for _, query := range []string{"", "&details=false", "&details=true"} {
		t.Run(query, func(t *testing.T) {
			repo := mocks.NewMockFilmRepository(gomock.NewController(t))
			repo.EXPECT().ListFilms(gomock.Any(), 2, 3).Return([]domain.Film{{ID: 1, Title: "Film", Description: &description, FilmType: domain.FilmTypeMovie, DurationMin: &duration, AgeLimit: 16}}, 7, nil)
			handler := newTestRouter(t, repo)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/films?limit=2&offset=3"+query, nil))
			var body struct {
				Films  []map[string]any `json:"films"`
				Total  int              `json:"total"`
				Limit  int              `json:"limit"`
				Offset int              `json:"offset"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 200 || len(body.Films) != 1 || body.Total != 7 || body.Limit != 2 || body.Offset != 3 {
				t.Fatalf("unexpected response: %s", rec.Body.String())
			}
			for _, field := range []string{"description", "film_type", "duration_min", "age_limit"} {
				_, present := body.Films[0][field]
				if present != (query == "&details=true") {
					t.Errorf("unexpected field %s in %s", field, rec.Body.String())
				}
			}
			if query == "&details=true" && body.Films[0]["description"] != description {
				t.Fatal("description was lost")
			}
		})
	}
}

func TestListFilmsByGenre(t *testing.T) {
	repo := mocks.NewMockFilmRepository(gomock.NewController(t))
	repo.EXPECT().
		ListFilmsByGenre(gomock.Any(), 2, 3, "comedy").
		Return([]domain.Film{{ID: 7, Title: "Comedy"}}, 4, nil)

	handler := newTestRouter(t, repo)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/films?limit=2&offset=3&genre=comedy", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Films []struct {
			ID domain.FilmID `json:"id"`
		} `json:"films"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Films) != 1 || body.Films[0].ID != 7 {
		t.Fatalf("films = %v, want one film with id 7", body.Films)
	}
	if body.Total != 4 {
		t.Fatalf("total = %d, want 4", body.Total)
	}
}
