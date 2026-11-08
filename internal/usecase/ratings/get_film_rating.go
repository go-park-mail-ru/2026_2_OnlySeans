package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetFilmRatingUseCase struct {
	ratings domain.FilmRatingRepository
	films   domain.FilmRepository
	logger  logger.Logger
}

func NewGetFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository, appLogger logger.Logger) (*GetFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &GetFilmRatingUseCase{
		ratings: ratings,
		films:   films,
		logger:  appLogger,
	}, nil
}

func (uc *GetFilmRatingUseCase) Run(ctx context.Context, input GetFilmRatingInput) (*GetFilmRatingOutput, error) {
	fields := map[string]any{"film_id": input.FilmID}
	uc.logger.Debug("get film rating started", fields)
	uc.logger.Debug("checking film before getting rating", fields)
	film, err := uc.films.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		logging.Failure(uc.logger, "film lookup before getting rating failed", err, fields)
		if errors.Is(err, domain.ErrFilmNotFound) {
			return nil, domain.ErrFilmNotFound
		}
		return nil, err
	}
	uc.logger.Debug("film found before getting rating", map[string]any{
		"film_id": film.ID,
		"title":   film.Title,
	})

	uc.logger.Debug("loading film rating summary", fields)
	summary, err := uc.ratings.GetFilmRatingSummary(ctx, input.FilmID)
	if err != nil {
		logging.Failure(uc.logger, "loading film rating summary failed", err, fields)
		return nil, err
	}
	uc.logger.Debug("film rating summary loaded", map[string]any{
		"film_id":       summary.FilmID,
		"average_score": summary.AverageScore,
		"votes":         summary.Votes,
	})
	return NewGetFilmRatingOutput(summary), nil
}
