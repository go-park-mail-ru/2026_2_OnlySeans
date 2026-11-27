package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
)

type PosterURLStorage interface {
	GetURL(ctx context.Context, key string) (string, error)
}

type PosterURLReader struct {
	films       repository.FilmReader
	lister      repository.FilmLister
	collections repository.CollectionReader
	storage     PosterURLStorage
}

func NewPosterURLReader(films repository.FilmReader, lister repository.FilmLister, collections repository.CollectionReader, storage PosterURLStorage) (*PosterURLReader, error) {
	if films == nil || lister == nil || collections == nil || storage == nil {
		return nil, errors.New("film reader, film lister, collection reader and poster URL storage are required")
	}

	return &PosterURLReader{
		films:       films,
		lister:      lister,
		collections: collections,
		storage:     storage,
	}, nil
}

func (r *PosterURLReader) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	film, err := r.films.GetFilmByID(ctx, id)
	if err != nil {
		return domain.Film{}, err
	}

	if err := r.attachPosterURL(ctx, &film); err != nil {
		return domain.Film{}, err
	}

	return film, nil
}

func (r *PosterURLReader) FilterFilms(ctx context.Context, filter domain.FilmFilter, limit, offset int) ([]domain.Film, int, error) {
	films, total, err := r.lister.FilterFilms(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result, err := r.attachPosterURLs(ctx, films)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *PosterURLReader) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	return r.FilterFilms(ctx, domain.FilmFilter{}, limit, offset)
}

func (r *PosterURLReader) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error) {
	return r.collections.ListCollections(ctx, limit, offset)
}

func (r *PosterURLReader) GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error) {
	return r.collections.GetCollectionByID(ctx, id)
}

func (r *PosterURLReader) ListCollectionFilms(ctx context.Context, id domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	films, total, err := r.collections.ListCollectionFilms(ctx, id, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	result, err := r.attachPosterURLs(ctx, films)
	if err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *PosterURLReader) attachPosterURLs(ctx context.Context, films []domain.Film) ([]domain.Film, error) {
	result := slices.Clone(films)

	for i := range result {
		if err := r.attachPosterURL(ctx, &result[i]); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *PosterURLReader) attachPosterURL(ctx context.Context, film *domain.Film) error {
	if film.PosterURL != nil || film.PosterStorageKey == nil {
		return nil
	}

	url, err := r.storage.GetURL(ctx, *film.PosterStorageKey)
	if errors.Is(err, domain.ErrFileNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("get poster URL for film %d: %w", film.ID, err)
	}

	film.PosterURL = &url

	return nil
}
