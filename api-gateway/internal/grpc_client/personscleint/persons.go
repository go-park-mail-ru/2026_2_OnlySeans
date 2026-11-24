package personsclient

import (
	"context"
	"errors"
	"time"

	pbPersons "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/persons"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrInvalidArgument = errors.New("invalid person data")
	ErrPersonNotFound  = errors.New("person not found")
	ErrFilmNotFound    = errors.New("film not found")
	ErrInternal        = errors.New("internal server error")
)

type PersonClient interface {
	CreatePerson(ctx context.Context, firstName, lastName string, birthDate *time.Time) (int64, error)
	GetByID(ctx context.Context, personID int64) (*pbPersons.Person, error)
	ListPersons(ctx context.Context, limit, offset int) (*pbPersons.ListPersonsResponse, error)
	UpdatePerson(ctx context.Context, personID int64, firstName, lastName string, birthDate *time.Time) (*pbPersons.Person, error)
	DeletePerson(ctx context.Context, personID int64) error
	GetFilmPersons(ctx context.Context, filmID int64) (*pbPersons.GetFilmPersonsResponse, error)
	GetPersonCatalog(ctx context.Context, personID int64) (*pbPersons.GetPersonCatalogResponse, error)
}

type personClient struct {
	client pbPersons.PersonServiceClient
}

func NewPersonClient(cl pbPersons.PersonServiceClient) PersonClient {
	return &personClient{
		client: cl,
	}
}

func (p *personClient) CreatePerson(ctx context.Context, firstName, lastName string, birthDate *time.Time) (int64, error) {
	var date *timestamppb.Timestamp
	if birthDate != nil {
		date = timestamppb.New(*birthDate)
	}

	resp, err := p.client.CreatePerson(ctx, &pbPersons.CreatePersonRequest{
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: date,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return 0, ErrInvalidArgument
		default:
			return 0, ErrInternal
		}
	}

	return resp.GetPersonId(), nil
}

func (p *personClient) GetByID(ctx context.Context, personID int64) (*pbPersons.Person, error) {
	resp, err := p.client.GetByID(ctx, &pbPersons.GetPersonRequest{
		PersonId: personID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return nil, ErrPersonNotFound
		case codes.InvalidArgument:
			return nil, ErrInvalidArgument
		default:
			return nil, ErrInternal
		}
	}

	return resp.GetPerson(), nil
}

func (p *personClient) ListPersons(ctx context.Context, limit, offset int) (*pbPersons.ListPersonsResponse, error) {
	resp, err := p.client.ListPersons(ctx, &pbPersons.ListPersonsRequest{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return nil, ErrInvalidArgument
		default:
			return nil, ErrInternal
		}
	}

	return resp, nil
}

func (p *personClient) UpdatePerson(ctx context.Context, personID int64, firstName, lastName string, birthDate *time.Time) (*pbPersons.Person, error) {
	var date *timestamppb.Timestamp

	if birthDate != nil {
		date = timestamppb.New(*birthDate)
	}

	resp, err := p.client.UpdatePerson(ctx, &pbPersons.UpdatePersonRequest{
		PersonId:  personID,
		FirstName: firstName,
		LastName:  lastName,
		BirthDate: date,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return nil, ErrPersonNotFound
		case codes.InvalidArgument:
			return nil, ErrInvalidArgument
		default:
			return nil, ErrInternal
		}
	}

	return resp.GetPerson(), nil
}

func (p *personClient) GetFilmPersons(ctx context.Context, filmID int64) (*pbPersons.GetFilmPersonsResponse, error) {
	resp, err := p.client.GetFilmPersons(ctx, &pbPersons.GetFilmPersonsRequest{
		FilmId: filmID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return nil, ErrFilmNotFound
		case codes.InvalidArgument:
			return nil, ErrInvalidArgument
		default:
			return nil, ErrInternal
		}
	}

	return resp, nil
}

func (p *personClient) DeletePerson(ctx context.Context, personID int64) error {
	_, err := p.client.DeletePerson(ctx, &pbPersons.DeletePersonRequest{
		PersonId: personID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return ErrPersonNotFound
		case codes.InvalidArgument:
			return ErrInvalidArgument
		default:
			return ErrInternal
		}
	}

	return nil
}

func (p *personClient) GetPersonCatalog(ctx context.Context, personID int64) (*pbPersons.GetPersonCatalogResponse, error) {
	resp, err := p.client.GetPersonCatalog(ctx, &pbPersons.GetPersonCatalogRequest{
		PersonId: personID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound:
			return nil, ErrPersonNotFound
		case codes.InvalidArgument:
			return nil, ErrInvalidArgument
		default:
			return nil, ErrInternal
		}
	}

	return resp, nil
}
