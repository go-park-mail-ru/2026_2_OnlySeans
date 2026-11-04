package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

type DeleteFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
}

func NewDeleteFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository) (*DeleteFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	return &DeleteFilmRatingUseCase{ratings: ratings, films: films}, nil
}

func (uc *DeleteFilmRatingUseCase) Run(ctx context.Context, input DeleteFilmRatingInput) error {
	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		if errors.Is(err, domain.ErrFilmNotFound) {
			return domain.ErrFilmNotFound
		}
		return err
	}

	return uc.ratings.Delete(ctx, input.AccountID, input.FilmID)
}
