package grpc

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
)

func (h *PersonHandler) GetFilmPersons(ctx context.Context, req *pb.GetFilmPersonsRequest) (*pb.GetFilmPersonsResponse, error) {
	people, err := h.catalog.GetFilmPersons(ctx, req.GetFilmId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	result := make([]*pb.PersonCredit, len(people))
	for i, credit := range people {
		result[i] = &pb.PersonCredit{
			Person:    toProtoCatalogPerson(credit.Person),
			Role:      credit.Role,
			Character: credit.Character,
		}
	}

	return &pb.GetFilmPersonsResponse{
		People: result,
	}, nil
}

func (h *PersonHandler) GetPersonCatalog(ctx context.Context, req *pb.GetPersonCatalogRequest) (*pb.GetPersonCatalogResponse, error) {
	details, err := h.catalog.GetPersonCatalog(ctx, req.GetPersonId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	films := make([]*pb.FilmCredit, len(details.Films))
	for i, credit := range details.Films {
		films[i] = &pb.FilmCredit{
			FilmId:    credit.FilmID,
			Role:      credit.Role,
			Character: credit.Character,
		}
	}

	return &pb.GetPersonCatalogResponse{
		Person: toProtoCatalogPerson(details.Person),
		Films:  films,
	}, nil
}

func toProtoCatalogPerson(person domain.CatalogPerson) *pb.CatalogPerson {
	return &pb.CatalogPerson{
		Id:           person.ID,
		Name:         person.Name,
		OriginalName: person.OriginalName,
		PhotoUrl:     person.PhotoURL,
		PhotoSource:  person.PhotoSource,
		PhotoCredit:  person.PhotoCredit,
		BirthDate:    toOptionalTimestamp(person.BirthDate),
		DeathDate:    toOptionalTimestamp(person.DeathDate),
		BirthPlace:   person.BirthPlace,
		Biography:    person.Biography,
		SourceUrl:    person.SourceURL,
	}
}

func toOptionalTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}

	return timestamppb.New(*value)
}
