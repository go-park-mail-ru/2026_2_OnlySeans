package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
)

const (
	minLimit  = 1
	maxLimit  = 100
	minOffset = 0
)

type PersonUseCase interface {
	CreatePerson(ctx context.Context, input domain.PersonInput) (domain.PersonID, error)
	GetPersonByID(ctx context.Context, personID domain.PersonID) (*domain.Person, error)
	ListPersons(ctx context.Context, limit, offset int) ([]domain.Person, int, error)
	UpdatePerson(ctx context.Context, personID domain.PersonID, input domain.PersonInput) (*domain.Person, error)
	DeletePerson(ctx context.Context, personID domain.PersonID) error
}

type personUseCase struct {
	personRepo domain.PersonRepository
}

func NewPersonUseCase(repo domain.PersonRepository) PersonUseCase {
	return &personUseCase{
		personRepo: repo,
	}
}

func (p *personUseCase) CreatePerson(ctx context.Context, input domain.PersonInput) (domain.PersonID, error) {
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)

	if err := input.Validate(); err != nil {
		return 0, err
	}

	id, err := p.personRepo.Create(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("create person: %w", err)
	}

	return id, nil
}

func (p *personUseCase) GetPersonByID(ctx context.Context, personID domain.PersonID) (*domain.Person, error) {
	if personID <= 0 {
		return nil, domain.ErrInvalidPersonID
	}

	person, err := p.personRepo.GetByID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("get person: %w", err)
	}

	return person, nil
}

func (p *personUseCase) ListPersons(ctx context.Context, limit, offset int) ([]domain.Person, int, error) {
	if limit < minLimit || limit > maxLimit || offset < minOffset {
		return nil, 0, domain.ErrInvalidPagination
	}

	persons, total, err := p.personRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list persons: %w", err)
	}

	return persons, total, nil
}

func (p *personUseCase) UpdatePerson(ctx context.Context, personID domain.PersonID, input domain.PersonInput) (*domain.Person, error) {
	if personID <= 0 {
		return nil, domain.ErrInvalidPersonID
	}

	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)

	if err := input.Validate(); err != nil {
		return nil, err
	}

	person, err := p.personRepo.Update(ctx, personID, input)
	if err != nil {
		return nil, fmt.Errorf("update person: %w", err)
	}

	return person, nil
}

func (p *personUseCase) DeletePerson(ctx context.Context, personID domain.PersonID) error {
	if personID <= 0 {
		return domain.ErrInvalidPersonID
	}

	if err := p.personRepo.Delete(ctx, personID); err != nil {
		return fmt.Errorf("delete person: %w", err)
	}

	return nil
}
