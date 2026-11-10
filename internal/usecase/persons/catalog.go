package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type CatalogFilmRepository interface {
	GetFilmByID(context.Context, domain.FilmID) (domain.Film, error)
}

type GetFilmPersonsUseCase struct {
	persons domain.PersonCatalogRepository
	films   CatalogFilmRepository
}

func NewGetFilmPersonsUseCase(persons domain.PersonCatalogRepository, films CatalogFilmRepository) *GetFilmPersonsUseCase {
	return &GetFilmPersonsUseCase{persons: persons, films: films}
}

func (uc *GetFilmPersonsUseCase) Run(ctx context.Context, id domain.FilmID) ([]domain.PersonCredit, error) {
	if id <= 0 {
		return nil, domain.ErrInvalidFilmInput
	}

	if _, err := uc.films.GetFilmByID(ctx, id); err != nil {
		return nil, err
	}

	return uc.persons.FilmPersons(ctx, int64(id))
}

type CatalogFilmCredit struct {
	Film      domain.Film
	Role      string
	Character string
}

type PersonCatalogOutput struct {
	Person domain.CatalogPerson
	Films  []CatalogFilmCredit
}

type GetPersonCatalogUseCase struct {
	persons domain.PersonCatalogRepository
	films   CatalogFilmRepository
}

func NewGetPersonCatalogUseCase(persons domain.PersonCatalogRepository, films CatalogFilmRepository) *GetPersonCatalogUseCase {
	return &GetPersonCatalogUseCase{persons: persons, films: films}
}

func (uc *GetPersonCatalogUseCase) Run(ctx context.Context, id domain.PersonID) (*PersonCatalogOutput, error) {
	if id <= 0 {
		return nil, domain.ErrInvalidPersonID
	}

	person, credits, err := uc.persons.Get(ctx, int64(id))
	if err != nil {
		return nil, err
	}

	result := &PersonCatalogOutput{Person: person, Films: make([]CatalogFilmCredit, 0, len(credits))}

	for _, credit := range credits {
		film, err := uc.films.GetFilmByID(ctx, credit.FilmID)
		if err != nil {
			return nil, err
		}

		result.Films = append(result.Films, CatalogFilmCredit{Film: film, Role: credit.Role, Character: credit.Character})
	}

	return result, nil
}
