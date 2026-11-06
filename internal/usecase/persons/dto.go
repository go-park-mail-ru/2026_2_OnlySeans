package usecase

import (
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CreatePersonInput struct {
	Person domain.PersonInput
}

func NewCreatePersonInput(person domain.PersonInput) CreatePersonInput {
	return CreatePersonInput{
		Person: person,
	}
}

type UpdatePersonInput struct {
	PersonID domain.PersonID
	Person   domain.PersonInput
}

func NewUpdatePersonInput(personID domain.PersonID, person domain.PersonInput) UpdatePersonInput {
	return UpdatePersonInput{
		PersonID: personID,
		Person:   person,
	}
}

type DeletePersonInput struct {
	PersonID domain.PersonID
}

func NewDeletePersonInput(personID domain.PersonID) DeletePersonInput {
	return DeletePersonInput{PersonID: personID}
}

type ListPersonsInput struct {
	Limit  int
	Offset int
}

func NewListPersonsInput(limit, offset int) ListPersonsInput {
	return ListPersonsInput{
		Limit:  limit,
		Offset: offset,
	}
}

type PersonOutput struct {
	Person domain.Person
}

func NewPersonOutput(person *domain.Person) *PersonOutput {
	return &PersonOutput{Person: *person}
}

type ListPersonsOutput struct {
	Persons []domain.Person
	Total   int
}

func NewListPersonsOutput(persons []domain.Person, total int) *ListPersonsOutput {
	return &ListPersonsOutput{
		Persons: persons,
		Total:   total,
	}
}

func preparePerson(in domain.PersonInput) (*domain.PersonInput, error) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.LastName = strings.TrimSpace(in.LastName)

	if err := in.Validate(); err != nil {
		return nil, err
	}

	return &in, nil
}

// CreatePersonOutput contains the identifier assigned by the repository.
type CreatePersonOutput struct {
	ID domain.PersonID
}
