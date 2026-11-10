package films_test

import (
	"context"
	"errors"
	"testing"

	entities "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"go.uber.org/mock/gomock"
)

func TestListFilmsUseCaseRun(t *testing.T) {
	ctx := context.Background()
	expectedFilms := []entities.Film{{ID: 1}}

	repository := mocks.NewMockFilmRepository(gomock.NewController(t))
	repository.EXPECT().ListFilms(ctx, 2, 1).Return(expectedFilms, 5, nil)
	uc, err := filmsusecase.NewListFilmsUseCase(repository, testutil.DiscardLogger())
	if err != nil {
		t.Fatalf("filmsusecase.NewListFilmsUseCase: %v", err)
	}

	result, err := uc.Run(ctx, filmsusecase.NewListFilmsInput(2, 1))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Films) != len(expectedFilms) || result.Films[0].ID != expectedFilms[0].ID {
		t.Fatalf("films = %v, want %v", result.Films, expectedFilms)
	}

	if result.Total != 5 {
		t.Fatalf("total = %d, want %d", result.Total, 5)
	}

}

func TestListFilmsUseCaseRunReturnsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("storage unavailable")

	repository := mocks.NewMockFilmRepository(gomock.NewController(t))
	repository.EXPECT().ListFilms(gomock.Any(), 10, 0).Return(nil, 0, repositoryErr)
	uc, err := filmsusecase.NewListFilmsUseCase(repository, testutil.DiscardLogger())
	if err != nil {
		t.Fatalf("filmsusecase.NewListFilmsUseCase: %v", err)
	}

	result, err := uc.Run(context.Background(), filmsusecase.NewListFilmsInput(10, 0))

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("Run error = %v, want wrapped %v", err, repositoryErr)
	}

	if result != nil {
		t.Fatalf("Run result = %v, want nil on error", result)
	}
}

func TestNewListFilmsUseCaseRejectsNilRepository(t *testing.T) {
	if _, err := filmsusecase.NewListFilmsUseCase(nil, testutil.DiscardLogger()); err == nil {
		t.Fatal("filmsusecase.NewListFilmsUseCase(nil) error = nil, want error")
	}
}

func TestNewListFilmsUseCaseRejectsNilLogger(t *testing.T) {
	repository := mocks.NewMockFilmRepository(gomock.NewController(t))

	if _, err := filmsusecase.NewListFilmsUseCase(repository, nil); err == nil {
		t.Fatal("filmsusecase.NewListFilmsUseCase with nil logger: expected error")
	}
}
