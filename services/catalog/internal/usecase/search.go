package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

type CatalogSearchRepository interface {
	// SearchFilms and SuggestFilms return counts before applying the page or suggestion limit.
	SearchFilms(context.Context, SearchFilmsInput) (SearchFilmsOutput, error)
	SuggestFilms(context.Context, SuggestFilmsInput) (SuggestFilmsOutput, error)
	CountFilms(context.Context, FilmSearchCriteria) (FilmSearchCounts, error)
}

type FilmSearchUseCase struct {
	repo      CatalogSearchRepository
	posters   PosterURLStorage
	appLogger logger.Logger
}

func NewFilmSearchUseCase(repo CatalogSearchRepository, posters PosterURLStorage, appLogger logger.Logger) *FilmSearchUseCase {
	return &FilmSearchUseCase{repo: repo, posters: posters, appLogger: appLogger}
}

func (s *FilmSearchUseCase) SearchFilms(ctx context.Context, input SearchFilmsInput) (SearchFilmsOutput, error) {
	input.Criteria.Query = strings.TrimSpace(input.Criteria.Query)
	if err := input.Validate(); err != nil {
		return SearchFilmsOutput{}, fmt.Errorf("%w: %w", searchquery.ErrInvalidInput, err)
	}

	input.Criteria.Match = defaultFilmSearchMatch(input.Criteria.Match)
	if input.Sort == "" {
		input.Sort = FilmSearchSortRelevance
	}

	result, err := s.repo.SearchFilms(ctx, input)
	if err != nil {
		return SearchFilmsOutput{}, fmt.Errorf("search films: %w", err)
	}

	for i := range result.Films {
		if err := attachSearchPosterURL(ctx, s.posters, &result.Films[i].Film); err != nil {
			return SearchFilmsOutput{}, err
		}
	}

	s.appLogger.DebugContext(ctx, "film search completed", map[string]any{
		"returned": len(result.Films),
		"movies":   result.Counts.Movies,
		"series":   result.Counts.Series,
	})

	return result, nil
}

func (s *FilmSearchUseCase) SuggestFilms(ctx context.Context, input SuggestFilmsInput) (SuggestFilmsOutput, error) {
	input.Query = strings.TrimSpace(input.Query)
	if err := input.Validate(); err != nil {
		return SuggestFilmsOutput{}, fmt.Errorf("%w: %w", searchquery.ErrInvalidInput, err)
	}

	input.Match = defaultFilmSearchMatch(input.Match)

	result, err := s.repo.SuggestFilms(ctx, input)
	if err != nil {
		return SuggestFilmsOutput{}, fmt.Errorf("suggest films: %w", err)
	}

	for i := range result.Films {
		if err := attachSearchPosterURL(ctx, s.posters, &result.Films[i].Film); err != nil {
			return SuggestFilmsOutput{}, err
		}
	}

	s.appLogger.DebugContext(ctx, "film suggestions completed", map[string]any{
		"returned": len(result.Films),
		"movies":   result.Counts.Movies,
		"series":   result.Counts.Series,
	})

	return result, nil
}

func (s *FilmSearchUseCase) CountFilms(ctx context.Context, input CountFilmsInput) (CountFilmsOutput, error) {
	input.Criteria.Query = strings.TrimSpace(input.Criteria.Query)
	if err := input.Validate(); err != nil {
		return CountFilmsOutput{}, fmt.Errorf("%w: %w", searchquery.ErrInvalidInput, err)
	}

	input.Criteria.Match = defaultFilmSearchMatch(input.Criteria.Match)

	counts, err := s.repo.CountFilms(ctx, input.Criteria)
	if err != nil {
		return CountFilmsOutput{}, fmt.Errorf("count films: %w", err)
	}

	s.appLogger.DebugContext(ctx, "film search count completed", map[string]any{
		"movies": counts.Movies,
		"series": counts.Series,
	})

	return CountFilmsOutput{Counts: counts}, nil
}

func defaultFilmSearchMatch(match FilmSearchMatch) FilmSearchMatch {
	if match == "" {
		return FilmSearchMatchAll
	}

	return match
}

func attachSearchPosterURL(ctx context.Context, posters PosterURLStorage, film *domain.Film) error {
	if film.PosterURL != nil || film.PosterStorageKey == nil {
		return nil
	}

	url, err := posters.GetURL(ctx, *film.PosterStorageKey)
	if errors.Is(err, domain.ErrFileNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("get search film poster: %w", err)
	}

	film.PosterURL = &url

	return nil
}
