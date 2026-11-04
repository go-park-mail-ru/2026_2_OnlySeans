package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type GetFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
}

func NewGetFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository) (*GetFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &GetFilmRatingUseCase{ratings: ratings, films: films}, nil
}

func (uc *GetFilmRatingUseCase) Run(ctx context.Context, input GetFilmRatingInput) (*GetFilmRatingOutput, error) {
	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		if errors.Is(err, domain.ErrFilmNotFound) {
			return nil, domain.ErrFilmNotFound
		}
		return nil, err
	}

	summary, err := uc.ratings.GetSummary(ctx, input.FilmID)
	if err != nil {
		return nil, err
	}
	return NewGetFilmRatingOutput(summary), nil
}
