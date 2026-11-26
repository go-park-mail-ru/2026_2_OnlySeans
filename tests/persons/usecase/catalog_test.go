package persons_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/persons/tests/mocks"
)

func TestCatalogGetFilmPersons(t *testing.T) {
	failure := errors.New("database failure")
	people := []domain.PersonCredit{
		{
			Person: domain.CatalogPerson{
				ID:   7,
				Name: "Director",
			},
			Role: "director",
		},
		{
			Person: domain.CatalogPerson{
				ID:   8,
				Name: "Actor",
			},
			Role:      "actor",
			Character: "Hero",
		},
	}

	tests := []struct {
		name    string
		id      int64
		people  []domain.PersonCredit
		repoErr error
		wantErr error
	}{
		{
			name:   "preserves people and order",
			id:     42,
			people: people,
		},
		{
			name:   "no credits",
			id:     42,
			people: []domain.PersonCredit{},
		},
		{
			name:    "zero ID",
			wantErr: domain.ErrInvalidFilmID,
		},
		{
			name:    "negative ID",
			id:      -1,
			wantErr: domain.ErrInvalidFilmID,
		},
		{
			name:    "database failure",
			id:      42,
			repoErr: failure,
			wantErr: failure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonCatalogRepository(gomock.NewController(t))
			if tt.id > 0 {
				repo.EXPECT().FilmPersons(t.Context(), tt.id).Return(tt.people, tt.repoErr)
			}

			uc := usecase.NewCatalogUseCase(repo)
			got, err := uc.GetFilmPersons(t.Context(), tt.id)
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr != nil {
				require.Nil(t, got)
			} else {
				require.Equal(t, tt.people, got)
			}
		})
	}
}

func TestCatalogGetPersonCatalog(t *testing.T) {
	failure := errors.New("database failure")
	person := domain.CatalogPerson{
		ID:   7,
		Name: "Person",
	}
	films := []domain.FilmCredit{
		{
			FilmID: 42,
			Role:   "director",
		},
		{
			FilmID:    42,
			Role:      "actor",
			Character: "Hero",
		},
	}

	tests := []struct {
		name    string
		id      int64
		films   []domain.FilmCredit
		repoErr error
		wantErr error
	}{
		{
			name:  "preserves multiple roles for the same film",
			id:    7,
			films: films,
		},
		{
			name:  "person without films",
			id:    7,
			films: []domain.FilmCredit{},
		},
		{
			name:    "zero ID",
			wantErr: domain.ErrInvalidPersonID,
		},
		{
			name:    "negative ID",
			id:      -1,
			wantErr: domain.ErrInvalidPersonID,
		},
		{
			name:    "missing person",
			id:      7,
			repoErr: domain.ErrPersonNotFound,
			wantErr: domain.ErrPersonNotFound,
		},
		{
			name:    "database failure",
			id:      7,
			repoErr: failure,
			wantErr: failure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonCatalogRepository(gomock.NewController(t))
			if tt.id > 0 {
				repo.EXPECT().Get(t.Context(), tt.id).Return(person, tt.films, tt.repoErr)
			}

			uc := usecase.NewCatalogUseCase(repo)
			got, err := uc.GetPersonCatalog(t.Context(), tt.id)
			require.ErrorIs(t, err, tt.wantErr)

			if tt.wantErr != nil {
				require.Empty(t, got)
			} else {
				require.Equal(t, person, got.Person)
				require.Equal(t, tt.films, got.Films)
			}
		})
	}
}
