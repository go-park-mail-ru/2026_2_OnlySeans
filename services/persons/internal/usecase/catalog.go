package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
)

type CatalogUseCase interface {
	GetFilmPersons(ctx context.Context, filmID int64) ([]domain.PersonCredit, error)
	GetPersonCatalog(ctx context.Context, personID int64) (CatalogDetails, error)
}

type CatalogDetails struct {
	Person domain.CatalogPerson
	Films  []domain.FilmCredit
}

type catalogUseCase struct {
	persons domain.PersonCatalogRepository
}

func NewCatalogUseCase(persons domain.PersonCatalogRepository) CatalogUseCase {
	return &catalogUseCase{
		persons: persons,
	}
}

func (u *catalogUseCase) GetFilmPersons(ctx context.Context, filmID int64) ([]domain.PersonCredit, error) {
	if filmID <= 0 {
		return nil, domain.ErrInvalidFilmID
	}

	people, err := u.persons.FilmPersons(ctx, filmID)
	if err != nil {
		return nil, fmt.Errorf("get film persons: %w", err)
	}

	return people, nil
}

func (u *catalogUseCase) GetPersonCatalog(ctx context.Context, personID int64) (CatalogDetails, error) {
	if personID <= 0 {
		return CatalogDetails{}, domain.ErrInvalidPersonID
	}

	person, films, err := u.persons.Get(ctx, personID)
	if err != nil {
		return CatalogDetails{}, fmt.Errorf("get person catalog: %w", err)
	}

	return CatalogDetails{
		Person: person,
		Films:  films,
	}, nil
}
