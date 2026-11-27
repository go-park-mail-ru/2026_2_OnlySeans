package grpc

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	pbpersons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

type SearchHandler struct {
	pb.UnimplementedCatalogSearchServiceServer
	films   *usecase.FilmSearchUseCase
	persons pbpersons.PersonSearchServiceClient
}

func NewSearchHandler(films *usecase.FilmSearchUseCase, persons pbpersons.PersonSearchServiceClient) *SearchHandler {
	return &SearchHandler{films: films, persons: persons}
}

func (h *SearchHandler) Suggest(ctx context.Context, req *pb.SearchSuggestionsRequest) (*pb.SearchSuggestionsResponse, error) {
	input := usecase.SuggestFilmsInput{Query: req.GetQuery(), Match: filmMatch(req.GetMatch()), Limit: int(req.GetLimit())}
	if err := input.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	group, ctx := errgroup.WithContext(ctx)
	var matches usecase.SuggestFilmsOutput
	var people *pbpersons.SearchPersonsResponse

	group.Go(func() error {
		result, err := h.films.SuggestFilms(ctx, input)
		matches = result

		return searchError(err)
	})
	group.Go(func() error {
		result, err := h.persons.SearchPersons(ctx, &pbpersons.SearchPersonsRequest{
			Query: req.GetQuery(), Limit: req.GetLimit(),
		})
		people = result

		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	films := make([]*pb.FilmSuggestion, 0, len(matches.Films))
	for _, match := range matches.Films {
		films = append(films, toSearchFilm(match))
	}

	return &pb.SearchSuggestionsResponse{
		Films: films, Persons: people.GetPersons(), FilmCounts: toFilmCounts(matches.Counts), PersonsCount: people.GetTotal(),
	}, nil
}

func (h *SearchHandler) SearchFilms(ctx context.Context, req *pb.SearchFilmsRequest) (*pb.SearchFilmsResponse, error) {
	criteria, err := filmCriteria(req.GetCriteria())
	if err != nil {
		return nil, err
	}

	page := req.GetPagination()

	limit := int64(usecase.DefaultLimit)
	if page != nil && page.Limit != nil {
		limit = page.GetLimit()
	}

	if limit < 1 || limit > searchquery.MaxPageLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be an integer from 1 to %d", searchquery.MaxPageLimit)
	}

	result, err := h.films.SearchFilms(ctx, usecase.SearchFilmsInput{
		Criteria: criteria, FilmType: searchFilmType(req.GetFilmType()), Sort: searchFilmSort(req.GetSort()),
		Limit: int(limit), Offset: page.GetOffset(),
	})
	if err != nil {
		return nil, searchError(err)
	}

	films := make([]*pb.FilmSearchResult, 0, len(result.Films))
	for _, match := range result.Films {
		film := &pb.FilmSearchResult{
			Film: toSearchFilm(match.FilmSuggestion), AgeLimit: int32(match.Film.AgeLimit),
			ReviewCount: match.ReviewCount, AddedAt: timestamppb.New(match.AddedAt),
		}
		if match.ReleaseDate != nil {
			date := match.ReleaseDate.Format(time.DateOnly)
			film.ReleaseDate = &date
		}

		if match.Rating.Votes > 0 {
			film.Rating = &match.Rating.AverageScore
		}

		films = append(films, film)
	}

	return &pb.SearchFilmsResponse{Films: films, Counts: toFilmCounts(result.Counts)}, nil
}

func (h *SearchHandler) CountFilms(ctx context.Context, req *pb.CountFilmsRequest) (*pb.CountFilmsResponse, error) {
	criteria, err := filmCriteria(req.GetCriteria())
	if err != nil {
		return nil, err
	}

	result, err := h.films.CountFilms(ctx, usecase.CountFilmsInput{Criteria: criteria})
	if err != nil {
		return nil, searchError(err)
	}

	return &pb.CountFilmsResponse{Counts: toFilmCounts(result.Counts)}, nil
}

func filmCriteria(in *pb.FilmSearchCriteria) (usecase.FilmSearchCriteria, error) {
	if in == nil {
		return usecase.FilmSearchCriteria{}, nil
	}

	from, err := searchDate(in.ReleaseFrom, "release_from")
	if err != nil {
		return usecase.FilmSearchCriteria{}, err
	}

	to, err := searchDate(in.ReleaseTo, "release_to")
	if err != nil {
		return usecase.FilmSearchCriteria{}, err
	}

	return usecase.FilmSearchCriteria{
		Query: in.GetQuery(), Match: filmMatch(in.GetMatch()),
		Genres: filmFacet(in.GetGenres()), Countries: filmFacet(in.GetCountries()),
		ReleaseFrom: from, ReleaseTo: to, AgeRatings: in.GetAgeRatings(),
		YearFrom: in.YearFrom, YearTo: in.YearTo, RatingFrom: in.RatingFrom, RatingTo: in.RatingTo,
	}, nil
}

func searchDate(value *string, field string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	date, err := time.Parse(time.DateOnly, *value)
	if err != nil || date.Format(time.DateOnly) != *value || date.Year() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "%s must be a date in YYYY-MM-DD format", field)
	}

	return &date, nil
}

func filmFacet(filter *pb.FilmFacetFilter) searchquery.Filter[string] {
	mode := searchquery.FilterMode("invalid")

	switch filter.GetMode() {
	case pb.FilmFilterMode_FILM_FILTER_MODE_UNSPECIFIED, pb.FilmFilterMode_FILM_FILTER_MODE_ALL:
		mode = searchquery.FilterAll
	case pb.FilmFilterMode_FILM_FILTER_MODE_ANY:
		mode = searchquery.FilterAny
	}

	return searchquery.Filter[string]{Include: filter.GetInclude(), Exclude: filter.GetExclude(), Mode: mode}
}

func filmMatch(value pb.FilmSearchMatch) usecase.FilmSearchMatch {
	switch value {
	case pb.FilmSearchMatch_FILM_SEARCH_MATCH_UNSPECIFIED, pb.FilmSearchMatch_FILM_SEARCH_MATCH_ALL:
		return usecase.FilmSearchMatchAll
	case pb.FilmSearchMatch_FILM_SEARCH_MATCH_TITLE:
		return usecase.FilmSearchMatchTitle
	default:
		return "invalid"
	}
}

func searchFilmType(value pb.FilmSearchType) domain.FilmType {
	switch value {
	case pb.FilmSearchType_FILM_SEARCH_TYPE_UNSPECIFIED:
		return ""
	case pb.FilmSearchType_FILM_SEARCH_TYPE_MOVIE:
		return domain.FilmTypeMovie
	case pb.FilmSearchType_FILM_SEARCH_TYPE_SERIES:
		return domain.FilmTypeSeries
	default:
		return "invalid"
	}
}

func searchFilmSort(value pb.FilmSearchSort) usecase.FilmSearchSort {
	switch value {
	case pb.FilmSearchSort_FILM_SEARCH_SORT_UNSPECIFIED, pb.FilmSearchSort_FILM_SEARCH_SORT_RELEVANCE:
		return usecase.FilmSearchSortRelevance
	case pb.FilmSearchSort_FILM_SEARCH_SORT_POPULARITY:
		return usecase.FilmSearchSortPopularity
	case pb.FilmSearchSort_FILM_SEARCH_SORT_RATING:
		return usecase.FilmSearchSortRating
	case pb.FilmSearchSort_FILM_SEARCH_SORT_RELEASE_DATE:
		return usecase.FilmSearchSortReleaseDate
	case pb.FilmSearchSort_FILM_SEARCH_SORT_ADDED_AT:
		return usecase.FilmSearchSortAddedAt
	default:
		return "invalid"
	}
}

func toSearchFilm(match usecase.FilmSuggestion) *pb.FilmSuggestion {
	film := match.Film

	genres := make([]string, 0, len(film.Genres))
	for _, genre := range film.Genres {
		genres = append(genres, genre.Name)
	}

	people := make([]*pb.SearchMatchedPerson, 0, len(match.MatchedPersons))
	for _, person := range match.MatchedPersons {
		roles := make([]pbpersons.SearchPersonRole, 0, len(person.Roles))
		for _, role := range person.Roles {
			switch role {
			case "actor":
				roles = append(roles, pbpersons.SearchPersonRole_SEARCH_PERSON_ROLE_ACTOR)
			case "director":
				roles = append(roles, pbpersons.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR)
			}
		}

		people = append(people, &pb.SearchMatchedPerson{Id: person.ID, Name: person.Name, Roles: roles})
	}

	return &pb.FilmSuggestion{
		Id: int64(film.ID), Title: film.Title, OriginalTitle: film.OriginalTitle, Year: int32(film.ReleaseYear),
		FilmType: string(film.FilmType), PosterUrl: film.PosterURL, Genres: genres,
		TitleMatch: match.TitleMatch, MatchedPersons: people,
	}
}

func toFilmCounts(counts usecase.FilmSearchCounts) *pb.FilmSearchCounts {
	return &pb.FilmSearchCounts{Movies: counts.Movies, Series: counts.Series}
}

func searchError(err error) error {
	if errors.Is(err, searchquery.ErrInvalidInput) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return toGRPCError(err)
}
