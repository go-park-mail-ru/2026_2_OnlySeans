package usecase

import (
	"context"
	"errors"
	"testing"

	entities "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type filmRepositoryStub struct {
	films  []entities.Film
	total  int
	err    error
	ctx    context.Context
	limit  int
	offset int
}

func (r *filmRepositoryStub) ListFilms(ctx context.Context, limit, offset int) ([]entities.Film, int, error) {
	r.ctx = ctx
	r.limit = limit
	r.offset = offset
	return r.films, r.total, r.err
}

func (r *filmRepositoryStub) GetFilmByID(context.Context, entities.FilmID) (entities.Film, error) {
	return entities.Film{}, r.err
}

func TestListFilmsUseCaseRun(t *testing.T) {
	ctx := context.Background()
	expectedFilms := []entities.Film{{ID: 1}}
	repository := &filmRepositoryStub{
		films: expectedFilms,
		total: 5,
	}
	uc, err := NewListFilmsUseCase(repository)
	if err != nil {
		t.Fatalf("NewListFilmsUseCase: %v", err)
	}

	result, err := uc.Run(ctx, NewListFilmsInput(2, 1))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Films) != len(expectedFilms) || result.Films[0].ID != expectedFilms[0].ID {
		t.Fatalf("films = %v, want %v", result.Films, expectedFilms)
	}
	if result.Total != repository.total {
		t.Fatalf("total = %d, want %d", result.Total, repository.total)
	}
	if repository.ctx != ctx || repository.limit != 2 || repository.offset != 1 {
		t.Fatalf("repository received ctx=%v limit=%d offset=%d", repository.ctx, repository.limit, repository.offset)
	}
}

func TestListFilmsUseCaseRunReturnsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("storage unavailable")
	uc, err := NewListFilmsUseCase(&filmRepositoryStub{err: repositoryErr})
	if err != nil {
		t.Fatalf("NewListFilmsUseCase: %v", err)
	}

	result, err := uc.Run(context.Background(), NewListFilmsInput(10, 0))
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Run error = %v, want wrapped %v", err, repositoryErr)
	}
	if result != nil {
		t.Fatalf("Run result = %v, want nil on error", result)
	}
}

func TestNewListFilmsUseCaseRejectsNilRepository(t *testing.T) {
	if _, err := NewListFilmsUseCase(nil); err == nil {
		t.Fatal("NewListFilmsUseCase(nil) error = nil, want error")
	}
}
