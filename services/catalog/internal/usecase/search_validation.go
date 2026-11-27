package usecase

import (
	"fmt"
	"regexp"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

var (
	searchGenreSlug   = regexp.MustCompile(`^[a-z0-9-]{2,64}$`)
	searchCountryCode = regexp.MustCompile(`^[A-Z]{2}$`)
)

func (in SearchFilmsInput) Validate() error {
	if err := in.Criteria.Validate(); err != nil {
		return err
	}

	switch in.FilmType {
	case "", domain.FilmTypeMovie, domain.FilmTypeSeries:
	default:
		return fmt.Errorf("invalid film type %q", in.FilmType)
	}

	switch in.Sort {
	case "", FilmSearchSortRelevance, FilmSearchSortPopularity, FilmSearchSortRating,
		FilmSearchSortReleaseDate, FilmSearchSortAddedAt:
	default:
		return fmt.Errorf("invalid film sort %q", in.Sort)
	}

	return searchquery.ValidatePage(in.Limit, in.Offset)
}

func (in CountFilmsInput) Validate() error {
	return in.Criteria.Validate()
}

func (in SuggestFilmsInput) Validate() error {
	if _, err := searchquery.Validate(in.Query, in.Limit); err != nil {
		return err
	}

	return in.Match.validate()
}

func (criteria FilmSearchCriteria) Validate() error {
	if _, err := searchquery.NormalizeQuery(criteria.Query); err != nil {
		return err
	}

	if err := criteria.Match.validate(); err != nil {
		return err
	}

	if err := criteria.Genres.Validate(50, searchGenreSlug.MatchString); err != nil {
		return fmt.Errorf("genres: %w", err)
	}

	if err := criteria.Countries.Validate(50, searchCountryCode.MatchString); err != nil {
		return fmt.Errorf("countries: %w", err)
	}

	if !validSearchYear(criteria.YearFrom) || !validSearchYear(criteria.YearTo) {
		return fmt.Errorf("year bounds must be from %d to %d", domain.MinFilmProductionYear, domain.MaxFilmProductionYear)
	}

	if criteria.YearFrom != nil && criteria.YearTo != nil && *criteria.YearFrom > *criteria.YearTo {
		return fmt.Errorf("year_from must be less than or equal to year_to")
	}

	if criteria.ReleaseFrom != nil && criteria.ReleaseTo != nil && criteria.ReleaseFrom.After(*criteria.ReleaseTo) {
		return fmt.Errorf("release_from must be less than or equal to release_to")
	}

	if !validSearchRating(criteria.RatingFrom) || !validSearchRating(criteria.RatingTo) {
		return fmt.Errorf("rating bounds must be from %d to %d", domain.MinFilmRating, domain.MaxFilmRating)
	}

	if criteria.RatingFrom != nil && criteria.RatingTo != nil && *criteria.RatingFrom > *criteria.RatingTo {
		return fmt.Errorf("rating_from must be less than or equal to rating_to")
	}

	if err := searchquery.ValidateValues(criteria.AgeRatings, 5, domain.ValidFilmAgeRating); err != nil {
		return fmt.Errorf("age_rating: %w", err)
	}

	return nil
}

func (match FilmSearchMatch) validate() error {
	switch match {
	case "", FilmSearchMatchAll, FilmSearchMatchTitle:
		return nil
	default:
		return fmt.Errorf("invalid film match %q", match)
	}
}

func validSearchYear(year *int32) bool {
	return year == nil || (*year >= domain.MinFilmProductionYear && *year <= domain.MaxFilmProductionYear)
}

func validSearchRating(rating *float64) bool {
	return rating == nil || (*rating >= float64(domain.MinFilmRating) && *rating <= float64(domain.MaxFilmRating))
}
