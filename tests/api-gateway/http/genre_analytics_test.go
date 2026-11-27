package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func TestGenreFilterAndDictionary(t *testing.T) {
	limit := proto.Int64(20)
	handler, verify := testGateway(t, []rpcExpectation{
		{
			method:   "ListFilms",
			request:  &pb.ListFilmsRequest{Pagination: &pb.Pagination{Limit: limit}, GenreSlug: "drama"},
			response: &pb.ListFilmsResponse{},
		},
		{
			method:   "ListGenres",
			request:  &emptypb.Empty{},
			response: &pb.ListGenresResponse{Genres: []*pb.Genre{{Id: 1, Name: "драма", Slug: "drama"}}},
		},
	}, "admin")

	request := httptest.NewRequest(http.MethodGet, "/api/films?genre=drama", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)

	request = httptest.NewRequest(http.MethodGet, "/api/genres", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"genres":[{"id":1,"name":"драма","slug":"drama"}]}`, response.Body.String())
	verify()
}

func TestAdminGenreAnalytics(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	handler, verify := testGateway(t, []rpcExpectation{{
		method:   "TopGenres",
		request:  &pb.TopGenresRequest{From: timestamppb.New(from), To: timestamppb.New(to), Limit: 10},
		response: &pb.TopGenresResponse{Genres: []*pb.GenreOpenStat{{GenreId: 1, Name: "драма", Slug: "drama", OpenCount: 12}}},
	}}, "admin")

	request := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01", nil)
	request.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"genres":[{"genre_id":1,"name":"драма","slug":"drama","open_count":12}]}`, response.Body.String())
	verify()
}
