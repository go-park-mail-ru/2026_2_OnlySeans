package films

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

var (
	ErrFilmNotFound            = domain.ErrFilmNotFound
	ErrFilmExists              = domain.ErrFilmExists
	ErrCollectionNotFound      = domain.ErrCollectionNotFound
	ErrCollectionExists        = domain.ErrCollectionExists
	ErrDuplicateCollectionFilm = domain.ErrDuplicateCollectionFilm
)

type Service interface {
	ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error)
	GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error)
	ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, int, error)
	GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error)
	ListCollectionFilms(ctx context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error)
}

type InMemoryService struct {
	mu              sync.RWMutex
	films           map[domain.FilmID]domain.Film
	filmIDs         []domain.FilmID
	collections     []domain.Collection
	collectionFilms map[domain.CollectionID][]domain.FilmID
}

func NewInMemoryService() *InMemoryService {
	return &InMemoryService{
		films:           make(map[domain.FilmID]domain.Film),
		collectionFilms: make(map[domain.CollectionID][]domain.FilmID),
	}
}

func (s *InMemoryService) AddFilm(film domain.Film) error {
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

func (s *InMemoryService) AddCollection(collection domain.Collection, filmIDs ...domain.FilmID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.collections {
		if existing.ID == collection.ID || existing.Slug == collection.Slug {
			return fmt.Errorf("%w: %q", ErrCollectionExists, collection.Slug)
		}
	}

	seen := make(map[domain.FilmID]struct{}, len(filmIDs))
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

func (s *InMemoryService) ListFilms(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.filmsPage(s.filmIDs, limit, offset), len(s.filmIDs), nil
}

func (s *InMemoryService) GetFilmByID(_ context.Context, id domain.FilmID) (domain.Film, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	film, ok := s.films[id]
	if !ok {
		return domain.Film{}, ErrFilmNotFound
	}
	return cloneFilm(film), nil
}

func (s *InMemoryService) ListCollections(_ context.Context, limit, offset int) ([]domain.Collection, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.collections)
	start, end := pageBounds(total, limit, offset)

	return append(make([]domain.Collection, 0, end-start), s.collections[start:end]...), total, nil
}

func (s *InMemoryService) GetCollectionByID(_ context.Context, id domain.CollectionID) (domain.Collection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, collection := range s.collections {
		if collection.ID == id {
			return collection, nil
		}
	}
	return domain.Collection{}, ErrCollectionNotFound
}

func (s *InMemoryService) ListCollectionFilms(_ context.Context, collectionID domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.collectionFilms[collectionID]
	if !ok {
		return nil, 0, ErrCollectionNotFound
	}

	return s.filmsPage(ids, limit, offset), len(ids), nil
}

func (s *InMemoryService) filmsPage(ids []domain.FilmID, limit, offset int) []domain.Film {
	start, end := pageBounds(len(ids), limit, offset)

	result := make([]domain.Film, 0, end-start)
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

func cloneFilm(film domain.Film) domain.Film {
	film.Genres = append(make([]domain.Genre, 0, len(film.Genres)), film.Genres...)
	return film
}
