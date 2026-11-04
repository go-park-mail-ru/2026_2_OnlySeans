package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type SetFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
}

func NewSetFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository) (*SetFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &SetFilmRatingUseCase{ratings: ratings, films: films}, nil
}

func (uc *SetFilmRatingUseCase) Run(ctx context.Context, input SetFilmRatingInput) (*SetFilmRatingOutput, error) {
	if input.Score < domain.MinFilmRating || input.Score > domain.MaxFilmRating {
		return nil, domain.ErrInvalidFilmRating
	}
	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		if errors.Is(err, domain.ErrFilmNotFound) {
			return nil, domain.ErrFilmNotFound
		}
		return nil, err
	}

	rating, err := uc.ratings.Set(ctx, input.AccountID, input.FilmID, input.Score)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrFilmNotFound):
			return nil, domain.ErrFilmNotFound
		case errors.Is(err, domain.ErrInvalidFilmRating):
			return nil, domain.ErrInvalidFilmRating
		default:
			return nil, err
		}
	}
	return NewSetFilmRatingOutput(rating), nil
}
