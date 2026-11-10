package usecase

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GetMyFilmRatingInput struct {
	AccountID domain.UserID
	FilmID    domain.FilmID
}

type GetMyFilmRatingOutput struct{ Score *int }

type GetMyFilmRatingUseCase struct {
	ratings   domain.FilmRatingRepository
	films     domain.FilmRepository
	appLogger logger.Logger
}

func NewGetMyFilmRatingUseCase(ratings domain.FilmRatingRepository, films domain.FilmRepository, appLogger logger.Logger) (*GetMyFilmRatingUseCase, error) {
	if ratings == nil {
		return nil, domain.ErrNilFilmRatingsRepo
	}
	if films == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &GetMyFilmRatingUseCase{ratings: ratings, films: films, appLogger: appLogger}, nil
}

func (uc *GetMyFilmRatingUseCase) Run(ctx context.Context, input GetMyFilmRatingInput) (*GetMyFilmRatingOutput, error) {
	if input.AccountID <= 0 {
		return nil, domain.ErrUnauthorized
	}
	if input.FilmID <= 0 {
		return nil, domain.ErrFilmNotFound
	}
	fields := map[string]any{"account_id": input.AccountID, "film_id": input.FilmID}
	if _, err := uc.films.GetFilmByID(ctx, input.FilmID); err != nil {
		logging.Failure(ctx, uc.appLogger, "check personal rating film failed", err, fields)
		return nil, err
	}
	score, err := uc.ratings.GetMyScore(ctx, input.AccountID, input.FilmID)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get personal rating failed", err, fields)
		return nil, err
	}
	return &GetMyFilmRatingOutput{Score: score}, nil
}
