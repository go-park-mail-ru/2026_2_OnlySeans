package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type DeleteFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
	logger  logger.Logger
}

func NewDeleteFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository, appLogger logger.Logger) (*DeleteFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &DeleteFilmRatingUseCase{ratings: ratings, films: films, logger: appLogger}, nil
}

func (uc *DeleteFilmRatingUseCase) Run(ctx context.Context, input DeleteFilmRatingInput) error {
	fields := map[string]any{
		"account_id": input.AccountID,
		"film_id":    input.FilmID,
	}
	uc.logger.Debug("delete film rating started", fields)
	uc.logger.Debug("checking film before rating deletion", fields)

	film, err := uc.films.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		uc.logger.Debug("film lookup before rating deletion failed", map[string]any{
			"film_id": input.FilmID,
			"error":   err,
		})
		if errors.Is(err, domain.ErrFilmNotFound) {
			return domain.ErrFilmNotFound
		}
		return err
	}
	uc.logger.Debug("film found before rating deletion", map[string]any{
		"film_id": film.ID,
		"title":   film.Title,
	})

	uc.logger.Debug("deleting film rating", fields)
	if err := uc.ratings.DeleteFilmRating(ctx, input.AccountID, input.FilmID); err != nil {
		uc.logger.Debug("film rating deletion failed", map[string]any{
			"account_id": input.AccountID,
			"film_id":    input.FilmID,
			"error":      err,
		})
		return err
	}
	uc.logger.Debug("film rating deleted", fields)
	uc.logger.Info("film rating deleted", fields)
	return nil
}
