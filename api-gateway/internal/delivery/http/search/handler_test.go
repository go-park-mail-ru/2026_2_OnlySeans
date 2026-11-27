package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	catalog "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	persons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

type searchClient struct {
	suggest     func(*catalog.SearchSuggestionsRequest) (*catalog.SearchSuggestionsResponse, error)
	searchFilms func(*catalog.SearchFilmsRequest) (*catalog.SearchFilmsResponse, error)
	countFilms  func(*catalog.CountFilmsRequest) (*catalog.CountFilmsResponse, error)
}

func (c searchClient) Suggest(_ context.Context, req *catalog.SearchSuggestionsRequest, _ ...grpc.CallOption) (*catalog.SearchSuggestionsResponse, error) {
	return c.suggest(req)
}

func (c searchClient) SearchFilms(_ context.Context, req *catalog.SearchFilmsRequest, _ ...grpc.CallOption) (*catalog.SearchFilmsResponse, error) {
	return c.searchFilms(req)
}

func (c searchClient) CountFilms(_ context.Context, req *catalog.CountFilmsRequest, _ ...grpc.CallOption) (*catalog.CountFilmsResponse, error) {
	return c.countFilms(req)
}

type peopleClient struct {
	searchPersons func(*persons.SearchPersonsRequest) (*persons.SearchPersonsResponse, error)
	countPersons  func(*persons.CountPersonsRequest) (*persons.CountPersonsResponse, error)
}

func (c peopleClient) SearchPersons(_ context.Context, req *persons.SearchPersonsRequest, _ ...grpc.CallOption) (*persons.SearchPersonsResponse, error) {
	return c.searchPersons(req)
}

func (c peopleClient) CountPersons(_ context.Context, req *persons.CountPersonsRequest, _ ...grpc.CallOption) (*persons.CountPersonsResponse, error) {
	return c.countPersons(req)
}

func TestSuggestKeepsHTTPContract(t *testing.T) {
	var request *catalog.SearchSuggestionsRequest
	client := searchClient{suggest: func(req *catalog.SearchSuggestionsRequest) (*catalog.SearchSuggestionsResponse, error) {
		request = req

		return &catalog.SearchSuggestionsResponse{
			Films:      []*catalog.FilmSuggestion{{Id: 1, Title: "Film", Year: 2010, FilmType: "movie"}},
			FilmCounts: &catalog.FilmSearchCounts{Movies: 6, Series: 1},
		}, nil
	}}
	handler := newSearchRouter(t, NewHandler(client, nil, nil))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/search/suggest?q=+%D0%9D%D0%BE%D0%BB%D0%B0%D0%BD+", nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "Нолан", request.GetQuery())
	require.EqualValues(t, 3, request.GetLimit())
	require.JSONEq(t, `{
  "counts":{"movies":6,"series":1,"persons":0,"total":7},
  "films":[{"id":1,"title":"Film","year":2010,"film_type":"movie","poster_url":null,"original_title":null,
             "genres":[],"matched_persons":[],"title_match":false}],
  "persons":[]
 }`, response.Body.String())
}

func TestSuggestRejectsInvalidParametersBeforeCallingCatalog(t *testing.T) {
	handler := newSearchRouter(t, NewHandler(nil, nil, nil))

	for _, query := range []string{"", "q=x", "q=one&q=two", "q=film&limit=0", "q=film&limit=NaN", "q=film&limit=6", "q=film&limit="} {
		t.Run(query, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/search/suggest?"+query, nil))
			require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
		})
	}
}

func TestSearchFilmsUsesGeneratedParamsAndCountsAllTabs(t *testing.T) {
	var filmRequest *catalog.SearchFilmsRequest
	var personRequest *persons.CountPersonsRequest
	films := searchClient{searchFilms: func(req *catalog.SearchFilmsRequest) (*catalog.SearchFilmsResponse, error) {
		filmRequest = req
		return &catalog.SearchFilmsResponse{Counts: &catalog.FilmSearchCounts{Movies: 5, Series: 3}}, nil
	}}
	people := peopleClient{countPersons: func(req *persons.CountPersonsRequest) (*persons.CountPersonsResponse, error) {
		personRequest = req
		return &persons.CountPersonsResponse{Total: 2}, nil
	}}
	handler := newSearchRouter(t, NewHandler(films, people, nil))
	url := "/api/search/films?q=film&type=series&limit=1&offset=40&age_rating=12&age_rating=16&" +
		"genre_include=drama&genre_include=crime&genre_mode=any&role_include=director&release_from=2020-02-29"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, url, nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, catalog.FilmSearchType_FILM_SEARCH_TYPE_SERIES, filmRequest.GetFilmType())
	require.EqualValues(t, 40, filmRequest.GetPagination().GetOffset())
	require.Equal(t, []int32{12, 16}, filmRequest.GetCriteria().GetAgeRatings())
	require.Equal(t, []string{"drama", "crime"}, filmRequest.GetCriteria().GetGenres().GetInclude())
	require.Equal(t, catalog.FilmFilterMode_FILM_FILTER_MODE_ANY, filmRequest.GetCriteria().GetGenres().GetMode())
	require.Equal(t, "2020-02-29", filmRequest.GetCriteria().GetReleaseFrom())
	require.Equal(t, []persons.SearchPersonRole{persons.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR}, personRequest.GetRoles().GetInclude())
	require.JSONEq(t, `{"films":[],"counts":{"movies":5,"series":3,"persons":2,"total":10}}`, response.Body.String())
}

func TestSearchPersonsRequestsFilmCountsAndWorksCount(t *testing.T) {
	var filmRequest *catalog.CountFilmsRequest
	var personRequest *persons.SearchPersonsRequest
	films := searchClient{countFilms: func(req *catalog.CountFilmsRequest) (*catalog.CountFilmsResponse, error) {
		filmRequest = req
		return &catalog.CountFilmsResponse{Counts: &catalog.FilmSearchCounts{Movies: 1}}, nil
	}}
	people := peopleClient{searchPersons: func(req *persons.SearchPersonsRequest) (*persons.SearchPersonsResponse, error) {
		personRequest = req
		works := int64(4)

		return &persons.SearchPersonsResponse{
			Total: 2,
			Persons: []*persons.PersonSuggestion{
				{Id: 6, Name: "Director", WorksCount: &works, Roles: []persons.SearchPersonRole{persons.SearchPersonRole_SEARCH_PERSON_ROLE_DIRECTOR}},
			},
		}, nil
	}}
	handler := newSearchRouter(t, NewHandler(films, people, nil))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/search/persons?q=film&country_exclude=GB&sort=added_at&limit=1", nil))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, []string{"GB"}, filmRequest.GetCriteria().GetCountries().GetExclude())
	require.True(t, personRequest.GetIncludeWorksCount())
	require.Equal(t, persons.PersonSearchSort_PERSON_SEARCH_SORT_ADDED_AT, personRequest.GetSort())
	var body openapi.SearchPersonsResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, openapi.SearchCounts{Movies: 1, Persons: 2, Total: 3}, body.Counts)
	require.Len(t, body.Persons, 1)
	require.EqualValues(t, 4, body.Persons[0].WorksCount)
	require.Equal(t, []openapi.SearchPersonRole{"director"}, body.Persons[0].Roles)
}

func newSearchRouter(t *testing.T, search *Handler) http.Handler {
	t.Helper()

	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
	require.NoError(t, err)

	handler, err := router.NewRouter(router.Handlers{Search: search}, middleware.NewAuthMiddleware(nil, nil, appLogger), nil, appLogger)
	require.NoError(t, err)

	return handler
}
