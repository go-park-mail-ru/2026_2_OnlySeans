package usecase

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type DeleteFilmRatingUseCase struct {
	ratings   domain.FilmRatingRepository
	films     domain.FilmRepository
	appLogger logger.Logger
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
	return &DeleteFilmRatingUseCase{ratings: ratings, films: films, appLogger: appLogger}, nil
}

func (uc *DeleteFilmRatingUseCase) Run(ctx context.Context, input DeleteFilmRatingInput) error {
	fields := map[string]any{
		"account_id": input.AccountID,
		"film_id":    input.FilmID,
	}
	uc.appLogger.DebugContext(ctx, "delete film rating started", fields)
	uc.appLogger.DebugContext(ctx, "checking film before rating deletion", fields)

	film, err := uc.films.GetFilmByID(ctx, input.FilmID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "film lookup before rating deletion failed", err, fields)
		if errors.Is(err, domain.ErrFilmNotFound) {
			return domain.ErrFilmNotFound
		}
		return err
	}
	uc.appLogger.DebugContext(ctx, "film found before rating deletion", map[string]any{
		"film_id": film.ID,
		"title":   film.Title,
	})

	uc.appLogger.DebugContext(ctx, "deleting film rating", fields)
	if err := uc.ratings.DeleteFilmRating(ctx, input.AccountID, input.FilmID); err != nil {
		logging.Failure(ctx, uc.appLogger, "film rating deletion failed", err, fields)
		return err
	}
	uc.appLogger.InfoContext(ctx, "film rating deleted", fields)
	return nil
}
