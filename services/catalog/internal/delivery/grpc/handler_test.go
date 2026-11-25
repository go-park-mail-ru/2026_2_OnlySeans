package grpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type catalogStub struct {
	mu        sync.Mutex
	calls     []string
	err       error
	noScore   bool
	imageData []byte
}

func (s *catalogStub) check(method string, valid bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, method)

	if !valid {
		return fmt.Errorf("unexpected %s arguments", method)
	}

	return s.err
}

func (s *catalogStub) called() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.calls...)
}

func sampleFilm() domain.Film {
	duration := int16(120)

	return domain.Film{
		ID:          42,
		Title:       "Film",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 2020,
		AgeLimit:    12,
		DurationMin: &duration,
		Genres: []domain.Genre{{
			ID:   3,
			Name: "Drama",
			Slug: "drama",
		}},
	}
}

func sampleCollection() domain.Collection {
	return domain.Collection{
		ID:    5,
		Title: "Picks",
		Slug:  "picks",
	}
}

func sampleTime() time.Time {
	return time.Date(2026, time.November, 25, 12, 0, 0, 0, time.UTC)
}

func sampleInput() *pb.FilmInput {
	return &pb.FilmInput{
		Title:       "Film",
		FilmType:    "movie",
		ReleaseYear: 2020,
		AgeLimit:    proto.Int32(0),
	}
}

func validInput(input domain.CreateFilmInput) bool {
	return input.Title == "Film" && input.FilmType == domain.FilmTypeMovie && input.ReleaseYear == 2020 && input.AgeLimit == 0
}

func (s *catalogStub) CreateFilm(_ context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	return 42, s.check("CreateFilm", validInput(input))
}

func (s *catalogStub) GetFilmByID(_ context.Context, id domain.FilmID) (domain.Film, error) {
	return sampleFilm(), s.check("GetFilm", id == 42)
}

func (s *catalogStub) ListFilms(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
	return []domain.Film{sampleFilm()}, 37, s.check("ListFilms", limit == 2 && offset == 4)
}

func (s *catalogStub) UpdateFilm(_ context.Context, id domain.FilmID, input domain.CreateFilmInput) error {
	return s.check("UpdateFilm", id == 42 && validInput(input))
}

func (s *catalogStub) DeleteFilm(_ context.Context, id domain.FilmID) error {
	return s.check("DeleteFilm", id == 42)
}

func (s *catalogStub) ListCollections(_ context.Context, limit, offset int) ([]domain.Collection, int, error) {
	return []domain.Collection{sampleCollection()}, 9, s.check("ListCollections", limit == 2 && offset == 4)
}

func (s *catalogStub) GetCollection(_ context.Context, id domain.CollectionID, limit, offset int) (usecase.CollectionDetails, error) {
	return usecase.CollectionDetails{
		Collection: sampleCollection(),
		Films:      []domain.Film{sampleFilm()},
		Total:      3,
	}, s.check("GetCollection", id == 5 && limit == 2 && offset == 4)
}

func (s *catalogStub) SetFilmRating(_ context.Context, user domain.UserID, film domain.FilmID, score int) (domain.FilmRating, error) {
	return domain.FilmRating{
		AccountID: user,
		FilmID:    film,
		Score:     score,
		CreatedAt: sampleTime(),
		UpdatedAt: sampleTime(),
	}, s.check("SetFilmRating", user == 7 && film == 42 && score == 8)
}

func (s *catalogStub) GetFilmRating(_ context.Context, id domain.FilmID) (domain.FilmRatingSummary, error) {
	return domain.FilmRatingSummary{
		FilmID:       42,
		AverageScore: 8.5,
		Votes:        2,
	}, s.check("GetFilmRating", id == 42)
}

func (s *catalogStub) GetMyFilmRating(_ context.Context, user domain.UserID, film domain.FilmID) (*int, error) {
	var score *int

	if !s.noScore {
		value := 8
		score = &value
	}

	return score, s.check("GetMyFilmRating", user == 7 && film == 42)
}

func (s *catalogStub) DeleteFilmRating(_ context.Context, user domain.UserID, film domain.FilmID) error {
	return s.check("DeleteFilmRating", user == 7 && film == 42)
}

func (s *catalogStub) GetCollectionRatings(_ context.Context, id domain.CollectionID, limit, offset int) ([]domain.FilmRatingSummary, int, error) {
	return []domain.FilmRatingSummary{{
		FilmID:       42,
		AverageScore: 8.5,
		Votes:        2,
	}}, 3, s.check("GetCollectionRatings", id == 5 && limit == 2 && offset == 4)
}

func (s *catalogStub) GetLibrary(_ context.Context, user domain.UserID) (domain.Library, error) {
	return domain.Library{
		Favorite: []domain.FilmID{42},
		Watched:  []domain.FilmID{43},
	}, s.check("GetLibrary", user == 7)
}

func (s *catalogStub) SetSaved(_ context.Context, user domain.UserID, list domain.List, film domain.FilmID, saved bool) error {
	return s.check("SetSaved", user == 7 && list == domain.Watched && film == 42 && saved)
}

func (s *catalogStub) GetPreferences(_ context.Context, user domain.UserID) (domain.Preferences, error) {
	collapsed := false
	view := domain.CatalogViewGrid

	return domain.Preferences{
		SidebarCollapsed: &collapsed,
		CatalogView:      &view,
	}, s.check("GetPreferences", user == 7)
}

func (s *catalogStub) SavePreference(_ context.Context, user domain.UserID, update domain.PreferenceUpdate) error {
	valid := update.SidebarCollapsed != nil && !*update.SidebarCollapsed && update.CatalogView == nil
	if update.CatalogView != nil {
		valid = *update.CatalogView == domain.CatalogViewGrid && update.SidebarCollapsed == nil
	}

	return s.check("SavePreference", user == 7 && valid)
}

func (s *catalogStub) UploadPoster(_ context.Context, input usecase.UploadPosterInput) (string, error) {
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return "", err
	}

	return "poster.png", s.check("UploadPoster", input.UserID == 7 && input.FilmID == 42 && input.Size == int64(len(s.imageData)) && bytes.Equal(data, s.imageData))
}

func (s *catalogStub) GetPosterURL(_ context.Context, film domain.FilmID) (string, error) {
	return "https://example.com/poster.png", s.check("GetPosterURL", film == 42)
}

func (s *catalogStub) DeletePoster(_ context.Context, film domain.FilmID) error {
	return s.check("DeletePoster", film == 42)
}

func testClient(t *testing.T, stub *catalogStub) pb.CatalogServiceClient {
	t.Helper()

	handler := NewCatalogHandler(stub, stub, stub, stub, stub, stub)
	server, err := NewServer(handler, 5<<20)
	require.NoError(t, err)

	listener := bufconn.Listen(1 << 20)
	serveErr := make(chan error, 1)

	go func() {
		serveErr <- server.Serve(listener)
	}()

	connection, err := grpcgo.NewClient("passthrough:///catalog",
		grpcgo.WithTransportCredentials(insecure.NewCredentials()),
		grpcgo.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpcgo.WithDefaultCallOptions(grpcgo.MaxCallSendMsgSize(6<<20)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
		server.Stop()
		require.NoError(t, listener.Close())

		err := <-serveErr
		if err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
			t.Errorf("serve catalog: %v", err)
		}
	})

	return pb.NewCatalogServiceClient(connection)
}

func TestCatalogRPCs(t *testing.T) {
	pagination := &pb.Pagination{
		Limit:  proto.Int64(2),
		Offset: 4,
	}
	expectedFilm := &pb.Film{
		Id:          42,
		Title:       "Film",
		FilmType:    "movie",
		ReleaseYear: 2020,
		DurationMin: proto.Int32(120),
		AgeLimit:    12,
		Genres: []*pb.Genre{{
			Id:   3,
			Name: "Drama",
			Slug: "drama",
		}},
	}
	expectedCollection := &pb.Collection{
		Id:    5,
		Title: "Picks",
		Slug:  "picks",
	}
	expectedRating := &pb.FilmRatingSummary{
		FilmId:       42,
		AverageScore: 8.5,
		Votes:        2,
	}
	cases := []struct {
		name string
		call func(context.Context, pb.CatalogServiceClient) (proto.Message, error)
		want proto.Message
	}{
		{
			name: "CreateFilm",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.CreateFilm(ctx, &pb.CreateFilmRequest{
					Film: sampleInput(),
				})
			},
			want: &pb.CreateFilmResponse{
				FilmId: 42,
			},
		},
		{
			name: "GetFilm",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetFilm(ctx, &pb.GetFilmRequest{
					FilmId: 42,
				})
			},
			want: &pb.GetFilmResponse{
				Film: expectedFilm,
			},
		},
		{
			name: "ListFilms",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.ListFilms(ctx, &pb.ListFilmsRequest{
					Pagination: pagination,
				})
			},
			want: &pb.ListFilmsResponse{
				Films: []*pb.Film{expectedFilm},
				Total: 37,
			},
		},
		{
			name: "UpdateFilm",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.UpdateFilm(ctx, &pb.UpdateFilmRequest{
					FilmId: 42,
					Film:   sampleInput(),
				})
			},
			want: &emptypb.Empty{},
		},
		{
			name: "DeleteFilm",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.DeleteFilm(ctx, &pb.DeleteFilmRequest{
					FilmId: 42,
				})
			},
			want: &emptypb.Empty{},
		},
		{
			name: "ListCollections",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.ListCollections(ctx, &pb.ListCollectionsRequest{
					Pagination: pagination,
				})
			},
			want: &pb.ListCollectionsResponse{
				Collections: []*pb.Collection{expectedCollection},
				Total:       9,
			},
		},
		{
			name: "GetCollection",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetCollection(ctx, &pb.GetCollectionRequest{
					CollectionId: 5,
					Pagination:   pagination,
				})
			},
			want: &pb.GetCollectionResponse{
				Collection: expectedCollection,
				Films:      []*pb.Film{expectedFilm},
				Total:      3,
			},
		},
		{
			name: "SetFilmRating",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.SetFilmRating(ctx, &pb.SetFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
					Score:  8,
				})
			},
			want: &pb.SetFilmRatingResponse{
				Rating: &pb.FilmRating{
					UserId:    7,
					FilmId:    42,
					Score:     8,
					CreatedAt: timestamppb.New(sampleTime()),
					UpdatedAt: timestamppb.New(sampleTime()),
				},
			},
		},
		{
			name: "GetFilmRating",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetFilmRating(ctx, &pb.GetFilmRatingRequest{
					FilmId: 42,
				})
			},
			want: &pb.GetFilmRatingResponse{
				Rating: expectedRating,
			},
		},
		{
			name: "GetMyFilmRating",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetMyFilmRating(ctx, &pb.GetMyFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
				})
			},
			want: &pb.GetMyFilmRatingResponse{
				Score: proto.Int32(8),
			},
		},
		{
			name: "DeleteFilmRating",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.DeleteFilmRating(ctx, &pb.DeleteFilmRatingRequest{
					UserId: 7,
					FilmId: 42,
				})
			},
			want: &emptypb.Empty{},
		},
		{
			name: "GetCollectionRatings",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetCollectionRatings(ctx, &pb.GetCollectionRatingsRequest{
					CollectionId: 5,
					Pagination:   pagination,
				})
			},
			want: &pb.GetCollectionRatingsResponse{
				Ratings: []*pb.FilmRatingSummary{expectedRating},
				Total:   3,
			},
		},
		{
			name: "GetLibrary",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetLibrary(ctx, &pb.GetLibraryRequest{
					UserId: 7,
				})
			},
			want: &pb.GetLibraryResponse{
				Library: &pb.Library{
					Favorite: []int64{42},
					Watched:  []int64{43},
				},
			},
		},
		{
			name: "SetSaved",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.SetSaved(ctx, &pb.SetSavedRequest{
					UserId: 7,
					List:   "watched",
					FilmId: 42,
					Saved:  true,
				})
			},
			want: &emptypb.Empty{},
		},
		{
			name: "GetPreferences",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetPreferences(ctx, &pb.GetPreferencesRequest{
					UserId: 7,
				})
			},
			want: &pb.GetPreferencesResponse{
				Preferences: &pb.Preferences{
					SidebarCollapsed: proto.Bool(false),
					CatalogView:      proto.String("grid"),
				},
			},
		},
		{
			name: "SavePreference",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.SavePreference(ctx, &pb.SavePreferenceRequest{
					UserId: 7,
					Update: &pb.SavePreferenceRequest_SidebarCollapsed{
						SidebarCollapsed: false,
					},
				})
			},
			want: &emptypb.Empty{},
		},
		{
			name: "UploadPoster",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.UploadPoster(ctx, &pb.UploadPosterRequest{
					UserId:    7,
					FilmId:    42,
					ImageData: []byte("image"),
				})
			},
			want: &pb.UploadPosterResponse{
				Filename: "poster.png",
			},
		},
		{
			name: "GetPosterURL",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.GetPosterURL(ctx, &pb.GetPosterURLRequest{
					FilmId: 42,
				})
			},
			want: &pb.GetPosterURLResponse{
				Url: "https://example.com/poster.png",
			},
		},
		{
			name: "DeletePoster",
			call: func(ctx context.Context, client pb.CatalogServiceClient) (proto.Message, error) {
				return client.DeletePoster(ctx, &pb.DeletePosterRequest{
					FilmId: 42,
				})
			},
			want: &emptypb.Empty{},
		},
	}
	outcomes := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "success",
			code: codes.OK,
		},
		{
			name: "not found",
			err:  fmt.Errorf("lookup: %w", domain.ErrFilmNotFound),
			code: codes.NotFound,
		},
		{
			name: "internal error",
			err:  errors.New("private database details"),
			code: codes.Internal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, outcome := range outcomes {
				t.Run(outcome.name, func(t *testing.T) {
					stub := &catalogStub{
						imageData: []byte("image"),
						err:       outcome.err,
					}
					client := testClient(t, stub)

					ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
					defer cancel()

					result, err := tc.call(ctx, client)
					require.Equal(t, outcome.code, status.Code(err))

					if outcome.err == nil {
						require.True(t, proto.Equal(tc.want, result), "want %v, got %v", tc.want, result)
					} else {
						require.NotContains(t, status.Convert(err).Message(), "private database details")
					}

					require.Equal(t, []string{tc.name}, stub.called())
				})
			}
		})
	}
}

func TestNilRequests(t *testing.T) {
	handler := NewCatalogHandler(nil, nil, nil, nil, nil, nil)

	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "CreateFilm",
			call: func(ctx context.Context) error {
				_, err := handler.CreateFilm(ctx, nil)
				return err
			},
		},
		{
			name: "GetFilm",
			call: func(ctx context.Context) error {
				_, err := handler.GetFilm(ctx, nil)
				return err
			},
		},
		{
			name: "ListFilms",
			call: func(ctx context.Context) error {
				_, err := handler.ListFilms(ctx, nil)
				return err
			},
		},
		{
			name: "UpdateFilm",
			call: func(ctx context.Context) error {
				_, err := handler.UpdateFilm(ctx, nil)
				return err
			},
		},
		{
			name: "DeleteFilm",
			call: func(ctx context.Context) error {
				_, err := handler.DeleteFilm(ctx, nil)
				return err
			},
		},
		{
			name: "ListCollections",
			call: func(ctx context.Context) error {
				_, err := handler.ListCollections(ctx, nil)
				return err
			},
		},
		{
			name: "GetCollection",
			call: func(ctx context.Context) error {
				_, err := handler.GetCollection(ctx, nil)
				return err
			},
		},
		{
			name: "SetFilmRating",
			call: func(ctx context.Context) error {
				_, err := handler.SetFilmRating(ctx, nil)
				return err
			},
		},
		{
			name: "GetFilmRating",
			call: func(ctx context.Context) error {
				_, err := handler.GetFilmRating(ctx, nil)
				return err
			},
		},
		{
			name: "GetMyFilmRating",
			call: func(ctx context.Context) error {
				_, err := handler.GetMyFilmRating(ctx, nil)
				return err
			},
		},
		{
			name: "DeleteFilmRating",
			call: func(ctx context.Context) error {
				_, err := handler.DeleteFilmRating(ctx, nil)
				return err
			},
		},
		{
			name: "GetCollectionRatings",
			call: func(ctx context.Context) error {
				_, err := handler.GetCollectionRatings(ctx, nil)
				return err
			},
		},
		{
			name: "GetLibrary",
			call: func(ctx context.Context) error {
				_, err := handler.GetLibrary(ctx, nil)
				return err
			},
		},
		{
			name: "SetSaved",
			call: func(ctx context.Context) error {
				_, err := handler.SetSaved(ctx, nil)
				return err
			},
		},
		{
			name: "GetPreferences",
			call: func(ctx context.Context) error {
				_, err := handler.GetPreferences(ctx, nil)
				return err
			},
		},
		{
			name: "SavePreference",
			call: func(ctx context.Context) error {
				_, err := handler.SavePreference(ctx, nil)
				return err
			},
		},
		{
			name: "UploadPoster",
			call: func(ctx context.Context) error {
				_, err := handler.UploadPoster(ctx, nil)
				return err
			},
		},
		{
			name: "GetPosterURL",
			call: func(ctx context.Context) error {
				_, err := handler.GetPosterURL(ctx, nil)
				return err
			},
		},
		{
			name: "DeletePoster",
			call: func(ctx context.Context) error {
				_, err := handler.DeletePoster(ctx, nil)
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, codes.InvalidArgument, status.Code(tc.call(t.Context())))
		})
	}
}

func TestPersonalScorePresenceOverGRPC(t *testing.T) {
	cases := []struct {
		name   string
		absent bool
	}{
		{
			name: "saved score",
		},
		{
			name:   "no score",
			absent: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := testClient(t, &catalogStub{
				noScore: tc.absent,
			})

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			result, err := client.GetMyFilmRating(ctx, &pb.GetMyFilmRatingRequest{
				UserId: 7,
				FilmId: 42,
			})
			require.NoError(t, err)

			if tc.absent {
				require.Nil(t, result.Score)
			} else {
				require.NotNil(t, result.Score)
				require.Equal(t, int32(8), *result.Score)
			}
		})
	}
}

func TestPreferenceUpdates(t *testing.T) {
	cases := []struct {
		name    string
		request *pb.SavePreferenceRequest
		code    codes.Code
		called  bool
	}{
		{
			name: "explicit false",
			request: &pb.SavePreferenceRequest{
				UserId: 7,
				Update: &pb.SavePreferenceRequest_SidebarCollapsed{
					SidebarCollapsed: false,
				},
			},
			code:   codes.OK,
			called: true,
		},
		{
			name: "grid",
			request: &pb.SavePreferenceRequest{
				UserId: 7,
				Update: &pb.SavePreferenceRequest_CatalogView{
					CatalogView: "grid",
				},
			},
			code:   codes.OK,
			called: true,
		},
		{
			name: "empty update",
			request: &pb.SavePreferenceRequest{
				UserId: 7,
			},
			code: codes.InvalidArgument,
		},
		{
			name: "nil boolean variant",
			request: &pb.SavePreferenceRequest{
				UserId: 7,
				Update: (*pb.SavePreferenceRequest_SidebarCollapsed)(nil),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "nil view variant",
			request: &pb.SavePreferenceRequest{
				UserId: 7,
				Update: (*pb.SavePreferenceRequest_CatalogView)(nil),
			},
			code: codes.InvalidArgument,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &catalogStub{}
			handler := NewCatalogHandler(stub, stub, stub, stub, stub, stub)
			_, err := handler.SavePreference(t.Context(), tc.request)
			require.Equal(t, tc.code, status.Code(err))

			if tc.called {
				require.Equal(t, []string{"SavePreference"}, stub.called())
			} else {
				require.Empty(t, stub.called())
			}
		})
	}
}

func TestPosterTransportSize(t *testing.T) {
	cases := []struct {
		name   string
		size   int
		code   codes.Code
		called bool
	}{
		{
			name:   "five MiB poster",
			size:   5 << 20,
			code:   codes.OK,
			called: true,
		},
		{
			name: "transport limit",
			size: 6 << 20,
			code: codes.ResourceExhausted,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := bytes.Repeat([]byte{1}, tc.size)
			stub := &catalogStub{
				imageData: data,
			}
			client := testClient(t, stub)

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			_, err := client.UploadPoster(ctx, &pb.UploadPosterRequest{
				UserId:    7,
				FilmId:    42,
				ImageData: data,
			})
			require.Equal(t, tc.code, status.Code(err))

			if tc.called {
				require.Equal(t, []string{"UploadPoster"}, stub.called())
			} else {
				require.Empty(t, stub.called())
			}
		})
	}
}

func TestServerConfiguration(t *testing.T) {
	cases := []struct {
		name    string
		handler *CatalogHandler
		size    int64
		valid   bool
	}{
		{
			name:    "valid",
			handler: &CatalogHandler{},
			size:    5 << 20,
			valid:   true,
		},
		{
			name: "nil handler",
			size: 5 << 20,
		},
		{
			name:    "zero size",
			handler: &CatalogHandler{},
		},
		{
			name:    "negative size",
			handler: &CatalogHandler{},
			size:    -1,
		},
		{
			name:    "overflow",
			handler: &CatalogHandler{},
			size:    int64(^uint64(0) >> 1),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, err := NewServer(tc.handler, tc.size)
			if tc.valid {
				require.NoError(t, err)
				server.Stop()
			} else {
				require.Error(t, err)
				require.Nil(t, server)
			}
		})
	}
}
