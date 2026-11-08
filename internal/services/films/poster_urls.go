package films

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type PosterURLStorage interface {
	GetURL(ctx context.Context, key string) (string, error)
}

type RepositoryWithPosterURLs struct {
	domain.FilmsRepository
	storage PosterURLStorage
	logger  logger.Logger
}

func NewRepositoryWithPosterURLs(repository domain.FilmsRepository, storage PosterURLStorage, appLogger logger.Logger) (*RepositoryWithPosterURLs, error) {
	if repository == nil {
		return nil, domain.ErrNilFilmsRepo
	}
	if storage == nil {
		return nil, errors.New("poster URL storage is nil")
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &RepositoryWithPosterURLs{FilmsRepository: repository, storage: storage, logger: appLogger}, nil
}

func (r *RepositoryWithPosterURLs) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	film, err := r.FilmsRepository.GetFilmByID(ctx, id)
	if err != nil {
		return domain.Film{}, err
	}
	if err := r.attachPosterURL(ctx, &film); err != nil {
		return domain.Film{}, err
	}
	return film, nil
}

func (r *RepositoryWithPosterURLs) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	films, total, err := r.FilmsRepository.ListFilms(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result, err := r.attachPosterURLs(ctx, films)
	return result, total, err
}

func (r *RepositoryWithPosterURLs) ListCollectionFilms(ctx context.Context, id domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	films, total, err := r.FilmsRepository.ListCollectionFilms(ctx, id, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result, err := r.attachPosterURLs(ctx, films)
	return result, total, err
}

func (r *RepositoryWithPosterURLs) attachPosterURLs(ctx context.Context, films []domain.Film) ([]domain.Film, error) {
	result := slices.Clone(films)
	for i := range result {
		if err := r.attachPosterURL(ctx, &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *RepositoryWithPosterURLs) attachPosterURL(ctx context.Context, film *domain.Film) error {
	if film.PosterURL != nil || film.PosterStorageKey == nil {
		return nil
	}
	url, err := r.storage.GetURL(ctx, *film.PosterStorageKey)
	if errors.Is(err, domain.ErrFileNotFound) {
		r.logger.Warn("film poster object not found", map[string]any{"film_id": film.ID, "storage_key": *film.PosterStorageKey})
		return nil
	}
	if err != nil {
		return fmt.Errorf("get poster URL for film %d: %w", film.ID, err)
	}
	film.PosterURL = &url
	r.logger.Debug("film poster URL generated", map[string]any{"film_id": film.ID})
	return nil
}
