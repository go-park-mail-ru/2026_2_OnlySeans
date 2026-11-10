package files

import (
	"context"
	"errors"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type FilmPosterUseCase struct {
	posters domain.PosterRepository
	storage AvatarURLStorage
	log     logger.Logger
}

func NewFilmPosterUseCase(posters domain.PosterRepository, storage AvatarURLStorage, log logger.Logger) (*FilmPosterUseCase, error) {
	if posters == nil {
		return nil, errors.New("poster repository is nil")
	}
	if storage == nil {
		return nil, errors.New("poster URL storage is nil")
	}
	if log == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &FilmPosterUseCase{posters: posters, storage: storage, log: log}, nil
}

func (uc *FilmPosterUseCase) GetURL(ctx context.Context, filmID domain.FilmID) (string, error) {
	if filmID <= 0 {
		return "", ErrInvalidPosterFilmID
	}
	poster, err := uc.posters.GetPoster(ctx, filmID)
	if err != nil {
		logging.Failure(ctx, uc.log, "get film poster failed", err, map[string]any{"film_id": filmID})
		return "", err
	}
	if poster == nil {
		return "", domain.ErrFileNotFound
	}
	if poster.ExternalURL != nil {
		return *poster.ExternalURL, nil
	}
	return uc.storage.GetURL(ctx, poster.Filename)
}

func (uc *FilmPosterUseCase) Delete(ctx context.Context, filmID domain.FilmID) error {
	if filmID <= 0 {
		return ErrInvalidPosterFilmID
	}
	if err := uc.posters.DeletePoster(ctx, filmID); err != nil {
		logging.Failure(ctx, uc.log, "delete film poster failed", err, map[string]any{"film_id": filmID})
		return err
	}
	uc.log.InfoContext(ctx, "film poster detached", map[string]any{"film_id": filmID})
	return nil
}
