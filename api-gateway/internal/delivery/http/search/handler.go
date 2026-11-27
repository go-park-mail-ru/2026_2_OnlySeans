package search

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/searchquery"
	catalog "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	persons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

type catalogClient interface {
	Suggest(context.Context, *catalog.SearchSuggestionsRequest, ...grpc.CallOption) (*catalog.SearchSuggestionsResponse, error)
	SearchFilms(context.Context, *catalog.SearchFilmsRequest, ...grpc.CallOption) (*catalog.SearchFilmsResponse, error)
	CountFilms(context.Context, *catalog.CountFilmsRequest, ...grpc.CallOption) (*catalog.CountFilmsResponse, error)
}

type personsClient interface {
	SearchPersons(context.Context, *persons.SearchPersonsRequest, ...grpc.CallOption) (*persons.SearchPersonsResponse, error)
	CountPersons(context.Context, *persons.CountPersonsRequest, ...grpc.CallOption) (*persons.CountPersonsResponse, error)
}

type Handler struct {
	catalog   catalogClient
	persons   personsClient
	appLogger logger.Logger
}

func NewHandler(catalog catalogClient, persons personsClient, appLogger logger.Logger) *Handler {
	return &Handler{catalog: catalog, persons: persons, appLogger: appLogger}
}

func (h *Handler) GetSearchSuggestions(w http.ResponseWriter, r *http.Request, params openapi.GetSearchSuggestionsParams) {
	limit := searchquery.DefaultLimit
	if params.Limit != nil {
		limit = *params.Limit
	}

	query, err := searchquery.Validate(params.Q, limit)
	if err != nil {
		h.badRequest(w, r, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := h.catalog.Suggest(ctx, &catalog.SearchSuggestionsRequest{
		Query: query, Limit: int32(limit), Match: enumParam(params.Match, filmMatch),
	})
	if err != nil {
		h.searchError(w, r, err)
		return
	}

	films := make([]openapi.SearchFilmSuggestion, 0, len(result.GetFilms()))
	for _, film := range result.GetFilms() {
		films = append(films, filmSuggestion(film))
	}

	people := make([]openapi.SearchPersonSuggestion, 0, len(result.GetPersons()))
	for _, person := range result.GetPersons() {
		people = append(people, openapi.SearchPersonSuggestion{
			Id: person.GetId(), Name: person.GetName(), PhotoUrl: person.PhotoUrl, Roles: searchRoles(person.GetRoles()),
		})
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, openapi.SearchSuggestionsResponse{
		Films: films, Persons: people, Counts: searchCounts(result.GetFilmCounts(), result.GetPersonsCount()),
	}, h.appLogger)
}

func (h *Handler) SearchFilms(w http.ResponseWriter, r *http.Request, params openapi.SearchFilmsParams) {
	limit, offset, err := pageParameters(params.Limit, params.Offset)
	if err != nil {
		h.badRequest(w, r, err.Error())
		return
	}

	criteria := filmCriteria(filmFilterParams{
		query: params.Q, match: enumParam(params.Match, filmMatch),
		genres:    facet(params.GenreInclude, params.GenreExclude, params.GenreMode),
		countries: facet(params.CountryInclude, params.CountryExclude, params.CountryMode),
		yearFrom:  params.YearFrom, yearTo: params.YearTo, releaseFrom: params.ReleaseFrom, releaseTo: params.ReleaseTo,
		ratingFrom: params.RatingFrom, ratingTo: params.RatingTo, age: params.AgeRating,
	})
	roles := roleFilter(params.RoleInclude, params.RoleExclude, params.RoleMode)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)
	var result *catalog.SearchFilmsResponse
	var people *persons.CountPersonsResponse

	group.Go(func() error {
		page, err := h.catalog.SearchFilms(ctx, &catalog.SearchFilmsRequest{
			Criteria: criteria, FilmType: enumParam(params.Type, filmType), Sort: enumParam(params.Sort, filmSort),
			Pagination: &catalog.Pagination{Limit: &limit, Offset: offset},
		})
		result = page

		return err
	})
	group.Go(func() error {
		count, err := h.persons.CountPersons(ctx, &persons.CountPersonsRequest{Query: params.Q, Roles: roles})
		people = count

		return err
	})

	if err := group.Wait(); err != nil {
		h.searchError(w, r, err)
		return
	}

	films := make([]openapi.SearchFilmResult, 0, len(result.GetFilms()))
	for _, item := range result.GetFilms() {
		film, err := filmResult(item)
		if err != nil {
			h.searchError(w, r, err)
			return
		}

		films = append(films, film)
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, openapi.SearchFilmsResponse{
		Films: films, Counts: searchCounts(result.GetCounts(), people.GetTotal()),
	}, h.appLogger)
}

func (h *Handler) SearchPersons(w http.ResponseWriter, r *http.Request, params openapi.SearchPersonsParams) {
	limit, offset, err := pageParameters(params.Limit, params.Offset)
	if err != nil {
		h.badRequest(w, r, err.Error())
		return
	}

	criteria := filmCriteria(filmFilterParams{
		query: params.Q, match: enumParam(params.Match, filmMatch),
		genres:    facet(params.GenreInclude, params.GenreExclude, params.GenreMode),
		countries: facet(params.CountryInclude, params.CountryExclude, params.CountryMode),
		yearFrom:  params.YearFrom, yearTo: params.YearTo, releaseFrom: params.ReleaseFrom, releaseTo: params.ReleaseTo,
		ratingFrom: params.RatingFrom, ratingTo: params.RatingTo, age: params.AgeRating,
	})
	roles := roleFilter(params.RoleInclude, params.RoleExclude, params.RoleMode)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)
	var result *persons.SearchPersonsResponse
	var films *catalog.CountFilmsResponse

	group.Go(func() error {
		page, err := h.persons.SearchPersons(ctx, &persons.SearchPersonsRequest{
			Query: params.Q, Roles: roles, Sort: enumParam(params.Sort, personSort),
			Limit: int32(limit), Offset: offset, IncludeWorksCount: true,
		})
		result = page

		return err
	})
	group.Go(func() error {
		count, err := h.catalog.CountFilms(ctx, &catalog.CountFilmsRequest{Criteria: criteria})
		films = count

		return err
	})

	if err := group.Wait(); err != nil {
		h.searchError(w, r, err)
		return
	}

	people := make([]openapi.SearchPersonResult, 0, len(result.GetPersons()))
	for _, person := range result.GetPersons() {
		people = append(people, openapi.SearchPersonResult{
			Id: person.GetId(), Name: person.GetName(), PhotoUrl: person.PhotoUrl,
			Roles: searchRoles(person.GetRoles()), WorksCount: person.GetWorksCount(),
		})
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, openapi.SearchPersonsResponse{
		Persons: people, Counts: searchCounts(films.GetCounts(), result.GetTotal()),
	}, h.appLogger)
}

func (h *Handler) badRequest(w http.ResponseWriter, r *http.Request, message string) {
	httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, message, h.appLogger)
}

func (h *Handler) searchError(w http.ResponseWriter, r *http.Request, err error) {
	switch status.Code(err) {
	case codes.InvalidArgument:
		h.badRequest(w, r, status.Convert(err).Message())
	case codes.DeadlineExceeded:
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusGatewayTimeout, "search timed out", h.appLogger)
	default:
		h.appLogger.ErrorContext(r.Context(), "search failed", map[string]any{"error": err})
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
	}
}
