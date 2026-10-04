package films

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

var (
	ErrFilmNotFound            = errors.New("film not found")
	ErrFilmExists              = errors.New("film already exists")
	ErrCollectionNotFound      = errors.New("collection not found")
	ErrCollectionExists        = errors.New("collection already exists")
	ErrDuplicateCollectionFilm = errors.New("film is already in collection")
)

type Service interface {
	ListFilms(ctx context.Context, limit, offset int) ([]entities.Film, int, error)
	GetFilmByID(ctx context.Context, id entities.FilmID) (entities.Film, error)
	ListCollections(ctx context.Context, limit, offset int) ([]entities.Collection, int, error)
	GetCollectionByID(ctx context.Context, id entities.CollectionID) (entities.Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID entities.CollectionID, limit, offset int) ([]entities.Film, int, error)
}

type InMemoryService struct {
	mu              sync.RWMutex
	films           map[entities.FilmID]entities.Film
	filmIDs         []entities.FilmID
	collections     []entities.Collection
	collectionFilms map[entities.CollectionID][]entities.FilmID
}

func NewInMemoryService() *InMemoryService {
	return &InMemoryService{
		films:           make(map[entities.FilmID]entities.Film),
		collectionFilms: make(map[entities.CollectionID][]entities.FilmID),
	}
}

func (s *InMemoryService) AddFilm(film entities.Film) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pos, found := slices.BinarySearch(s.filmIDs, film.ID)
	if found {
		return fmt.Errorf("%w: id %d", ErrFilmExists, film.ID)
	}

	s.filmIDs = slices.Insert(s.filmIDs, pos, film.ID)
	s.films[film.ID] = cloneFilm(film)
	return nil
}

func (s *InMemoryService) AddCollection(collection entities.Collection, filmIDs ...entities.FilmID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.collections {
		if existing.ID == collection.ID || existing.Slug == collection.Slug {
			return fmt.Errorf("%w: %q", ErrCollectionExists, collection.Slug)
		}
	}

	seen := make(map[entities.FilmID]struct{}, len(filmIDs))
	for _, id := range filmIDs {
		if _, ok := s.films[id]; !ok {
			return fmt.Errorf("%w: id %d", ErrFilmNotFound, id)
		}
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%w: id %d", ErrDuplicateCollectionFilm, id)
		}
		seen[id] = struct{}{}
	}

	s.collections = append(s.collections, collection)
	s.collectionFilms[collection.ID] = slices.Clone(filmIDs)
	return nil
}

func (s *InMemoryService) ListFilms(_ context.Context, limit, offset int) ([]entities.Film, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.filmsPage(s.filmIDs, limit, offset), len(s.filmIDs), nil
}

func (s *InMemoryService) GetFilmByID(_ context.Context, id entities.FilmID) (entities.Film, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	film, ok := s.films[id]
	if !ok {
		return entities.Film{}, ErrFilmNotFound
	}
	return cloneFilm(film), nil
}

func (s *InMemoryService) ListCollections(_ context.Context, limit, offset int) ([]entities.Collection, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.collections)
	start, end := pageBounds(total, limit, offset)

	return append(make([]entities.Collection, 0, end-start), s.collections[start:end]...), total, nil
}

func (s *InMemoryService) GetCollectionByID(_ context.Context, id entities.CollectionID) (entities.Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, collection := range s.collections {
		if collection.ID == id {
			return collection, nil
		}
	}
	return entities.Collection{}, ErrCollectionNotFound
}

func (s *InMemoryService) ListCollectionFilms(_ context.Context, collectionID entities.CollectionID, limit, offset int) ([]entities.Film, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.collectionFilms[collectionID]
	if !ok {
		return nil, 0, ErrCollectionNotFound
	}

	return s.filmsPage(ids, limit, offset), len(ids), nil
}

func (s *InMemoryService) filmsPage(ids []entities.FilmID, limit, offset int) []entities.Film {
	start, end := pageBounds(len(ids), limit, offset)

	result := make([]entities.Film, 0, end-start)
	for _, id := range ids[start:end] {
		result = append(result, cloneFilm(s.films[id]))
	}
	return result
}

func pageBounds(total, limit, offset int) (start, end int) {
	start = min(max(offset, 0), total)
	end = start + min(max(limit, 0), total-start)
	return start, end
}

func cloneFilm(film entities.Film) entities.Film {
	film.Genres = append(make([]entities.Genre, 0, len(film.Genres)), film.Genres...)
	return film
}
