package search_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	searchusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/search"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

type searchRepository struct {
	queries []string
	limits  []int
}

func (r *searchRepository) SearchFilms(_ context.Context, query string, limit int) ([]domain.SearchFilm, error) {
	r.queries = append(r.queries, query)
	r.limits = append(r.limits, limit)

	return []domain.SearchFilm{{ID: 1}}, nil
}

func (r *searchRepository) SearchPersons(_ context.Context, query string, limit int) ([]domain.SearchPerson, error) {
	r.queries = append(r.queries, query)
	r.limits = append(r.limits, limit)

	return []domain.SearchPerson{{ID: 2}}, nil
}

func TestSearchSuggestionsInputValidation(t *testing.T) {
	cases := []struct {
		name  string
		query string
		limit int
		want  error
	}{
		{"whitespace", " \t\n", 3, domain.ErrInvalidSearchQuery},
		{"short Unicode query", "я", 3, domain.ErrInvalidSearchQuery},
		{"long Unicode query", strings.Repeat("я", 201), 3, domain.ErrInvalidSearchQuery},
		{"zero limit", "кино", 0, domain.ErrInvalidSearchLimit},
		{"negative limit", "кино", -1, domain.ErrInvalidSearchLimit},
		{"limit above contract", "кино", 6, domain.ErrInvalidSearchLimit},
		{"minimum boundaries", " \tяя\n", 1, nil},
		{"maximum boundaries", " " + strings.Repeat("я", 200) + " ", 5, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &searchRepository{}

			uc, err := searchusecase.NewSearchSuggestionUseCase(repo, testutil.DiscardLogger())
			if err != nil {
				t.Fatal(err)
			}

			input := searchusecase.NewSearchSuggestionUsecaseInput(tc.query, tc.limit)

			output, err := uc.Run(context.Background(), *input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("Run error = %v, want %v", err, tc.want)
			}

			if input.Query != tc.query || input.Limit != tc.limit {
				t.Fatal("Run modified the caller's input")
			}

			if tc.want != nil {
				if output != nil || len(repo.queries) != 0 {
					t.Fatal("invalid input must not reach the repository or produce output")
				}

				return
			}

			if len(repo.queries) != 2 {
				t.Fatalf("repository calls = %d, want 2", len(repo.queries))
			}

			for i, query := range repo.queries {
				if query != strings.TrimSpace(tc.query) || repo.limits[i] != tc.limit {
					t.Fatal("repository did not receive normalized input")
				}
			}

			if output == nil || len(output.FilmSuggestions) != 1 || len(output.PersonSuggestions) != 1 {
				t.Fatalf("unexpected output: %+v", output)
			}
		})
	}
}
