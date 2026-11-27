package grpc_test

import (
	"context"
	"math"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cataloggrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func TestParsePagination(t *testing.T) {
	cases := []struct {
		name   string
		input  *pb.Pagination
		limit  int
		offset int
		want   error
	}{
		{
			name:  "omitted pagination",
			limit: 20,
		},
		{
			name: "omitted limit",
			input: &pb.Pagination{
				Offset: 40,
			},
			limit:  20,
			offset: 40,
		},
		{
			name: "minimum",
			input: &pb.Pagination{
				Limit: proto.Int64(1),
			},
			limit: 1,
		},
		{
			name: "maximum",
			input: &pb.Pagination{
				Limit:  proto.Int64(100),
				Offset: 40,
			},
			limit:  100,
			offset: 40,
		},
		{
			name: "explicit zero",
			input: &pb.Pagination{
				Limit: proto.Int64(0),
			},
			want: usecase.ErrInvalidLimit,
		},
		{
			name: "negative limit",
			input: &pb.Pagination{
				Limit: proto.Int64(-1),
			},
			want: usecase.ErrInvalidLimit,
		},
		{
			name: "large limit",
			input: &pb.Pagination{
				Limit: proto.Int64(math.MaxInt64),
			},
			want: usecase.ErrInvalidLimit,
		},
		{
			name: "negative offset",
			input: &pb.Pagination{
				Offset: -1,
			},
			want: usecase.ErrInvalidOffset,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &conversionFilms{}
			handler := cataloggrpc.NewCatalogHandler(stub, nil, nil, nil, nil, nil)
			_, err := handler.ListFilms(t.Context(), &pb.ListFilmsRequest{
				Pagination: tc.input,
			})

			expectedCode := codes.OK
			if tc.want != nil {
				expectedCode = codes.InvalidArgument

				require.Zero(t, stub.calls)
			} else {
				require.Equal(t, 1, stub.calls)
			}

			require.Equal(t, expectedCode, status.Code(err))

			limit, offset := stub.limit, stub.offset
			require.Equal(t, tc.limit, limit)
			require.Equal(t, tc.offset, offset)
		})
	}
}

func TestParseFilmInput(t *testing.T) {
	cases := []struct {
		name  string
		input *pb.FilmInput
		want  error
	}{
		{
			name: "missing input",
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "missing age",
			input: &pb.FilmInput{
				Title:       "Film",
				ReleaseYear: 2020,
			},
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "explicit zero age",
			input: &pb.FilmInput{
				Title:       "Film",
				FilmType:    "movie",
				ReleaseYear: 2020,
				AgeLimit:    proto.Int32(0),
			},
		},
		{
			name: "overflowing year",
			input: &pb.FilmInput{
				ReleaseYear: 65536 + 2020,
				AgeLimit:    proto.Int32(0),
			},
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "overflowing age",
			input: &pb.FilmInput{
				ReleaseYear: 2020,
				AgeLimit:    proto.Int32(65536),
			},
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "overflowing duration",
			input: &pb.FilmInput{
				ReleaseYear: 2020,
				AgeLimit:    proto.Int32(0),
				DurationMin: proto.Int32(65536 + 120),
			},
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "negative overflow",
			input: &pb.FilmInput{
				ReleaseYear: math.MinInt32,
				AgeLimit:    proto.Int32(0),
			},
			want: domain.ErrInvalidFilmInput,
		},
		{
			name: "optional fields",
			input: &pb.FilmInput{
				Title:         "Film",
				FilmType:      "movie",
				ReleaseYear:   2020,
				AgeLimit:      proto.Int32(12),
				DurationMin:   proto.Int32(120),
				OriginalTitle: proto.String("Original"),
				Description:   proto.String("Description"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &conversionFilms{}
			handler := cataloggrpc.NewCatalogHandler(stub, nil, nil, nil, nil, nil)
			_, err := handler.CreateFilm(t.Context(), &pb.CreateFilmRequest{
				Film: tc.input,
			})

			expectedCode := codes.OK
			if tc.want != nil {
				expectedCode = codes.InvalidArgument

				require.Zero(t, stub.calls)
			}

			require.Equal(t, expectedCode, status.Code(err))

			input := stub.input

			if tc.want != nil {
				return
			}

			require.Equal(t, tc.input.GetTitle(), input.Title)
			require.Equal(t, int16(tc.input.GetReleaseYear()), input.ReleaseYear)
			require.Equal(t, int16(tc.input.GetAgeLimit()), input.AgeLimit)
			require.Equal(t, tc.input.OriginalTitle, input.OriginalTitle)
			require.Equal(t, tc.input.Description, input.Description)

			if tc.input.DurationMin == nil {
				require.Nil(t, input.DurationMin)
			} else {
				require.NotNil(t, input.DurationMin)
				require.Equal(t, int16(tc.input.GetDurationMin()), *input.DurationMin)
			}
		})
	}
}

type conversionFilms struct {
	usecase.FilmUseCase
	calls  int
	limit  int
	offset int
	input  domain.CreateFilmInput
}

func (s *conversionFilms) ListFilms(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
	s.calls++
	s.limit = limit
	s.offset = offset

	return nil, 0, nil
}

func (s *conversionFilms) FilterFilms(ctx context.Context, _ domain.FilmFilter, limit, offset int) ([]domain.Film, int, error) {
	return s.ListFilms(ctx, limit, offset)
}
func (s *conversionFilms) CreateFilm(_ context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	s.calls++
	s.input = input

	return 42, nil
}
