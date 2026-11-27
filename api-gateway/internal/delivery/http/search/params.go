package search

import (
	"math"
	"time"

	openapitypes "github.com/oapi-codegen/runtime/types"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	catalog "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	persons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

var (
	filmMatch = map[string]catalog.FilmSearchMatch{
		"all":   catalog.FilmSearchMatch_FILM_SEARCH_MATCH_ALL,
		"title": catalog.FilmSearchMatch_FILM_SEARCH_MATCH_TITLE,
	}
	filmType = map[string]catalog.FilmSearchType{
		"movie":  catalog.FilmSearchType_FILM_SEARCH_TYPE_MOVIE,
		"series": catalog.FilmSearchType_FILM_SEARCH_TYPE_SERIES,
	}
	filmSort = map[string]catalog.FilmSearchSort{
		"relevance":    catalog.FilmSearchSort_FILM_SEARCH_SORT_RELEVANCE,
		"popularity":   catalog.FilmSearchSort_FILM_SEARCH_SORT_POPULARITY,
		"rating":       catalog.FilmSearchSort_FILM_SEARCH_SORT_RATING,
		"release_date": catalog.FilmSearchSort_FILM_SEARCH_SORT_RELEASE_DATE,
		"added_at":     catalog.FilmSearchSort_FILM_SEARCH_SORT_ADDED_AT,
	}
	facetMode = map[string]catalog.FilmFilterMode{
		"all": catalog.FilmFilterMode_FILM_FILTER_MODE_ALL,
		"any": catalog.FilmFilterMode_FILM_FILTER_MODE_ANY,
	}
	personSort = map[string]persons.PersonSearchSort{
		"relevance": persons.PersonSearchSort_PERSON_SEARCH_SORT_RELEVANCE,
		"added_at":  persons.PersonSearchSort_PERSON_SEARCH_SORT_ADDED_AT,
	}
	personMode = map[string]persons.PersonRoleFilterMode{
		"all": persons.PersonRoleFilterMode_PERSON_ROLE_FILTER_MODE_ALL,
		"any": persons.PersonRoleFilterMode_PERSON_ROLE_FILTER_MODE_ANY,
	}
	personRole = map[string]persons.SearchPersonRole{
		"actor":    persons.SearchPersonRole_SEARCH_PERSON_ROLE_ACTOR,
		"director": persons.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR,
	}
)

type filmFilterParams struct {
	query                  string
	match                  catalog.FilmSearchMatch
	genres, countries      *catalog.FilmFacetFilter
	yearFrom, yearTo       *int
	releaseFrom, releaseTo *openapitypes.Date
	ratingFrom, ratingTo   *float64
	age                    *[]int
}

func filmCriteria(params filmFilterParams) *catalog.FilmSearchCriteria {
	ages := make([]int32, 0, len(optional(params.age)))
	for _, age := range optional(params.age) {
		ages = append(ages, searchInt32(age))
	}

	return &catalog.FilmSearchCriteria{
		Query: params.query, Match: params.match, Genres: params.genres, Countries: params.countries,
		YearFrom: optionalInt32(params.yearFrom), YearTo: optionalInt32(params.yearTo),
		ReleaseFrom: calendarDate(params.releaseFrom), ReleaseTo: calendarDate(params.releaseTo),
		RatingFrom: params.ratingFrom, RatingTo: params.ratingTo, AgeRatings: ages,
	}
}

func optional[T any](value *T) T {
	if value != nil {
		return *value
	}

	var zero T

	return zero
}

func enumParam[S ~string, E ~int32](value *S, options map[string]E) E {
	if value == nil {
		return 0
	}

	if parsed, ok := options[string(*value)]; ok {
		return parsed
	}

	// Keep an invalid explicit selection distinct from an omitted parameter.
	return -1
}

func facet[S ~string](include, exclude *[]string, mode *S) *catalog.FilmFacetFilter {
	return &catalog.FilmFacetFilter{Include: optional(include), Exclude: optional(exclude), Mode: enumParam(mode, facetMode)}
}

func roleFilter[S ~string](include, exclude *[]string, mode *S) *persons.PersonRoleFilter {
	return &persons.PersonRoleFilter{
		Include: roleParams(optional(include)), Exclude: roleParams(optional(exclude)), Mode: enumParam(mode, personMode),
	}
}

func roleParams(in []string) []persons.SearchPersonRole {
	roles := make([]persons.SearchPersonRole, 0, len(in))
	for _, role := range in {
		roles = append(roles, enumParam(&role, personRole))
	}

	return roles
}

func calendarDate(value *openapitypes.Date) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format(time.DateOnly)

	return &formatted
}

func optionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}

	converted := searchInt32(*value)

	return &converted
}

func searchInt32(value int) int32 {
	if value < math.MinInt32 || value > math.MaxInt32 {
		// The service will reject this value as an invalid year or age.
		return math.MinInt32
	}

	return int32(value)
}

func pageParameters(limit, offset *int) (int64, int64, error) {
	size := 20
	if limit != nil {
		size = *limit
	}

	skip := int64(optional(offset))

	return int64(size), skip, searchquery.ValidatePage(size, skip)
}
