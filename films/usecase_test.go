package films

import (
	"context"
	"errors"
	"slices"
	"testing"
)

var errStorage = errors.New("storage is down")

type brokenRepo struct {
	failCollectionFilms bool
}

func (b brokenRepo) ListFilms(context.Context, int, int) ([]Film, int, error) {
	return nil, 0, errStorage
}

func (b brokenRepo) ListCollections(context.Context) ([]Collection, error) {
	if b.failCollectionFilms {
		return []Collection{{ID: 1, Slug: "family"}}, nil
	}
	return nil, errStorage
}

func (b brokenRepo) GetCollectionBySlug(_ context.Context, slug string) (Collection, error) {
	if b.failCollectionFilms {
		return Collection{ID: 1, Slug: slug}, nil
	}
	return Collection{}, errStorage
}

func (b brokenRepo) ListCollectionFilms(context.Context, int64) ([]Film, error) {
	return nil, errStorage
}

func newSeededUseCase(t *testing.T) *UseCase {
	t.Helper()

	repo, err := NewSeededRepo()
	if err != nil {
		t.Fatalf("NewSeededRepo: %v", err)
	}
	return NewUseCase(repo)
}

func TestUseCase_ListFilms(t *testing.T) {
	uc := newSeededUseCase(t)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantErr error
		wantIDs []int64
	}{
		{"первая страница", 3, 0, nil, []int64{1, 2, 3}},
		{"последняя страница", 5, 10, nil, []int64{11, 12}},
		{"максимальный limit", MaxLimit, 0, nil, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{"limit = 0", 0, 0, ErrInvalidLimit, nil},
		{"limit больше максимума", MaxLimit + 1, 0, ErrInvalidLimit, nil},
		{"отрицательный offset", 5, -1, ErrInvalidOffset, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := uc.ListFilms(context.Background(), tt.limit, tt.offset)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got := filmIDs(page.Films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, ожидали %v", got, tt.wantIDs)
			}
			if page.Total != len(seedFilms) || page.Limit != tt.limit || page.Offset != tt.offset {
				t.Errorf("метаданные страницы = %+v", page)
			}
		})
	}
}

func TestUseCase_ListCollections(t *testing.T) {
	collections, err := newSeededUseCase(t).ListCollections(context.Background())
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(collections) != len(seedCollections) {
		t.Fatalf("получили %d подборок, ожидали %d", len(collections), len(seedCollections))
	}
	for i, collection := range collections {
		want := seedCollections[i]
		if collection.Slug != want.collection.Slug {
			t.Errorf("collections[%d].Slug = %q, ожидали %q", i, collection.Slug, want.collection.Slug)
		}
		if got := filmIDs(collection.Films); !slices.Equal(got, want.filmIDs) {
			t.Errorf("фильмы подборки %q = %v, ожидали %v", collection.Slug, got, want.filmIDs)
		}
	}
}

func TestUseCase_GetCollection(t *testing.T) {
	uc := newSeededUseCase(t)

	tests := []struct {
		name    string
		slug    string
		wantErr error
		wantIDs []int64
	}{
		{"существующая подборка", "family", nil, []int64{7, 8, 3, 9}},
		{"несуществующая подборка", "no-such-collection", ErrCollectionNotFound, nil},
		{"пустой slug", "", ErrInvalidSlug, nil},
		{"короткий slug", "ab", ErrInvalidSlug, nil},
		{"запрещённые символы", "Family_1", ErrInvalidSlug, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collection, err := uc.GetCollection(context.Background(), tt.slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if collection.Slug != tt.slug {
				t.Errorf("Slug = %q, ожидали %q", collection.Slug, tt.slug)
			}
			if got := filmIDs(collection.Films); !slices.Equal(got, tt.wantIDs) {
				t.Errorf("ids = %v, ожидали %v", got, tt.wantIDs)
			}
		})
	}
}

func TestUseCase_StorageErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		repo brokenRepo
		call func(*UseCase) error
	}{
		{"ListFilms", brokenRepo{}, func(uc *UseCase) error {
			_, err := uc.ListFilms(ctx, 10, 0)
			return err
		}},
		{"ListCollections", brokenRepo{}, func(uc *UseCase) error {
			_, err := uc.ListCollections(ctx)
			return err
		}},
		{"ListCollections: фильмы подборки", brokenRepo{failCollectionFilms: true}, func(uc *UseCase) error {
			_, err := uc.ListCollections(ctx)
			return err
		}},
		{"GetCollection", brokenRepo{}, func(uc *UseCase) error {
			_, err := uc.GetCollection(ctx, "family")
			return err
		}},
		{"GetCollection: фильмы подборки", brokenRepo{failCollectionFilms: true}, func(uc *UseCase) error {
			_, err := uc.GetCollection(ctx, "family")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(NewUseCase(tt.repo)); !errors.Is(err, errStorage) {
				t.Errorf("error = %v, ожидали обёрнутую %v", err, errStorage)
			}
		})
	}
}
