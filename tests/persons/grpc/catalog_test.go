package grpc_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	personsgrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/tests/mocks"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

func catalogPersonFixture() (domain.CatalogPerson, *pb.CatalogPerson) {
	birth := time.Date(1950, time.January, 2, 0, 0, 0, 0, time.UTC)
	death := time.Date(2020, time.February, 3, 0, 0, 0, 0, time.UTC)
	photo := "/static/persons/7.jpg"

	return domain.CatalogPerson{
		ID:           7,
		Name:         "Person Name",
		OriginalName: "Original Name",
		PhotoURL:     &photo,
		PhotoSource:  "https://source.example/photo",
		PhotoCredit:  "Photographer",
		BirthDate:    &birth,
		DeathDate:    &death,
		BirthPlace:   "City",
		Biography:    "Biography",
		SourceURL:    "https://source.example/person",
	}, &pb.CatalogPerson{
		Id:           7,
		Name:         "Person Name",
		OriginalName: "Original Name",
		PhotoUrl:     &photo,
		PhotoSource:  "https://source.example/photo",
		PhotoCredit:  "Photographer",
		BirthDate:    timestamppb.New(birth),
		DeathDate:    timestamppb.New(death),
		BirthPlace:   "City",
		Biography:    "Biography",
		SourceUrl:    "https://source.example/person",
	}
}

func TestGetFilmPersons(t *testing.T) {
	person, expectedPerson := catalogPersonFixture()
	minimal := domain.CatalogPerson{
		ID:   8,
		Name: "Actor",
	}

	tests := []struct {
		name   string
		people []domain.PersonCredit
		want   []*pb.PersonCredit
	}{
		{
			name: "all person fields and credit order",
			people: []domain.PersonCredit{
				{
					Person: person,
					Role:   "director",
				},
				{
					Person:    minimal,
					Role:      "actor",
					Character: "Hero",
				},
			},
			want: []*pb.PersonCredit{
				{
					Person: expectedPerson,
					Role:   "director",
				},
				{
					Person: &pb.CatalogPerson{
						Id:   8,
						Name: "Actor",
					},
					Role:      "actor",
					Character: "Hero",
				},
			},
		},
		{
			name: "empty credits",
			want: []*pb.PersonCredit{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			persons := mocks.NewMockPersonUseCase(ctrl)
			catalog := mocks.NewMockCatalogUseCase(ctrl)
			catalog.EXPECT().GetFilmPersons(t.Context(), int64(42)).Return(tt.people, nil)

			handler := personsgrpc.NewPersonHandler(persons, catalog)
			response, err := handler.GetFilmPersons(t.Context(), &pb.GetFilmPersonsRequest{
				FilmId: 42,
			})
			require.NoError(t, err)
			require.True(t, proto.Equal(&pb.GetFilmPersonsResponse{
				People: tt.want,
			}, response))
		})
	}
}

func TestGetPersonCatalog(t *testing.T) {
	person, expectedPerson := catalogPersonFixture()

	tests := []struct {
		name  string
		films []domain.FilmCredit
		want  []*pb.FilmCredit
	}{
		{
			name: "preserves films and roles",
			films: []domain.FilmCredit{
				{
					FilmID: 42,
					Role:   "director",
				},
				{
					FilmID:    42,
					Role:      "actor",
					Character: "Hero",
				},
			},
			want: []*pb.FilmCredit{
				{
					FilmId: 42,
					Role:   "director",
				},
				{
					FilmId:    42,
					Role:      "actor",
					Character: "Hero",
				},
			},
		},
		{
			name: "person without films",
			want: []*pb.FilmCredit{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			persons := mocks.NewMockPersonUseCase(ctrl)
			catalog := mocks.NewMockCatalogUseCase(ctrl)
			catalog.EXPECT().GetPersonCatalog(t.Context(), int64(7)).Return(usecase.CatalogDetails{
				Person: person,
				Films:  tt.films,
			}, nil)

			handler := personsgrpc.NewPersonHandler(persons, catalog)
			response, err := handler.GetPersonCatalog(t.Context(), &pb.GetPersonCatalogRequest{
				PersonId: 7,
			})
			require.NoError(t, err)
			require.True(t, proto.Equal(&pb.GetPersonCatalogResponse{
				Person: expectedPerson,
				Films:  tt.want,
			}, response))
		})
	}
}

func TestCatalogErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{
			name: "invalid film ID",
			err:  fmt.Errorf("film: %w", domain.ErrInvalidFilmID),
			code: codes.InvalidArgument,
		},
		{
			name: "invalid person ID",
			err:  fmt.Errorf("person: %w", domain.ErrInvalidPersonID),
			code: codes.InvalidArgument,
		},
		{
			name: "missing person",
			err:  fmt.Errorf("catalog: %w", domain.ErrPersonNotFound),
			code: codes.NotFound,
		},
		{
			name: "internal error",
			err:  errors.New("private database details"),
			code: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			persons := mocks.NewMockPersonUseCase(ctrl)
			catalog := mocks.NewMockCatalogUseCase(ctrl)
			catalog.EXPECT().GetFilmPersons(t.Context(), int64(0)).Return(nil, tt.err)
			catalog.EXPECT().GetPersonCatalog(t.Context(), int64(0)).Return(usecase.CatalogDetails{}, tt.err)

			handler := personsgrpc.NewPersonHandler(persons, catalog)
			people, peopleErr := handler.GetFilmPersons(t.Context(), nil)
			person, personErr := handler.GetPersonCatalog(t.Context(), nil)
			require.Nil(t, people)
			require.Nil(t, person)
			require.Equal(t, tt.code, status.Code(peopleErr))
			require.Equal(t, tt.code, status.Code(personErr))

			if tt.code == codes.Internal {
				require.Equal(t, "internal server error", status.Convert(peopleErr).Message())
				require.Equal(t, "internal server error", status.Convert(personErr).Message())
			}
		})
	}
}
