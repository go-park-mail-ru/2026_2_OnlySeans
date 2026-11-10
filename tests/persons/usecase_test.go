package persons_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"go.uber.org/mock/gomock"
)

func newTestPerson() *domain.Person {
	return &domain.Person{
		ID:        1,
		FirstName: "Ada",
		LastName:  "Lovelace",
	}
}

func newTestLogger() logger.Logger {
	return testutil.DiscardLogger()
}

// CREATE

func TestCreatePersonUseCase(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name          string
		input         domain.PersonInput
		repoErr       error
		wantErr       error
		wantCalls     int
		wantFirstName string
	}{
		{
			name: "success",
			input: domain.PersonInput{
				FirstName: "Ada",
				LastName:  "Lovelace",
			},
			wantCalls:     1,
			wantFirstName: "Ada",
		},
		{
			name: "trim spaces",
			input: domain.PersonInput{
				FirstName: " Ada ",
				LastName:  " Lovelace ",
			},
			wantCalls:     1,
			wantFirstName: "Ada",
		},
		{
			name: "invalid first name",
			input: domain.PersonInput{
				FirstName: "",
				LastName:  "Lovelace",
			},
			wantErr:   domain.ErrInvalidFirstName,
			wantCalls: 0,
		},
		{
			name: "repository error",
			input: domain.PersonInput{
				FirstName: "Ada",
				LastName:  "Lovelace",
			},
			repoErr:       repoErr,
			wantErr:       repoErr,
			wantCalls:     1,
			wantFirstName: "Ada",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonRepository(gomock.NewController(t))
			person := newTestPerson()
			expected := tt.input
			expected.FirstName = strings.TrimSpace(expected.FirstName)
			expected.LastName = strings.TrimSpace(expected.LastName)
			repo.EXPECT().Create(gomock.Any(), expected).Return(person.ID, tt.repoErr).Times(tt.wantCalls)

			uc, err := personusecase.NewCreatePersonUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			result, err := uc.Run(
				context.Background(),
				personusecase.CreatePersonInput{Person: tt.input},
			)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}

				if result.ID != person.ID {
					t.Errorf("expected ID %d, got %d", person.ID, result.ID)
				}
			}
		})
	}
}

// GET

func TestGetPersonUseCase(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name      string
		id        domain.PersonID
		repoErr   error
		wantErr   error
		wantCalls int
	}{
		{
			name:      "success",
			id:        1,
			wantCalls: 1,
		},
		{
			name:      "person not found",
			id:        2,
			repoErr:   domain.ErrPersonNotFound,
			wantErr:   domain.ErrPersonNotFound,
			wantCalls: 1,
		},
		{
			name:      "repository error",
			id:        1,
			repoErr:   repoErr,
			wantErr:   repoErr,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonRepository(gomock.NewController(t))
			person := newTestPerson()
			person.ID = tt.id
			repo.EXPECT().GetByID(gomock.Any(), tt.id).Return(person, tt.repoErr).Times(tt.wantCalls)

			uc, err := personusecase.NewGetPersonUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			result, err := uc.Run(context.Background(), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}

				if result.Person.ID != tt.id {
					t.Errorf("expected person ID %d, got %d",
						tt.id, result.Person.ID)
				}
			}
		})
	}
}

// LIST

func TestListPersonsUseCase(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name    string
		persons []domain.Person
		total   int
		repoErr error
		wantErr error
	}{
		{
			name: "success",
			persons: []domain.Person{
				*newTestPerson(),
				{ID: 2, FirstName: "Grace", LastName: "Hopper"},
			},
			total: 2,
		},
		{
			name:    "empty list",
			persons: []domain.Person{},
			total:   0,
		},
		{
			name:    "repository error",
			repoErr: repoErr,
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonRepository(gomock.NewController(t))
			repo.EXPECT().List(gomock.Any(), 10, 0).Return(tt.persons, tt.total, tt.repoErr).Times(1)

			uc, err := personusecase.NewListPersonsUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			result, err := uc.Run(context.Background(),
				personusecase.ListPersonsInput{
					Limit:  10,
					Offset: 0,
				},
			)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}

				if len(result.Persons) != len(tt.persons) {
					t.Errorf("expected %d persons, got %d",
						len(tt.persons), len(result.Persons))
				}

				if result.Total != tt.total {
					t.Errorf("expected total %d, got %d",
						tt.total, result.Total)
				}
			}
		})
	}
}

// UPDATE

func TestUpdatePersonUseCase(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name      string
		input     domain.PersonInput
		repoErr   error
		wantErr   error
		wantCalls int
		wantName  string
	}{
		{
			name: "success",
			input: domain.PersonInput{
				FirstName: "Grace",
				LastName:  "Hopper",
			},
			wantCalls: 1,
			wantName:  "Grace",
		},
		{
			name: "trim spaces",
			input: domain.PersonInput{
				FirstName: " Grace ",
				LastName:  " Hopper ",
			},
			wantCalls: 1,
			wantName:  "Grace",
		},
		{
			name: "invalid first name",
			input: domain.PersonInput{
				FirstName: "",
				LastName:  "Hopper",
			},
			wantErr:   domain.ErrInvalidFirstName,
			wantCalls: 0,
		},
		{
			name: "repository error",
			input: domain.PersonInput{
				FirstName: "Grace",
				LastName:  "Hopper",
			},
			repoErr:   repoErr,
			wantErr:   repoErr,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonRepository(gomock.NewController(t))
			person := newTestPerson()
			expected := tt.input
			expected.FirstName = strings.TrimSpace(expected.FirstName)
			expected.LastName = strings.TrimSpace(expected.LastName)
			repo.EXPECT().Update(gomock.Any(), domain.PersonID(1), expected).Return(person, tt.repoErr).Times(tt.wantCalls)

			uc, err := personusecase.NewUpdatePersonUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			result, err := uc.Run(context.Background(),
				personusecase.UpdatePersonInput{
					PersonID: 1,
					Person:   tt.input,
				},
			)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}
			}
		})
	}
}

// DELETE

func TestDeletePersonUseCase(t *testing.T) {
	repoErr := errors.New("database error")

	tests := []struct {
		name    string
		id      domain.PersonID
		repoErr error
		wantErr error
	}{
		{
			name: "success",
			id:   1,
		},
		{
			name:    "person not found",
			id:      2,
			repoErr: domain.ErrPersonNotFound,
			wantErr: domain.ErrPersonNotFound,
		},
		{
			name:    "repository error",
			id:      1,
			repoErr: repoErr,
			wantErr: repoErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPersonRepository(gomock.NewController(t))
			repo.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.repoErr).Times(1)

			uc, err := personusecase.NewDeletePersonUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			err = uc.Run(context.Background(),
				personusecase.DeletePersonInput{
					PersonID: tt.id,
				},
			)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
