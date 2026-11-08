package films

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
)

type detailsRepo struct {
	domain.FilmRepository
	films []domain.Film
	calls int
}

func (r *detailsRepo) ListFilms(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
	r.calls++
	return r.films, 7, nil
}

type detailsLogger struct{}

func (detailsLogger) Info(string, map[string]any)  {}
func (detailsLogger) Debug(string, map[string]any) {}
func (detailsLogger) Warn(string, map[string]any)  {}
func (detailsLogger) Error(string, map[string]any) {}

func TestListFilmsDetails(t *testing.T) {
	description := "A film description"
	duration := int16(120)
	for _, query := range []string{"", "&details=false", "&details=true"} {
		t.Run(query, func(t *testing.T) {
			repo := &detailsRepo{films: []domain.Film{{ID: 1, Title: "Film", Description: &description, FilmType: domain.FilmTypeMovie, DurationMin: &duration, AgeLimit: 16}}}
			uc, err := usecase.NewListFilmsUseCase(repo, detailsLogger{})
			if err != nil {
				t.Fatal(err)
			}
			h := &Handler{listFilms: uc, appLogger: detailsLogger{}}
			rec := httptest.NewRecorder()
			h.ListFilms(rec, httptest.NewRequest("GET", "/api/films?limit=2&offset=3"+query, nil))
			var body struct {
				Films  []map[string]any `json:"films"`
				Total  int              `json:"total"`
				Limit  int              `json:"limit"`
				Offset int              `json:"offset"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 200 || len(body.Films) != 1 || body.Total != 7 || body.Limit != 2 || body.Offset != 3 || repo.calls != 1 {
				t.Fatalf("unexpected response or repository calls: %s, calls=%d", rec.Body.String(), repo.calls)
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
