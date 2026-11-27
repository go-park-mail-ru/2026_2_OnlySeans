package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/storage"
)

type FilmSearchRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

var _ usecase.CatalogSearchRepository = (*FilmSearchRepo)(nil)

func NewFilmSearchRepo(db *sql.DB, appLogger logger.Logger) *FilmSearchRepo {
	return &FilmSearchRepo{db: db, appLogger: appLogger}
}

func (r *FilmSearchRepo) SearchFilms(ctx context.Context, input usecase.SearchFilmsInput) (_ usecase.SearchFilmsOutput, operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, "postgres", "film.search_filtered", nil)
	defer func() { operation.Finish(operationErr) }()

	return r.searchPage(ctx, input, true)
}

func (r *FilmSearchRepo) SuggestFilms(ctx context.Context, input usecase.SuggestFilmsInput) (_ usecase.SuggestFilmsOutput, operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, "postgres", "film.suggest", nil)
	defer func() { operation.Finish(operationErr) }()

	page, err := r.searchPage(ctx, usecase.SearchFilmsInput{
		Criteria: usecase.FilmSearchCriteria{Query: input.Query, Match: input.Match},
		Limit:    input.Limit,
	}, false)
	if err != nil {
		return usecase.SuggestFilmsOutput{}, err
	}

	films := make([]usecase.FilmSuggestion, 0, len(page.Films))
	for _, item := range page.Films {
		films = append(films, item.FilmSuggestion)
	}

	return usecase.SuggestFilmsOutput{Films: films, Counts: page.Counts}, nil
}

func (r *FilmSearchRepo) CountFilms(ctx context.Context, criteria usecase.FilmSearchCriteria) (_ usecase.FilmSearchCounts, operationErr error) {
	operation := storage.StartOperation(ctx, r.appLogger, "postgres", "film.count_matches", nil)
	defer func() { operation.Finish(operationErr) }()

	var counts usecase.FilmSearchCounts

	err := r.db.QueryRowContext(ctx, filmSearchMatches+`SELECT movies, series FROM totals`,
		filmSearchArgs(criteria, false)...).Scan(&counts.Movies, &counts.Series)
	if err != nil {
		return usecase.FilmSearchCounts{}, fmt.Errorf("count film matches: %w", err)
	}

	return counts, nil
}

func (r *FilmSearchRepo) searchPage(ctx context.Context, input usecase.SearchFilmsInput, fullCards bool) (usecase.SearchFilmsOutput, error) {
	args := append(filmSearchArgs(input.Criteria, fullCards), string(input.FilmType), input.Limit, input.Offset)
	query := filmSearchMatches + fmt.Sprintf(filmSearchPage, filmSearchOrder(input.Sort))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return usecase.SearchFilmsOutput{}, fmt.Errorf("query film search page: %w", err)
	}
	defer func() { _ = rows.Close() }()

	output := usecase.SearchFilmsOutput{Films: make([]usecase.FilmSearchResult, 0)}
	for rows.Next() {
		item, err := scanFilmSearchResult(rows, &output.Counts)
		if err != nil {
			return usecase.SearchFilmsOutput{}, err
		}

		if item != nil {
			output.Films = append(output.Films, *item)
		}
	}

	if err := rows.Err(); err != nil {
		return usecase.SearchFilmsOutput{}, fmt.Errorf("read film search page: %w", err)
	}

	return output, nil
}

func scanFilmSearchResult(row rowScanner, counts *usecase.FilmSearchCounts) (*usecase.FilmSearchResult, error) {
	var (
		result         usecase.FilmSearchResult
		id             *domain.FilmID
		genres, people []byte
	)

	film := &result.Film

	err := row.Scan(&counts.Movies, &counts.Series,
		&id, &film.Title, &film.OriginalTitle, &film.FilmType, &film.ReleaseYear, &film.DurationMin, &film.AgeLimit, &film.Description,
		&film.PosterURL, &film.PosterStorageKey, &film.TrailerURL, &genres, &result.TitleMatch, &people,
		&result.Rating.AverageScore, &result.Rating.Votes, &result.ReviewCount, &result.ReleaseDate, &result.AddedAt)
	if err != nil {
		return nil, fmt.Errorf("scan film search result: %w", err)
	}

	// An empty page still has a row with the totals and a null film ID.
	if id == nil {
		return nil, nil
	}

	film.ID = *id
	result.Rating.FilmID = *id

	if err := json.Unmarshal(genres, &film.Genres); err != nil {
		return nil, fmt.Errorf("decode film search genres: %w", err)
	}

	if err := json.Unmarshal(people, &result.MatchedPersons); err != nil {
		return nil, fmt.Errorf("decode film search credits: %w", err)
	}

	return &result, nil
}

func filmSearchArgs(criteria usecase.FilmSearchCriteria, fullCards bool) []any {
	prefix, contains := searchquery.Patterns(criteria.Query)

	return []any{
		criteria.Query, prefix, contains, criteria.Match != usecase.FilmSearchMatchTitle,
		criteria.Genres.Include, criteria.Genres.Exclude, criteria.Genres.Mode == searchquery.FilterAny,
		criteria.Countries.Include, criteria.Countries.Exclude, criteria.Countries.Mode == searchquery.FilterAny,
		criteria.YearFrom, criteria.YearTo, criteria.ReleaseFrom, criteria.ReleaseTo,
		criteria.RatingFrom, criteria.RatingTo, criteria.AgeRatings, fullCards,
	}
}

// SQL fragments are selected from fixed literals. Client text is always bound separately.
func filmSearchOrder(sort usecase.FilmSearchSort) string {
	switch sort {
	case usecase.FilmSearchSortPopularity:
		return "m.review_count DESC"
	case usecase.FilmSearchSortRating:
		return "m.average_score DESC NULLS LAST"
	case usecase.FilmSearchSortReleaseDate:
		return "m.release_date DESC NULLS LAST"
	case usecase.FilmSearchSortAddedAt:
		return "m.created_at DESC"
	default:
		return "m.match_kind, m.score DESC"
	}
}
