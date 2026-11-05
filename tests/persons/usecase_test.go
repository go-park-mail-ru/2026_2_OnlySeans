package persons_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type testLogger struct{}

func (l *testLogger) Info(string, map[string]any)  {}
func (l *testLogger) Debug(string, map[string]any) {}
func (l *testLogger) Warn(string, map[string]any)  {}
func (l *testLogger) Error(string, map[string]any) {}

// Fake-репозиторий

type fakePersonsRepository struct {
	person  *domain.Person
	persons []domain.Person
	total   int
	err     error

	createCalls int
	getCalls    int
	listCalls   int
	updateCalls int
	deleteCalls int

	createInput domain.PersonInput
	updateInput domain.PersonInput
	getID       domain.PersonID
	updateID    domain.PersonID
	deleteID    domain.PersonID
}

func (r *fakePersonsRepository) Create(
	ctx context.Context,
	input domain.PersonInput,
) (domain.PersonID, error) {
	r.createCalls++
	r.createInput = input
	if r.err != nil {
		return 0, r.err
	}
	return r.person.ID, nil
}

func (r *fakePersonsRepository) GetByID(
	ctx context.Context,
	id domain.PersonID,
) (*domain.Person, error) {
	r.getCalls++
	r.getID = id
	return r.person, r.err
}

func (r *fakePersonsRepository) List(
	ctx context.Context,
	limit, offset int,
) ([]domain.Person, int, error) {
	r.listCalls++
	return r.persons, r.total, r.err
}

func (r *fakePersonsRepository) Update(
	ctx context.Context,
	id domain.PersonID,
	input domain.PersonInput,
) (*domain.Person, error) {
	r.updateCalls++
	r.updateID = id
	r.updateInput = input
	return r.person, r.err
}

func (r *fakePersonsRepository) Delete(
	ctx context.Context,
	id domain.PersonID,
) error {
	r.deleteCalls++
	r.deleteID = id
	return r.err
}

func newTestPerson() *domain.Person {
	return &domain.Person{
		ID:        1,
		FirstName: "Ada",
		LastName:  "Lovelace",
	}
}

func newTestLogger() logger.Logger {
	return &testLogger{}
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
			repo := &fakePersonsRepository{
				person: newTestPerson(),
				err:    tt.repoErr,
			}

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

			if repo.createCalls != tt.wantCalls {
				t.Errorf("expected %d calls, got %d", tt.wantCalls, repo.createCalls)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}

				if result.ID != repo.person.ID {
					t.Errorf("expected ID %d, got %d", repo.person.ID, result.ID)
				}
				if repo.getCalls != 0 {
					t.Error("creation must not load the person")
				}
			}

			if tt.wantCalls > 0 && repo.createInput.FirstName != tt.wantFirstName {
				t.Errorf("expected repository input first name %q, got %q",
					tt.wantFirstName, repo.createInput.FirstName)
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
			repo := &fakePersonsRepository{
				person: newTestPerson(),
				err:    tt.repoErr,
			}

			uc, err := personusecase.NewGetPersonUseCase(repo, newTestLogger())
			if err != nil {
				t.Fatal(err)
			}

			result, err := uc.Run(context.Background(), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if repo.getCalls != tt.wantCalls {
				t.Errorf("expected %d calls, got %d", tt.wantCalls, repo.getCalls)
			}

			if repo.getID != tt.id {
				t.Errorf("expected ID %d, got %d", tt.id, repo.getID)
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
			repo := &fakePersonsRepository{
				persons: tt.persons,
				total:   tt.total,
				err:     tt.repoErr,
			}

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

			if repo.listCalls != 1 {
				t.Errorf("expected 1 call, got %d", repo.listCalls)
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
			repo := &fakePersonsRepository{
				person: newTestPerson(),
				err:    tt.repoErr,
			}

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

			if repo.updateCalls != tt.wantCalls {
				t.Errorf("expected %d calls, got %d", tt.wantCalls, repo.updateCalls)
			}

			if tt.wantErr == nil {
				if result == nil {
					t.Fatal("expected result, got nil")
				}

				if repo.updateInput.FirstName != tt.wantName {
					t.Errorf("expected first name %q, got %q",
						tt.wantName, repo.updateInput.FirstName)
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
			repo := &fakePersonsRepository{
				err: tt.repoErr,
			}

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

			if repo.deleteCalls != 1 {
				t.Errorf("expected 1 call, got %d", repo.deleteCalls)
			}

			if repo.deleteID != tt.id {
				t.Errorf("expected ID %d, got %d", tt.id, repo.deleteID)
			}
		})
	}
}
