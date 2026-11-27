package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	fileshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/files"
	filmshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/films"
	libraryhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/library"
	personshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/persons"
	ratingshttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/authclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	personsclient "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/personscleint"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	pbpersons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
	pbuser "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
)

type authStub struct {
	authclient.AuthClient
	role string
}

func (s authStub) CheckSession(context.Context, string) (authclient.SessionInfo, error) {
	return authclient.SessionInfo{
		UserID: 7,
		Role:   s.role,
	}, nil
}

type userStub struct {
	userclient.UserClient
	role string
}

func (s userStub) GetByID(context.Context, int64) (*pbuser.User, error) {
	return &pbuser.User{
		Id:   7,
		Role: s.role,
	}, nil
}

type personStub struct {
	personsclient.PersonClient
}

func (personStub) GetPersonCatalog(context.Context, int64) (*pbpersons.GetPersonCatalogResponse, error) {
	return &pbpersons.GetPersonCatalogResponse{
		Person: &pbpersons.CatalogPerson{
			Id:   1,
			Name: "Person",
		},
		Films: []*pbpersons.FilmCredit{{
			FilmId:    42,
			Role:      "actor",
			Character: "Hero",
		}},
	}, nil
}

func (personStub) GetFilmPersons(_ context.Context, filmID int64) (*pbpersons.GetFilmPersonsResponse, error) {
	if filmID != 42 {
		return nil, errors.New("unexpected film id")
	}

	return &pbpersons.GetFilmPersonsResponse{}, nil
}

type rpcExpectation struct {
	method   string
	request  proto.Message
	response proto.Message
	err      error
}

func testGateway(t *testing.T, expected []rpcExpectation, role string) (http.Handler, func()) {
	t.Helper()
	var mu sync.Mutex
	called := 0
	var failures []error
	server := grpc.NewServer(grpc.UnaryInterceptor(func(_ context.Context, req any, info *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
		mu.Lock()
		defer mu.Unlock()

		if called >= len(expected) {
			failures = append(failures, fmt.Errorf("unexpected RPC %s", info.FullMethod))
			return nil, status.Error(codes.Internal, "unexpected RPC")
		}

		want := expected[called]
		called++

		if info.FullMethod != "/catalog.v1.CatalogService/"+want.method || !proto.Equal(want.request, req.(proto.Message)) {
			failures = append(failures, fmt.Errorf("unexpected RPC: %s %v, want %s %v", info.FullMethod, req, want.method, want.request))
			return nil, status.Error(codes.Internal, "unexpected RPC arguments")
		}

		return want.response, want.err
	}))
	pb.RegisterCatalogServiceServer(server, &pb.UnimplementedCatalogServiceServer{})

	listener := bufconn.Listen(1 << 20)
	served := make(chan error, 1)

	go func() { served <- server.Serve(listener) }()

	connection, err := grpc.NewClient("passthrough:///catalog",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
		server.Stop()
		require.NoError(t, listener.Close())

		err := <-served
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve catalog stub: %v", err)
		}
	})

	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
	require.NoError(t, err)

	client := catalogclient.NewCatalogClient(pb.NewCatalogServiceClient(connection))
	films, err := filmshttp.NewHandler(client, appLogger)
	require.NoError(t, err)
	ratings, err := ratingshttp.NewHandler(client, appLogger)
	require.NoError(t, err)
	posters, err := fileshttp.NewPosterHandler(client, 5<<20, appLogger)
	require.NoError(t, err)
	people, err := personshttp.NewCatalogHandler(personStub{}, films, appLogger)
	require.NoError(t, err)

	auth := authStub{
		role: role,
	}
	users := userStub{
		role: role,
	}
	handler, err := router.NewRouter(router.Handlers{
		Auth:           authhttp.NewAuthHandler(auth, users, appLogger),
		Films:          films,
		Ratings:        ratings,
		PersonalRating: ratingshttp.NewMyRatingHandler(client, appLogger),
		Library:        libraryhttp.NewHandler(client, appLogger),
		Posters:        posters,
		PersonCatalog:  people,
	}, middleware.NewAuthMiddleware(auth, users, appLogger), []string{"http://localhost:8000"}, appLogger)
	require.NoError(t, err)

	return handler, func() {
		mu.Lock()
		defer mu.Unlock()
		require.Empty(t, failures)
		require.Equal(t, len(expected), called)
	}
}

func posterBody(t *testing.T) (string, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	file, err := writer.CreateFormFile("poster", "poster.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	return buffer.String(), writer.FormDataContentType()
}

func TestCatalogHTTPContract(t *testing.T) {
	film := &pb.Film{
		Id:          42,
		Title:       "Film",
		FilmType:    "movie",
		ReleaseYear: 2020,
		AgeLimit:    12,
	}
	collection := &pb.Collection{
		Id:    5,
		Title: "Picks",
		Slug:  "picks",
	}
	timestamp := timestamppb.New(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	defaultPage := &pb.Pagination{
		Limit: proto.Int64(20),
	}
	multipartData, multipartType := posterBody(t)

	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		anonymous   bool
		role        string
		rpc         []rpcExpectation
		status      int
		json        string
	}{
		{
			name:   "film cards",
			method: http.MethodGet,
			path:   "/api/films",
			rpc: []rpcExpectation{{
				method: "ListFilms",
				request: &pb.ListFilmsRequest{
					Pagination: defaultPage,
				},
				response: &pb.ListFilmsResponse{
					Films: []*pb.Film{film},
					Total: 37,
				},
			}},
			status: 200,
			json:   `{"films":[{"id":42,"title":"Film","release_year":2020,"genres":[]}],"total":37,"limit":20,"offset":0}`,
		},
		{
			name:   "detailed films",
			method: http.MethodGet,
			path:   "/api/films?details=true&limit=2&offset=4",
			rpc: []rpcExpectation{{
				method: "ListFilms",
				request: &pb.ListFilmsRequest{
					Pagination: &pb.Pagination{
						Limit:  proto.Int64(2),
						Offset: 4,
					},
				},
				response: &pb.ListFilmsResponse{
					Films: []*pb.Film{film},
					Total: 37,
				},
			}},
			status: 200,
			json:   `{"films":[{"id":42,"title":"Film","film_type":"movie","release_year":2020,"age_limit":12,"genres":[]}],"total":37,"limit":2,"offset":4}`,
		},
		{
			name:   "empty films",
			method: http.MethodGet,
			path:   "/api/films",
			rpc: []rpcExpectation{{
				method: "ListFilms",
				request: &pb.ListFilmsRequest{
					Pagination: defaultPage,
				},
				response: &pb.ListFilmsResponse{},
			}},
			status: 200,
			json:   `{"films":[],"total":0,"limit":20,"offset":0}`,
		},
		{
			name:   "film details",
			method: http.MethodGet,
			path:   "/api/films/42",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				response: &pb.GetFilmResponse{
					Film: film,
				},
			}},
			status: 200,
			json:   `{"film":{"id":42,"title":"Film","film_type":"movie","release_year":2020,"age_limit":12,"genres":[]}}`,
		},
		{
			name:   "create film",
			method: http.MethodPost,
			path:   "/api/films",
			body:   `{"title":"Film","film_type":"movie","release_year":2020,"age_limit":0}`,
			rpc: []rpcExpectation{{
				method: "CreateFilm",
				request: &pb.CreateFilmRequest{
					Film: &pb.FilmInput{
						Title:       "Film",
						FilmType:    "movie",
						ReleaseYear: 2020,
						AgeLimit:    proto.Int32(0),
					},
				},
				response: &pb.CreateFilmResponse{
					FilmId: 42,
				},
			}},
			status: 201,
			json:   `{"id":42}`,
		},
		{
			name:   "update film",
			method: http.MethodPut,
			path:   "/api/films/42",
			body:   `{"title":"Film","film_type":"movie","release_year":2020,"age_limit":0}`,
			rpc: []rpcExpectation{{
				method: "UpdateFilm",
				request: &pb.UpdateFilmRequest{
					FilmId: 42,
					Film: &pb.FilmInput{
						Title:       "Film",
						FilmType:    "movie",
						ReleaseYear: 2020,
						AgeLimit:    proto.Int32(0),
					},
				},
				response: &emptypb.Empty{},
			}},
			status: 204,
		},
		{
			name:   "delete film",
			method: http.MethodDelete,
			path:   "/api/films/42",
			rpc: []rpcExpectation{{
				method: "DeleteFilm",
				request: &pb.DeleteFilmRequest{
					FilmId: 42,
				},
				response: &emptypb.Empty{},
			}},
			status: 204,
		},
		{
			name:   "collections",
			method: http.MethodGet,
			path:   "/api/collections",
			rpc: []rpcExpectation{{
				method: "ListCollections",
				request: &pb.ListCollectionsRequest{
					Pagination: defaultPage,
				},
				response: &pb.ListCollectionsResponse{
					Collections: []*pb.Collection{collection},
					Total:       1,
				},
			}},
			status: 200,
			json:   `{"collections":[{"id":5,"title":"Picks","slug":"picks"}],"total":1,"limit":20,"offset":0}`,
		},
		{
			name:   "collection films",
			method: http.MethodGet,
			path:   "/api/collections/5",
			rpc: []rpcExpectation{{
				method: "GetCollection",
				request: &pb.GetCollectionRequest{
					CollectionId: 5,
					Pagination:   defaultPage,
				},
				response: &pb.GetCollectionResponse{
					Collection: collection,
					Films:      []*pb.Film{film},
					Total:      3,
				},
			}},
			status: 200,
			json:   `{"collection":{"id":5,"title":"Picks","slug":"picks","films":[{"id":42,"title":"Film","release_year":2020,"genres":[]}]},"total":3,"limit":20,"offset":0}`,
		},
		{
			name:   "set rating",
			method: http.MethodPut,
			path:   "/api/films/42/rating",
			body:   `{"score":8}`,
			rpc: []rpcExpectation{{
				method: "SetFilmRating",
				request: &pb.SetFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
					Score:  8,
				},
				response: &pb.SetFilmRatingResponse{
					Rating: &pb.FilmRating{
						FilmId:    42,
						Score:     8,
						CreatedAt: timestamp,
						UpdatedAt: timestamp,
					},
				},
			}},
			status: 200,
			json:   `{"rating":{"film_id":42,"score":8,"created_at":"2026-11-25T12:00:00Z","updated_at":"2026-11-25T12:00:00Z"}}`,
		},
		{
			name:   "rating summary",
			method: http.MethodGet,
			path:   "/api/films/42/rating",
			rpc: []rpcExpectation{{
				method: "GetFilmRating",
				request: &pb.GetFilmRatingRequest{
					FilmId: 42,
				},
				response: &pb.GetFilmRatingResponse{
					Rating: &pb.FilmRatingSummary{
						FilmId:       42,
						AverageScore: 8.5,
						Votes:        2,
					},
				},
			}},
			status: 200,
			json:   `{"rating":{"film_id":42,"average_score":8.5,"votes":2}}`,
		},
		{
			name:   "delete rating",
			method: http.MethodDelete,
			path:   "/api/films/42/rating",
			rpc: []rpcExpectation{{
				method: "DeleteFilmRating",
				request: &pb.DeleteFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
				},
				response: &emptypb.Empty{},
			}},
			status: 204,
		},
		{
			name:   "personal rating absent",
			method: http.MethodGet,
			path:   "/api/films/42/rating/me",
			rpc: []rpcExpectation{{
				method: "GetMyFilmRating",
				request: &pb.GetMyFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
				},
				response: &pb.GetMyFilmRatingResponse{},
			}},
			status: 200,
			json:   `{"rating":null}`,
		},
		{
			name:   "personal rating present",
			method: http.MethodGet,
			path:   "/api/films/42/rating/me",
			rpc: []rpcExpectation{{
				method: "GetMyFilmRating",
				request: &pb.GetMyFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
				},
				response: &pb.GetMyFilmRatingResponse{
					Score: proto.Int32(8),
				},
			}},
			status: 200,
			json:   `{"rating":{"score":8}}`,
		},
		{
			name:   "collection ratings empty page",
			method: http.MethodGet,
			path:   "/api/collections/5/ratings",
			rpc: []rpcExpectation{{
				method: "GetCollectionRatings",
				request: &pb.GetCollectionRatingsRequest{
					CollectionId: 5,
					Pagination:   defaultPage,
				},
				response: &pb.GetCollectionRatingsResponse{
					Total: 3,
				},
			}},
			status: 200,
			json:   `{"ratings":[],"total":3,"limit":20,"offset":0}`,
		},
		{
			name:   "library empty lists",
			method: http.MethodGet,
			path:   "/api/me/library",
			rpc: []rpcExpectation{{
				method: "GetLibrary",
				request: &pb.GetLibraryRequest{
					UserId: 7,
				},
				response: &pb.GetLibraryResponse{
					Library: &pb.Library{
						Favorite: []int64{42},
					},
				},
			}},
			status: 200,
			json:   `{"favorite":[42],"watchlater":[],"watched":[]}`,
		},
		{
			name:   "save library returns updated library",
			method: http.MethodPut,
			path:   "/api/me/library/lists/watched/42",
			body:   `{"saved":false}`,
			rpc: []rpcExpectation{
				{
					method: "SetSaved",
					request: &pb.SetSavedRequest{
						UserId: 7,
						List:   "watched",
						FilmId: 42,
						Saved:  false,
					},
					response: &emptypb.Empty{},
				},
				{
					method: "GetLibrary",
					request: &pb.GetLibraryRequest{
						UserId: 7,
					},
					response: &pb.GetLibraryResponse{
						Library: &pb.Library{},
					},
				},
			},
			status: 200,
			json:   `{"favorite":[],"watchlater":[],"watched":[]}`,
		},
		{
			name:   "get preferences legacy keys",
			method: http.MethodGet,
			path:   "/api/me/preferences",
			rpc: []rpcExpectation{{
				method: "GetPreferences",
				request: &pb.GetPreferencesRequest{
					UserId: 7,
				},
				response: &pb.GetPreferencesResponse{
					Preferences: &pb.Preferences{
						SidebarCollapsed: proto.Bool(false),
						CatalogView:      proto.String("grid"),
					},
				},
			}},
			status: 200,
			json:   `{"frame:sidebar-collapsed":false,"frame:catalog-view":"grid"}`,
		},
		{
			name:   "save false preference",
			method: http.MethodPut,
			path:   "/api/me/preferences",
			body:   `{"key":"frame:sidebar-collapsed","value":false}`,
			rpc: []rpcExpectation{{
				method: "SavePreference",
				request: &pb.SavePreferenceRequest{
					UserId: 7,
					Update: &pb.SavePreferenceRequest_SidebarCollapsed{
						SidebarCollapsed: false,
					},
				},
				response: &emptypb.Empty{},
			}},
			status: 204,
		},
		{
			name:        "upload poster",
			method:      http.MethodPost,
			path:        "/api/films/42/poster",
			body:        multipartData,
			contentType: multipartType,
			rpc: []rpcExpectation{{
				method: "UploadPoster",
				request: &pb.UploadPosterRequest{
					UserId:    7,
					FilmId:    42,
					ImageData: []byte("image"),
				},
				response: &pb.UploadPosterResponse{
					Filename: "poster.png",
				},
			}},
			status: 201,
			json:   `{"filename":"poster.png"}`,
		},
		{
			name:   "get poster",
			method: http.MethodGet,
			path:   "/api/films/42/poster",
			rpc: []rpcExpectation{{
				method: "GetPosterURL",
				request: &pb.GetPosterURLRequest{
					FilmId: 42,
				},
				response: &pb.GetPosterURLResponse{
					Url: "https://example.com/poster.png",
				},
			}},
			status: 200,
			json:   `{"url":"https://example.com/poster.png"}`,
		},
		{
			name:   "delete poster",
			method: http.MethodDelete,
			path:   "/api/films/42/poster",
			rpc: []rpcExpectation{{
				method: "DeletePoster",
				request: &pb.DeletePosterRequest{
					FilmId: 42,
				},
				response: &emptypb.Empty{},
			}},
			status: 204,
		},
		{
			name:   "person filmography",
			method: http.MethodGet,
			path:   "/api/people/1",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				response: &pb.GetFilmResponse{
					Film: film,
				},
			}},
			status: 200,
			json:   `{"person":{"id":1,"name":"Person","original_name":"","photo_source":"","photo_credit":"","birth_place":"","biography":"","source_url":""},"films":[{"film":{"id":42,"title":"Film","film_type":"movie","release_year":2020,"age_limit":12,"genres":[]},"role":"actor","character":"Hero"}]}`,
		},
		{
			name:   "film people checks film existence",
			method: http.MethodGet,
			path:   "/api/films/42/people",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				response: &pb.GetFilmResponse{
					Film: film,
				},
			}},
			status: 200,
			json:   `{"people":[]}`,
		},
		{
			name:   "missing film people",
			method: http.MethodGet,
			path:   "/api/films/42/people",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				err: status.Error(codes.NotFound, "film not found"),
			}},
			status: 404,
			json:   `{"error":"film not found"}`,
		},
		{
			name:   "person filmography missing film",
			method: http.MethodGet,
			path:   "/api/people/1",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				err: status.Error(codes.NotFound, "film not found"),
			}},
			status: 404,
			json:   `{"error":"film not found"}`,
		},

		{
			name:   "invalid pagination",
			method: http.MethodGet,
			path:   "/api/films?limit=0",
			status: 400,
			json:   `{"error":"invalid limit: must be an integer from 1 to 100"}`,
		},
		{
			name:   "missing age limit",
			method: http.MethodPost,
			path:   "/api/films",
			body:   `{"title":"Film"}`,
			status: 400,
			json:   `{"error":"age_limit is required: invalid film input"}`,
		},
		{
			name:      "missing session",
			method:    http.MethodGet,
			path:      "/api/me/library",
			anonymous: true,
			status:    401,
			json:      `{"error":"unauthorized"}`,
		},
		{
			name:   "non admin film write",
			method: http.MethodDelete,
			path:   "/api/films/42",
			role:   "user",
			status: 403,
			json:   `{"error":"forbidden"}`,
		},
		{
			name:   "invalid preference",
			method: http.MethodPut,
			path:   "/api/me/preferences",
			body:   `{"key":"frame:catalog-view","value":"invalid"}`,
			status: 400,
			json:   `{"error":"invalid request data"}`,
		},
		{
			name:   "missing film error",
			method: http.MethodGet,
			path:   "/api/films/42",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				err: status.Error(codes.NotFound, "film not found"),
			}},
			status: 404,
			json:   `{"error":"film not found"}`,
		},
		{
			name:   "library missing film localized",
			method: http.MethodPut,
			path:   "/api/me/library/lists/watched/42",
			body:   `{"saved":true}`,
			rpc: []rpcExpectation{{
				method: "SetSaved",
				request: &pb.SetSavedRequest{
					UserId: 7,
					List:   "watched",
					FilmId: 42,
					Saved:  true,
				},
				err: status.Error(codes.NotFound, "film not found"),
			}},
			status: 404,
			json:   `{"error":"film not found"}`,
		},
		{
			name:        "oversized poster error",
			method:      http.MethodPost,
			path:        "/api/films/42/poster",
			body:        multipartData,
			contentType: multipartType,
			rpc: []rpcExpectation{{
				method: "UploadPoster",
				request: &pb.UploadPosterRequest{
					UserId:    7,
					FilmId:    42,
					ImageData: []byte("image"),
				},
				err: status.Error(codes.ResourceExhausted, "poster file is too large"),
			}},
			status: 413,
			json:   `{"error":"poster file is too large"}`,
		},
		{
			name:   "internal details hidden",
			method: http.MethodGet,
			path:   "/api/films/42",
			rpc: []rpcExpectation{{
				method: "GetFilm",
				request: &pb.GetFilmRequest{
					FilmId: 42,
				},
				err: status.Error(codes.Internal, "private database details"),
			}},
			status: 500,
			json:   `{"error":"internal server error"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			role := tc.role
			if role == "" {
				role = "admin"
			}

			handler, verify := testGateway(t, tc.rpc, role)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			request := httptest.NewRequest(tc.method, "http://gateway.example"+tc.path, strings.NewReader(tc.body)).WithContext(ctx)
			request.Header.Set("Origin", "http://localhost:8000")

			if tc.contentType != "" {
				request.Header.Set("Content-Type", tc.contentType)
			} else {
				request.Header.Set("Content-Type", "application/json")
			}

			if !tc.anonymous {
				request.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code, response.Body.String())

			if tc.json == "" {
				require.Empty(t, response.Body.String())
			} else {
				require.JSONEq(t, tc.json, response.Body.String())
			}

			verify()
		})
	}
}
