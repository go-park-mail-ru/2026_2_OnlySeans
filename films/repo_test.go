package films

import (
	"context"
	"errors"
	"slices"
	"testing"
	"unicode/utf8"
)

func newTestRepo(t *testing.T, filmCount int) *InMemoryRepo {
	t.Helper()

	repo := NewInMemoryRepo()
	for id := int64(filmCount); id >= 1; id-- {
		film := Film{ID: id, Title: "film", FilmType: FilmTypeMovie, ReleaseYear: 2000, Genres: []Genre{genreDrama}}
		if err := repo.AddFilm(film); err != nil {
			t.Fatalf("AddFilm(%d): %v", id, err)
		}
	}
	return repo
}

func filmIDs(films []Film) []int64 {
	ids := make([]int64, 0, len(films))
	for _, film := range films {
		ids = append(ids, film.ID)
	}
	return ids
}

func TestInMemoryRepo_AddFilm_Duplicate(t *testing.T) {
	repo := newTestRepo(t, 1)

	if err := repo.AddFilm(Film{ID: 1}); !errors.Is(err, ErrFilmExists) {
		t.Errorf("error = %v, want %v", err, ErrFilmExists)
	}
}

func TestInMemoryRepo_AddCollection(t *testing.T) {
	tests := []struct {
		name       string
		collection Collection
		filmIDs    []int64
		wantErr    error
	}{
		{"подборка создаётся", Collection{ID: 2, Slug: "new"}, []int64{1, 2}, nil},
		{"пустая подборка создаётся", Collection{ID: 2, Slug: "empty"}, nil, nil},
		{"повторный slug", Collection{ID: 2, Slug: "existing"}, []int64{1}, ErrCollectionExists},
		{"повторный id", Collection{ID: 1, Slug: "other"}, []int64{1}, ErrCollectionExists},
		{"несуществующий фильм", Collection{ID: 2, Slug: "other"}, []int64{1, 42}, ErrFilmNotFound},
		{"фильм дважды", Collection{ID: 2, Slug: "other"}, []int64{1, 2, 1}, ErrDuplicateCollectionFilm},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newTestRepo(t, 3)
			if err := repo.AddCollection(Collection{ID: 1, Slug: "existing"}); err != nil {
				t.Fatalf("AddCollection: %v", err)
			}

			err := repo.AddCollection(tt.collection, tt.filmIDs...)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryRepo_ListFilms(t *testing.T) {
	repo := newTestRepo(t, 5)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantIDs []int64
	}{
		{"первая страница", 2, 0, []int64{1, 2}},
		{"середина", 2, 2, []int64{3, 4}},
		{"хвост короче limit", 2, 4, []int64{5}},
		{"offset за пределами", 2, 10, []int64{}},
		{"отрицательный offset", 2, -3, []int64{1, 2}},
		{"отрицательный limit", -1, 0, []int64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			films, total, err := repo.ListFilms(context.Background(), tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("ListFilms: %v", err)
			}
			if total != 5 {
				t.Errorf("total = %d, ожидали 5", total)
			}
			if got := filmIDs(films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, ожидали %v", got, tt.wantIDs)
			}
		})
	}
}

func TestInMemoryRepo_ReturnsCopies(t *testing.T) {
	repo := newTestRepo(t, 1)
	ctx := context.Background()

	films, _, _ := repo.ListFilms(ctx, 1, 0)
	films[0].Genres[0].Name = "изменено"
	films[0].Title = "изменено"

	again, _, _ := repo.ListFilms(ctx, 1, 0)
	if again[0].Genres[0].Name != genreDrama.Name || again[0].Title != "film" {
		t.Errorf("изменение результата повлияло на хранилище: %+v", again[0])
	}
}

func TestInMemoryRepo_ListCollectionFilms(t *testing.T) {
	repo := newTestRepo(t, 3)
	ctx := context.Background()
	if err := repo.AddCollection(Collection{ID: 7, Slug: "ordered"}, 3, 1, 2); err != nil {
		t.Fatalf("AddCollection: %v", err)
	}

	films, err := repo.ListCollectionFilms(ctx, 7)
	if err != nil {
		t.Fatalf("ListCollectionFilms: %v", err)
	}
	if got, want := filmIDs(films), []int64{3, 1, 2}; !slices.Equal(got, want) {
		t.Errorf("ids = %v, ожидали %v", got, want)
	}

	if _, err := repo.ListCollectionFilms(ctx, 99); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("error = %v, want %v", err, ErrCollectionNotFound)
	}
}

func TestInMemoryRepo_GetCollectionBySlug(t *testing.T) {
	repo := newTestRepo(t, 1)
	ctx := context.Background()
	if err := repo.AddCollection(Collection{ID: 1, Slug: "found"}, 1); err != nil {
		t.Fatalf("AddCollection: %v", err)
	}

	collection, err := repo.GetCollectionBySlug(ctx, "found")
	if err != nil || collection.ID != 1 {
		t.Errorf("GetCollectionBySlug(found) = %+v, %v", collection, err)
	}
	if _, err := repo.GetCollectionBySlug(ctx, "missing"); !errors.Is(err, ErrCollectionNotFound) {
		t.Errorf("error = %v, want %v", err, ErrCollectionNotFound)
	}
}

func TestInMemoryRepo_ListCollections_Empty(t *testing.T) {
	collections, err := NewInMemoryRepo().ListCollections(context.Background())
	if err != nil || collections == nil || len(collections) != 0 {
		t.Errorf("ListCollections() = %v, %v; ожидали пустой не-nil срез", collections, err)
	}
}

func TestSeedMatchesDatabaseConstraints(t *testing.T) {
	ageLimits := []int16{0, 6, 12, 16, 18}

	for _, film := range seedFilms {
		titleLen := utf8.RuneCountInString(film.Title)
		switch {
		case titleLen < 1 || titleLen > 255:
			t.Errorf("фильм %d: длина title %d", film.ID, titleLen)
		case film.FilmType != FilmTypeMovie && film.FilmType != FilmTypeSeries:
			t.Errorf("фильм %d: film_type %q", film.ID, film.FilmType)
		case film.ReleaseYear < 1888 || film.ReleaseYear > 2200:
			t.Errorf("фильм %d: release_year %d", film.ID, film.ReleaseYear)
		case !slices.Contains(ageLimits, film.AgeLimit):
			t.Errorf("фильм %d: age_limit %d", film.ID, film.AgeLimit)
		case film.DurationMin != nil && (*film.DurationMin < 1 || *film.DurationMin > 6000):
			t.Errorf("фильм %d: duration_min %d", film.ID, *film.DurationMin)
		case len(film.Genres) == 0:
			t.Errorf("фильм %d: нет жанров", film.ID)
		}
	}

	for _, seed := range seedCollections {
		if !slugPattern.MatchString(seed.collection.Slug) {
			t.Errorf("подборка %d: slug %q", seed.collection.ID, seed.collection.Slug)
		}
		if titleLen := utf8.RuneCountInString(seed.collection.Title); titleLen < 3 || titleLen > 255 {
			t.Errorf("подборка %d: длина title %d", seed.collection.ID, titleLen)
		}
		if len(seed.filmIDs) == 0 {
			t.Errorf("подборка %d пустая", seed.collection.ID)
		}
	}

	if _, err := NewSeededRepo(); err != nil {
		t.Errorf("NewSeededRepo: %v", err)
	}
}
