package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type DeleteFilmUseCase struct {
	repository domain.FilmDeleter
}

func NewDeleteFilmUseCase(repository domain.FilmDeleter) (*DeleteFilmUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &DeleteFilmUseCase{repository: repository}, nil
}

func (uc *DeleteFilmUseCase) Run(ctx context.Context, id domain.FilmID) error {
	if err := uc.repository.DeleteFilm(ctx, id); err != nil {
		return fmt.Errorf("delete film: %w", err)
	}
	return nil
}
