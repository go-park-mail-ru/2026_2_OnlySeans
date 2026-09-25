package films

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
)

var (
	ErrFilmNotFound            = errors.New("фильм не найден")
	ErrFilmExists              = errors.New("фильм уже существует")
	ErrCollectionNotFound      = errors.New("подборка не найдена")
	ErrCollectionExists        = errors.New("подборка уже существует")
	ErrDuplicateCollectionFilm = errors.New("фильм уже есть в подборке")
)

type Repository interface {
	ListFilms(ctx context.Context, limit, offset int) ([]Film, int, error)
	ListCollections(ctx context.Context) ([]Collection, error)
	GetCollectionBySlug(ctx context.Context, slug string) (Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID int64) ([]Film, error)
}

type InMemoryRepo struct {
	mu              sync.RWMutex
	films           map[int64]Film
	filmIDs         []int64
	collections     []Collection
	collectionFilms map[int64][]int64
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		films:           make(map[int64]Film),
		collectionFilms: make(map[int64][]int64),
	}
}

func (r *InMemoryRepo) AddFilm(film Film) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pos, found := slices.BinarySearch(r.filmIDs, film.ID)
	if found {
		return fmt.Errorf("%w: id %d", ErrFilmExists, film.ID)
	}

	r.filmIDs = slices.Insert(r.filmIDs, pos, film.ID)
	r.films[film.ID] = cloneFilm(film)
	return nil
}

func (r *InMemoryRepo) AddCollection(collection Collection, filmIDs ...int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.collections {
		if existing.ID == collection.ID || existing.Slug == collection.Slug {
			return fmt.Errorf("%w: %q", ErrCollectionExists, collection.Slug)
		}
	}

	seen := make(map[int64]struct{}, len(filmIDs))
	for _, id := range filmIDs {
		if _, ok := r.films[id]; !ok {
			return fmt.Errorf("%w: id %d", ErrFilmNotFound, id)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%w: id %d", ErrDuplicateCollectionFilm, id)
		}
		seen[id] = struct{}{}
	}

	r.collections = append(r.collections, collection)
	r.collectionFilms[collection.ID] = slices.Clone(filmIDs)
	return nil
}

func (r *InMemoryRepo) ListFilms(_ context.Context, limit, offset int) ([]Film, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := len(r.filmIDs)
	start := min(max(offset, 0), total)
	end := min(start+max(limit, 0), total)

	result := make([]Film, 0, end-start)
	for _, id := range r.filmIDs[start:end] {
		result = append(result, cloneFilm(r.films[id]))
	}
	return result, total, nil
}

func (r *InMemoryRepo) ListCollections(_ context.Context) ([]Collection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append(make([]Collection, 0, len(r.collections)), r.collections...), nil
}

func (r *InMemoryRepo) GetCollectionBySlug(_ context.Context, slug string) (Collection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, collection := range r.collections {
		if collection.Slug == slug {
			return collection, nil
		}
	}
	return Collection{}, ErrCollectionNotFound
}

func (r *InMemoryRepo) ListCollectionFilms(_ context.Context, collectionID int64) ([]Film, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, ok := r.collectionFilms[collectionID]
	if !ok {
		return nil, ErrCollectionNotFound
	}

	result := make([]Film, 0, len(ids))
	for _, id := range ids {
		result = append(result, cloneFilm(r.films[id]))
	}
	return result, nil
}

func cloneFilm(film Film) Film {
	film.Genres = append(make([]Genre, 0, len(film.Genres)), film.Genres...)
	return film
}
