package search

import (
	"time"

	openapitypes "github.com/oapi-codegen/runtime/types"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
	catalog "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	persons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

func searchCounts(films *catalog.FilmSearchCounts, people int64) openapi.SearchCounts {
	movies, series := films.GetMovies(), films.GetSeries()

	return openapi.SearchCounts{Movies: movies, Series: series, Persons: people, Total: movies + series + people}
}

func filmSuggestion(film *catalog.FilmSuggestion) openapi.SearchFilmSuggestion {
	people := make([]openapi.SearchMatchedPerson, 0, len(film.GetMatchedPersons()))
	for _, person := range film.GetMatchedPersons() {
		people = append(people, openapi.SearchMatchedPerson{
			Id: person.GetId(), Name: person.GetName(), Roles: searchRoles(person.GetRoles()),
		})
	}

	return openapi.SearchFilmSuggestion{
		Id: film.GetId(), Title: film.GetTitle(), OriginalTitle: film.OriginalTitle, Year: int(film.GetYear()),
		FilmType: openapi.SearchFilmSuggestionFilmType(film.GetFilmType()), PosterUrl: film.PosterUrl,
		Genres: append([]string{}, film.GetGenres()...), TitleMatch: film.GetTitleMatch(), MatchedPersons: people,
	}
}

func filmResult(item *catalog.FilmSearchResult) (openapi.SearchFilmResult, error) {
	film := filmSuggestion(item.GetFilm())

	result := openapi.SearchFilmResult{
		Id: film.Id, Title: film.Title, OriginalTitle: film.OriginalTitle, Year: film.Year,
		FilmType: openapi.SearchFilmResultFilmType(film.FilmType), PosterUrl: film.PosterUrl,
		Genres: film.Genres, TitleMatch: film.TitleMatch, MatchedPersons: film.MatchedPersons,
		AgeLimit: openapi.SearchFilmResultAgeLimit(item.GetAgeLimit()), Rating: item.Rating,
		ReviewCount: item.GetReviewCount(), AddedAt: item.GetAddedAt().AsTime(),
	}
	if item.ReleaseDate != nil {
		date, err := time.Parse(time.DateOnly, item.GetReleaseDate())
		if err != nil {
			return openapi.SearchFilmResult{}, err
		}

		result.ReleaseDate = &openapitypes.Date{Time: date}
	}

	return result, nil
}

func searchRoles(roles []persons.SearchPersonRole) []openapi.SearchPersonRole {
	result := make([]openapi.SearchPersonRole, 0, len(roles))
	for _, role := range roles {
		switch role {
		case persons.SearchPersonRole_SEARCH_PERSON_ROLE_ACTOR:
			result = append(result, "actor")
		case persons.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR:
			result = append(result, "director")
		}
	}

	return result
}
