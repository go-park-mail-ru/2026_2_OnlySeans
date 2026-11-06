package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type SetFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
	logger  logger.Logger
}

func NewSetFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository, appLogger logger.Logger) (*SetFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &SetFilmRatingUseCase{ratings: ratings, films: films, logger: appLogger}, nil
}

func (uc *SetFilmRatingUseCase) Run(ctx context.Context, input SetFilmRatingInput) (*SetFilmRatingOutput, error) {
	fields := map[string]any{
		"account_id": input.AccountID,
		"film_id":    input.FilmID,
		"score":      input.Score,
	}
	uc.logger.Debug("set film rating started", fields)
	if input.Score < domain.MinFilmRating || input.Score > domain.MaxFilmRating {
		err := domain.InvalidFilmRatingLimitsError()
		uc.logger.Debug("film rating validation failed", map[string]any{
			"film_id": input.FilmID,
			"score":   input.Score,
			"error":   err,
		})
		return nil, err
	}
	uc.logger.Debug("checking film before setting rating", map[string]any{"film_id": input.FilmID})
	film, err := uc.films.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		uc.logger.Debug("film lookup before setting rating failed", map[string]any{
			"film_id": input.FilmID,
			"error":   err,
		})
		if errors.Is(err, domain.ErrFilmNotFound) {
			return nil, domain.ErrFilmNotFound
		}
		return nil, err
	}
	uc.logger.Debug("film found before setting rating", map[string]any{
		"film_id": film.ID,
		"title":   film.Title,
	})

	uc.logger.Debug("saving film rating", fields)
	rating, err := uc.ratings.SetFilmRating(ctx, input.AccountID, input.FilmID, input.Score)
	if err != nil {
		uc.logger.Debug("saving film rating failed", map[string]any{
			"film_id": input.FilmID,
			"error":   err,
		})
		switch {
		case errors.Is(err, domain.ErrFilmNotFound):
			return nil, domain.ErrFilmNotFound
		case errors.Is(err, domain.ErrInvalidFilmRating):
			return nil, err
		default:
			return nil, err
		}
	}
	uc.logger.Debug("film rating saved", map[string]any{
		"account_id": rating.AccountID,
		"film_id":    rating.FilmID,
		"score":      rating.Score,
	})
	uc.logger.Info("film rating set", map[string]any{
		"account_id": rating.AccountID,
		"film_id":    rating.FilmID,
		"score":      rating.Score,
	})
	return NewSetFilmRatingOutput(rating), nil
}
