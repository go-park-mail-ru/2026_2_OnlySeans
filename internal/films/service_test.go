package films

import (
	"context"
	"errors"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"
)

func newTestService(t *testing.T, filmCount int) *InMemoryService {
	t.Helper()

	service := NewInMemoryService()
	for id := entities.FilmID(filmCount); id >= 1; id-- {
		film := entities.Film{
			ID:          id,
			Title:       "film",
			FilmType:    entities.FilmTypeMovie,
			ReleaseYear: 2000,
			Genres:      []entities.Genre{genreDrama},
		}
		if err := service.AddFilm(film); err != nil {
			t.Fatalf("AddFilm(%d): %v", id, err)
		}
	}
	return service
}

func filmIDs(films []entities.Film) []entities.FilmID {
	ids := make([]entities.FilmID, 0, len(films))
	for _, film := range films {
		ids = append(ids, film.ID)
	}
	return ids
}

func TestInMemoryService_AddFilm_Duplicate(t *testing.T) {
	service := newTestService(t, 1)

	if err := service.AddFilm(entities.Film{ID: 1}); !errors.Is(err, ErrFilmExists) {
		t.Errorf("error = %v, want %v", err, ErrFilmExists)
	}
}

func TestInMemoryService_AddCollection(t *testing.T) {
	tests := []struct {
		name       string
		collection entities.Collection
		filmIDs    []entities.FilmID
		wantErr    error
	}{
		{
			name: "collection is created",
			collection: entities.Collection{
				ID:   2,
				Slug: "new",
			},
			filmIDs: []entities.FilmID{1, 2},
		},
		{
			name: "empty collection is created",
			collection: entities.Collection{
				ID:   2,
				Slug: "empty",
			},
		},
		{
			name: "duplicate slug",
			collection: entities.Collection{
				ID:   2,
				Slug: "existing",
			},
			filmIDs: []entities.FilmID{1},
			wantErr: ErrCollectionExists,
		},
		{
			name: "duplicate id",
			collection: entities.Collection{
				ID:   1,
				Slug: "other",
			},
			filmIDs: []entities.FilmID{1},
			wantErr: ErrCollectionExists,
		},
		{
			name: "unknown film",
			collection: entities.Collection{
				ID:   2,
				Slug: "other",
			},
			filmIDs: []entities.FilmID{1, 42},
			wantErr: ErrFilmNotFound,
		},
		{
			name: "film added twice",
			collection: entities.Collection{
				ID:   2,
				Slug: "other",
			},
			filmIDs: []entities.FilmID{1, 2, 1},
			wantErr: ErrDuplicateCollectionFilm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newTestService(t, 3)
			existing := entities.Collection{
				ID:   1,
				Slug: "existing",
			}
			if err := service.AddCollection(existing); err != nil {
				t.Fatalf("AddCollection: %v", err)
			}

			err := service.AddCollection(tt.collection, tt.filmIDs...)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryService_ListFilms(t *testing.T) {
	service := newTestService(t, 5)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantIDs []entities.FilmID
	}{
		{
			name:    "first page",
			limit:   2,
			offset:  0,
			wantIDs: []entities.FilmID{1, 2},
		},
		{
			name:    "middle page",
			limit:   2,
			offset:  2,
			wantIDs: []entities.FilmID{3, 4},
		},
		{
			name:    "tail shorter than limit",
			limit:   2,
			offset:  4,
			wantIDs: []entities.FilmID{5},
		},
		{
			name:    "offset out of range",
			limit:   2,
			offset:  10,
			wantIDs: []entities.FilmID{},
		},
		{
			name:    "negative offset",
			limit:   2,
			offset:  -3,
			wantIDs: []entities.FilmID{1, 2},
		},
		{
			name:    "negative limit",
			limit:   -1,
			offset:  0,
			wantIDs: []entities.FilmID{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			films, total, err := service.ListFilms(context.Background(), tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("ListFilms: %v", err)
			}
			if total != 5 {
				t.Errorf("total = %d, want 5", total)
			}
			if got := filmIDs(films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", got, tt.wantIDs)
			}
		})
	}
}

func TestInMemoryService_ReturnsCopies(t *testing.T) {
	service := newTestService(t, 1)
	ctx := context.Background()

	films, _, _ := service.ListFilms(ctx, 1, 0)
	films[0].Genres[0].Name = "changed"
	films[0].Title = "changed"

	again, _, _ := service.ListFilms(ctx, 1, 0)
	if again[0].Genres[0].Name != genreDrama.Name || again[0].Title != "film" {
		t.Errorf("mutating result changed stored film: %+v", again[0])
	}
}

func TestInMemoryService_ListCollectionFilms(t *testing.T) {
	service := newTestService(t, 3)
	ctx := context.Background()
	ordered := entities.Collection{
		ID:   7,
		Slug: "ordered",
	}
	if err := service.AddCollection(ordered, 3, 1, 2); err != nil {
		t.Fatalf("AddCollection: %v", err)
	}

	films, err := service.ListCollectionFilms(ctx, 7)
	if err != nil {
		t.Fatalf("ListCollectionFilms: %v", err)
	}
	if got, want := filmIDs(films), []entities.FilmID{3, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}

	if _, err := service.ListCollectionFilms(ctx, 99); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("error = %v, want %v", err, ErrCollectionNotFound)
	}
}

func TestInMemoryService_GetCollectionBySlug(t *testing.T) {
	service := newTestService(t, 1)
	ctx := context.Background()
	found := entities.Collection{
		ID:   1,
		Slug: "found",
	}
	if err := service.AddCollection(found, 1); err != nil {
		t.Fatalf("AddCollection: %v", err)
	}

	collection, err := service.GetCollectionBySlug(ctx, "found")
	if err != nil || collection.ID != 1 {
		t.Errorf("GetCollectionBySlug(found) = %+v, %v", collection, err)
	}
	if _, err := service.GetCollectionBySlug(ctx, "missing"); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("error = %v, want %v", err, ErrCollectionNotFound)
	}
}

func TestInMemoryService_ListCollections_Empty(t *testing.T) {
	collections, err := NewInMemoryService().ListCollections(context.Background())
	if err != nil || collections == nil || len(collections) != 0 {
		t.Errorf("ListCollections() = %v, %v; want empty non-nil slice", collections, err)
	}
}

func TestSeedMatchesDatabaseConstraints(t *testing.T) {
	ageLimits := []int16{0, 6, 12, 16, 18}

	for _, film := range seedFilms {
		titleLen := utf8.RuneCountInString(film.Title)
		switch {
		case titleLen < 1 || titleLen > 255:
			t.Errorf("film %d: title length %d", film.ID, titleLen)
		case film.FilmType != entities.FilmTypeMovie && film.FilmType != entities.FilmTypeSeries:
			t.Errorf("film %d: film_type %q", film.ID, film.FilmType)
		case film.ReleaseYear < 1888 || film.ReleaseYear > 2200:
			t.Errorf("film %d: release_year %d", film.ID, film.ReleaseYear)
		case !slices.Contains(ageLimits, film.AgeLimit):
			t.Errorf("film %d: age_limit %d", film.ID, film.AgeLimit)
		case film.DurationMin != nil && (*film.DurationMin < 1 || *film.DurationMin > 6000):
			t.Errorf("film %d: duration_min %d", film.ID, *film.DurationMin)
		case len(film.Genres) == 0:
			t.Errorf("film %d: no genres", film.ID)
		}
	}

	for _, seed := range seedCollections {
		if !slugPattern.MatchString(seed.collection.Slug) {
			t.Errorf("collection %d: slug %q", seed.collection.ID, seed.collection.Slug)
		}
		if titleLen := utf8.RuneCountInString(seed.collection.Title); titleLen < 3 || titleLen > 255 {
			t.Errorf("collection %d: title length %d", seed.collection.ID, titleLen)
		}
		if len(seed.filmIDs) == 0 {
			t.Errorf("collection %d is empty", seed.collection.ID)
		}
	}

	if _, err := NewSeededService(); err != nil {
		t.Errorf("NewSeededService: %v", err)
	}
}
