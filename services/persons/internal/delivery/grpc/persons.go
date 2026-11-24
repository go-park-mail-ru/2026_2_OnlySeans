package grpc

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PersonHandler struct {
	pb.UnimplementedPersonServiceServer
	persons usecase.PersonUseCase
}

func NewPersonHandler(persons usecase.PersonUseCase) *PersonHandler {
	return &PersonHandler{
		persons: persons,
	}
}

func (h *PersonHandler) CreatePerson(ctx context.Context, req *pb.CreatePersonRequest) (*pb.CreatePersonResponse, error) {
	birthDate, err := parseProtoBirthDate(req.GetBirthDate())
	if err != nil {
		return nil, toGRPCError(err)
	}

	input := domain.PersonInput{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		BirthDate: birthDate,
	}

	id, err := h.persons.CreatePerson(ctx, input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.CreatePersonResponse{PersonId: int64(id)}, nil
}

func (h *PersonHandler) GetByID(ctx context.Context, req *pb.GetPersonRequest) (*pb.GetPersonResponse, error) {
	person, err := h.persons.GetPersonByID(ctx, domain.PersonID(req.GetPersonId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetPersonResponse{Person: toProtoPerson(person)}, nil
}

func (h *PersonHandler) ListPersons(ctx context.Context, req *pb.ListPersonsRequest) (*pb.ListPersonsResponse, error) {
	limit := int(req.GetLimit())
	offset := int(req.GetOffset())

	if int64(limit) != req.GetLimit() || int64(offset) != req.GetOffset() {
		return nil, toGRPCError(domain.ErrInvalidPagination)
	}

	persons, total, err := h.persons.ListPersons(ctx, limit, offset)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.ListPersonsResponse{
		Persons: toProtoPersons(persons),
		Total:   int64(total),
	}, nil
}

func (h *PersonHandler) UpdatePerson(ctx context.Context, req *pb.UpdatePersonRequest) (*pb.UpdatePersonResponse, error) {
	birthDate, err := parseProtoBirthDate(req.GetBirthDate())
	if err != nil {
		return nil, toGRPCError(err)
	}

	input := domain.PersonInput{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		BirthDate: birthDate,
	}

	person, err := h.persons.UpdatePerson(ctx, domain.PersonID(req.GetPersonId()), input)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UpdatePersonResponse{Person: toProtoPerson(person)}, nil
}

func (h *PersonHandler) DeletePerson(ctx context.Context, req *pb.DeletePersonRequest) (*pb.DeletePersonResponse, error) {
	if err := h.persons.DeletePerson(ctx, domain.PersonID(req.GetPersonId())); err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.DeletePersonResponse{}, nil
}

func parseProtoBirthDate(date *timestamppb.Timestamp) (*time.Time, error) {
	if date == nil {
		return nil, nil
	}

	if err := date.CheckValid(); err != nil {
		return nil, domain.ErrInvalidBirthDate
	}

	value := date.AsTime()
	return &value, nil
}

func toProtoPerson(person *domain.Person) *pb.Person {
	if person == nil {
		return nil
	}

	var birthDate *timestamppb.Timestamp
	if person.BirthDate != nil {
		birthDate = timestamppb.New(*person.BirthDate)
	}

	return &pb.Person{
		Id:        int64(person.ID),
		FirstName: person.FirstName,
		LastName:  person.LastName,
		BirthDate: birthDate,
		PhotoId:   person.PhotoID,
		CreatedAt: timestamppb.New(person.CreatedAt),
		UpdatedAt: timestamppb.New(person.UpdatedAt),
	}
}

func toProtoPersons(persons []domain.Person) []*pb.Person {
	result := make([]*pb.Person, len(persons))
	for i := range persons {
		result[i] = toProtoPerson(&persons[i])
	}
	return result
}
