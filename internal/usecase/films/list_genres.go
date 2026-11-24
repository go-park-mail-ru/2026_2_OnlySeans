package usecase

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ListGenresUseCase struct {
	repository domain.GenreRepository
	appLogger  logger.Logger
}

func NewListGenresUseCase(repository domain.GenreRepository, appLogger logger.Logger) (*ListGenresUseCase, error) {
	if repository == nil {
		return nil, domain.ErrNilGenresRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &ListGenresUseCase{
		repository: repository,
		appLogger:  appLogger,
	}, nil
}

func (uc *ListGenresUseCase) Run(ctx context.Context) ([]domain.Genre, error) {
	genres, err := uc.repository.ListGenres(ctx)

	if err != nil {
		return nil, fmt.Errorf("list genres: %w", err)
	}
	return genres, nil
}
