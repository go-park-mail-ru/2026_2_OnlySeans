package usecase

import (
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
)

type FilmSearchMatch string

const (
	FilmSearchMatchAll   FilmSearchMatch = "all"
	FilmSearchMatchTitle FilmSearchMatch = "title"
)

type FilmSearchSort string

const (
	FilmSearchSortRelevance   FilmSearchSort = "relevance"
	FilmSearchSortPopularity  FilmSearchSort = "popularity"
	FilmSearchSortRating      FilmSearchSort = "rating"
	FilmSearchSortReleaseDate FilmSearchSort = "release_date"
	FilmSearchSortAddedAt     FilmSearchSort = "added_at"
)

// FilmSearchCriteria defines which titles qualify for both a page and counts.
type FilmSearchCriteria struct {
	Query       string
	Match       FilmSearchMatch
	Genres      searchquery.Filter[string]
	Countries   searchquery.Filter[string]
	YearFrom    *int32
	YearTo      *int32
	ReleaseFrom *time.Time // Calendar date at midnight UTC.
	ReleaseTo   *time.Time
	RatingFrom  *float64
	RatingTo    *float64
	AgeRatings  []int32
}

type SearchFilmsInput struct {
	Criteria FilmSearchCriteria
	FilmType domain.FilmType // An empty type includes both movies and series in the page.
	Sort     FilmSearchSort
	Limit    int
	Offset   int64
}

type SearchFilmsOutput struct {
	Films  []FilmSearchResult
	Counts FilmSearchCounts
}

type CountFilmsInput struct {
	Criteria FilmSearchCriteria
}

type CountFilmsOutput struct {
	Counts FilmSearchCounts
}

type SuggestFilmsInput struct {
	Query string
	Match FilmSearchMatch
	Limit int
}

type SuggestFilmsOutput struct {
	Films  []FilmSuggestion
	Counts FilmSearchCounts
}

type FilmSearchCounts struct {
	Movies int64
	Series int64
}

type FilmSuggestion struct {
	Film           domain.Film
	TitleMatch     bool
	MatchedPersons []SearchMatchedPerson
}

type SearchMatchedPerson struct {
	ID    int64
	Name  string
	Roles []string
}

type FilmSearchResult struct {
	FilmSuggestion
	Rating      domain.FilmRatingSummary
	ReviewCount int64
	ReleaseDate *time.Time
	AddedAt     time.Time
}
